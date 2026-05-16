package codex

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

const (
	defaultBaseURL       = "https://chatgpt.com/backend-api/codex"
	defaultUserAgent     = "codex-cli/1.0.18 (macOS; arm64)"
	sessionTTL           = time.Hour
	sessionCleanupPeriod = 10 * time.Minute
)

type Client struct {
	httpClient *http.Client
	connection config.ConnectionConfig
	baseURL    string
	sessions   *sessionStore
	tokenMu    sync.Mutex
}

func NewClient(connection config.ConnectionConfig) *Client {
	return NewClientWithHTTPClient(nil, connection)
}

func NewClientWithHTTPClient(httpClient *http.Client, connection config.ConnectionConfig) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		httpClient: httpClient,
		connection: connection,
		baseURL:    defaultBaseURL,
		sessions:   newSessionStore(),
	}
}

func (c *Client) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	body := req
	body.Stream = false

	respBody, err := c.doResponses(ctx, body, target)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}
	defer respBody.Close()

	var upstream codexResponse
	if err := json.NewDecoder(respBody).Decode(&upstream); err != nil {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("decode codex response: %w", err)
	}

	return openaiwire.ChatCompletionsResponse{
		ID:     upstream.ID,
		Object: "chat.completion",
		Model:  target.RequestedModel,
		Choices: []openaiwire.ChatChoice{{
			Index: 0,
			Message: openaiwire.ChatMessage{
				Role:    "assistant",
				Content: upstream.outputText(),
			},
		}},
	}, nil
}

func (c *Client) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	body := req
	body.Stream = true
	return c.doResponses(ctx, body, target)
}

func (c *Client) doResponses(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	credential, err := c.resolveAccessToken(false)
	if err != nil {
		return nil, chatcompletion.ConnectionConfigurationError{
			ConnectionID:   c.connection.ID,
			ConnectionName: c.connection.Name,
			Message:        err.Error(),
		}
	}

	machineID := machineID()
	sessionID := c.sessions.resolve(req.Messages, machineID)
	upstreamRequest := chatCompletionsToCodexResponses(req, target.RequestedModel)

	payload, err := json.Marshal(upstreamRequest)
	if err != nil {
		return nil, fmt.Errorf("encode codex request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build codex request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+credential)
	httpReq.Header.Set("originator", "codex-cli")
	httpReq.Header.Set("User-Agent", defaultUserAgent)
	httpReq.Header.Set("session_id", sessionID)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute codex request: %w", err)
	}

	if shouldRetryWithTokenRefresh(resp.StatusCode, c.connection) {
		resp.Body.Close()

		credential, err = c.resolveAccessToken(true)
		if err != nil {
			return nil, chatcompletion.ConnectionConfigurationError{
				ConnectionID:   c.connection.ID,
				ConnectionName: c.connection.Name,
				Message:        err.Error(),
			}
		}

		httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/responses", bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("build codex retry request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Authorization", "Bearer "+credential)
		httpReq.Header.Set("originator", "codex-cli")
		httpReq.Header.Set("User-Agent", defaultUserAgent)
		httpReq.Header.Set("session_id", sessionID)

		resp, err = c.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("execute codex retry request: %w", err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	return resp.Body, nil
}

func (c *Client) resolveAccessToken(forceRefresh bool) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	connection := c.connection
	if forceRefresh {
		var err error
		connection, err = refreshConnectionToken(connection, true)
		if err != nil {
			return "", err
		}
		c.connection = connection
		return GetAccessToken(connection)
	}

	if strings.TrimSpace(connection.APIKey) != "" && strings.TrimSpace(connection.AccessToken) == "" {
		return strings.TrimSpace(connection.APIKey), nil
	}

	var err error
	connection, err = refreshConnectionToken(connection, false)
	if err != nil {
		return "", err
	}
	c.connection = connection

	token, err := GetAccessToken(connection)
	if err == nil {
		return token, nil
	}
	if strings.TrimSpace(connection.APIKey) != "" {
		return strings.TrimSpace(connection.APIKey), nil
	}

	return "", err
}

func shouldRetryWithTokenRefresh(statusCode int, connection config.ConnectionConfig) bool {
	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return false
	}

	return strings.TrimSpace(connection.RefreshToken) != ""
}

type codexResponsesRequest struct {
	Model        string                 `json:"model"`
	Instructions string                 `json:"instructions,omitempty"`
	Input        []codexInputItem       `json:"input"`
	Stream       bool                   `json:"stream"`
	Store        bool                   `json:"store"`
	Reasoning    any                    `json:"reasoning,omitempty"`
	Include      []string               `json:"include,omitempty"`
	Extra        map[string]interface{} `json:"-"`
}

type codexInputItem struct {
	Type      string                 `json:"type"`
	Role      string                 `json:"role,omitempty"`
	Content   []codexContentPart     `json:"content,omitempty"`
	CallID    string                 `json:"call_id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Arguments string                 `json:"arguments,omitempty"`
	Output    string                 `json:"output,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

type codexContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type reasoningConfig struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

func chatCompletionsToCodexResponses(req openaiwire.ChatCompletionsRequest, model string) codexResponsesRequest {
	var instructions string
	input := make([]codexInputItem, 0, len(req.Messages))

	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			if instructions == "" {
				instructions = extractMessageText(msg)
			}
		case "user", "assistant":
			input = append(input, codexInputItem{
				Type:    "message",
				Role:    msg.Role,
				Content: contentToResponsesContent(msg.Role, msg.Content),
			})
			if msg.Role == "assistant" {
				for _, toolCall := range msg.ToolCalls {
					input = append(input, codexInputItem{
						Type:      "function_call",
						CallID:    toolCall.ID,
						Name:      toolCall.Function.Name,
						Arguments: defaultString(toolCall.Function.Arguments, "{}"),
					})
				}
			}
		case "tool":
			input = append(input, codexInputItem{
				Type:   "function_call_output",
				CallID: msg.ToolCallID,
				Output: stringifyContent(msg.Content),
			})
		}
	}

	if len(input) == 0 {
		input = append(input, codexInputItem{
			Type:    "message",
			Role:    "user",
			Content: []codexContentPart{{Type: "input_text", Text: "..."}},
		})
	}

	reasoning := any(reasoningConfig{Effort: defaultString(req.ReasoningEffort, "low"), Summary: "auto"})
	if len(req.Reasoning) > 0 {
		var raw any
		if err := json.Unmarshal(req.Reasoning, &raw); err == nil {
			reasoning = raw
		}
	}

	out := codexResponsesRequest{
		Model:        stripProviderPrefix(model),
		Instructions: instructions,
		Input:        input,
		Stream:       req.Stream,
		Store:        false,
		Reasoning:    reasoning,
	}
	if reasoningEffort(reasoning) != "none" {
		out.Include = []string{"reasoning.encrypted_content"}
	}

	return out
}

func stripProviderPrefix(model string) string {
	if strings.HasPrefix(model, "cx/") {
		return strings.TrimPrefix(model, "cx/")
	}
	if strings.HasPrefix(model, "codex/") {
		return strings.TrimPrefix(model, "codex/")
	}
	return model
}

func contentToResponsesContent(role string, content any) []codexContentPart {
	textType := "input_text"
	if role == "assistant" {
		textType = "output_text"
	}

	switch v := content.(type) {
	case string:
		return []codexContentPart{{Type: textType, Text: v}}
	case []any:
		parts := make([]codexContentPart, 0, len(v))
		for _, part := range v {
			partMap, ok := part.(map[string]any)
			if !ok {
				parts = append(parts, codexContentPart{Type: textType, Text: stringifyContent(part)})
				continue
			}
			switch partMap["type"] {
			case "text":
				parts = append(parts, codexContentPart{Type: textType, Text: stringField(partMap, "text")})
			case "image_url":
				imageURL, detail := imageURLFields(partMap["image_url"])
				parts = append(parts, codexContentPart{Type: "input_image", ImageURL: imageURL, Detail: defaultString(detail, "auto")})
			default:
				parts = append(parts, codexContentPart{Type: textType, Text: stringifyContent(part)})
			}
		}
		return parts
	default:
		return []codexContentPart{{Type: textType, Text: stringifyContent(content)}}
	}
}

func imageURLFields(value any) (string, string) {
	switch v := value.(type) {
	case string:
		return v, ""
	case map[string]any:
		return stringField(v, "url"), stringField(v, "detail")
	default:
		return "", ""
	}
}

func stringField(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func stringifyContent(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(data)
	}
}

func extractMessageText(message openaiwire.ChatMessage) string {
	return strings.Join(extractTextParts(message.Content), "")
}

func extractTextParts(content any) []string {
	switch v := content.(type) {
	case nil:
		return nil
	case string:
		return []string{v}
	case []any:
		parts := make([]string, 0, len(v))
		for _, part := range v {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if text := stringField(partMap, "text"); text != "" {
				parts = append(parts, text)
			} else if output := stringField(partMap, "output"); output != "" {
				parts = append(parts, output)
			}
		}
		return parts
	default:
		return []string{stringifyContent(content)}
	}
}

func reasoningEffort(reasoning any) string {
	switch v := reasoning.(type) {
	case reasoningConfig:
		return v.Effort
	case map[string]any:
		effort, _ := v["effort"].(string)
		return effort
	default:
		return ""
	}
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func machineID() string {
	// 1. Explicit override
	if v := clean(os.Getenv("MACHINE_ID")); v != "" {
		return v
	}

	// 2. Stable OS hostname
	if host, err := os.Hostname(); err == nil {
		if host = clean(host); host != "" {
			return hash(host)
		}
	}

	// 3. Container / k8s fallback
	for _, key := range []string{
		"HOSTNAME",
		"POD_NAME",
		"CONTAINER_NAME",
	} {
		if v := clean(os.Getenv(key)); v != "" {
			return hash(v)
		}
	}

	// 4. Last fallback
	return "default-machine"
}

func clean(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

func hash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:8]) // 16 chars
}

type sessionEntry struct {
	sessionID string
	lastUsed  time.Time
}

type sessionStore struct {
	mu      sync.Mutex
	entries map[string]sessionEntry
	now     func() time.Time
}

func newSessionStore() *sessionStore {
	store := &sessionStore{
		entries: make(map[string]sessionEntry),
		now:     time.Now,
	}
	go store.cleanupLoop()
	return store
}

func (s *sessionStore) resolve(messages []openaiwire.ChatMessage, machineID string) string {
	machineSessionID := generateSessionID()
	if machineID != "" {
		machineSessionID = "sess_" + hashContent(machineID)
	}

	var firstAssistantText string
	for _, msg := range messages {
		if msg.Role == "assistant" {
			firstAssistantText = extractMessageText(msg)
			break
		}
	}
	if firstAssistantText == "" {
		return machineSessionID
	}

	key := hashContent(machineID + firstAssistantText)
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.entries[key]; ok {
		existing.lastUsed = s.now()
		s.entries[key] = existing
		return existing.sessionID
	}

	sessionID := generateSessionID()
	s.entries[key] = sessionEntry{sessionID: sessionID, lastUsed: s.now()}
	return sessionID
}

func (s *sessionStore) cleanupLoop() {
	ticker := time.NewTicker(sessionCleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		s.expire()
	}
}

func (s *sessionStore) expire() {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, entry := range s.entries {
		if now.Sub(entry.lastUsed) > sessionTTL {
			delete(s.entries, key)
		}
	}
}

func hashContent(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])[:16]
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%s_%s", strconv.FormatInt(time.Now().UnixMilli(), 36), randomBase36(7))
}

func randomBase36(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	var builder strings.Builder
	for i := 0; i < length; i++ {
		index, err := crand.Int(crand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			builder.WriteByte(alphabet[time.Now().UnixNano()%int64(len(alphabet))])
			continue
		}
		builder.WriteByte(alphabet[index.Int64()])
	}
	return builder.String()
}

type codexResponse struct {
	ID     string            `json:"id"`
	Output []codexOutputItem `json:"output"`
}

type codexOutputItem struct {
	Content []codexContentPart `json:"content"`
}

func (r codexResponse) outputText() string {
	var builder strings.Builder
	for _, item := range r.Output {
		for _, part := range item.Content {
			if part.Text != "" {
				builder.WriteString(part.Text)
			}
		}
	}
	return builder.String()
}

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

const defaultBaseURL = "https://api.openai.com"

type Client struct {
	httpClient *http.Client
	provider   config.ProviderConfig
}

func NewClient(httpClient *http.Client, provider config.ProviderConfig) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{httpClient: httpClient, provider: provider}
}

func (c *Client) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	if target.ProviderType != c.provider.Type {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("openai-compatible executor cannot handle provider type %q", target.ProviderType)
	}

	credential, err := c.credential()
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, chatcompletion.ProviderConfigurationError{
			ProviderID:   c.provider.ID,
			ProviderName: c.provider.Name,
			Message:      "missing api_key or access_token",
		}
	}

	upstreamRequest := req
	upstreamRequest.Model = target.RequestedModel

	httpReq, err := c.newChatCompletionsRequest(ctx, upstreamRequest, credential)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("execute upstream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return openaiwire.ChatCompletionsResponse{}, chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	var out openaiwire.ChatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("decode upstream response: %w", err)
	}

	return out, nil
}

func (c *Client) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	if target.ProviderType != c.provider.Type {
		return nil, fmt.Errorf("openai-compatible executor cannot handle provider type %q", target.ProviderType)
	}

	credential, err := c.credential()
	if err != nil {
		return nil, chatcompletion.ProviderConfigurationError{
			ProviderID:   c.provider.ID,
			ProviderName: c.provider.Name,
			Message:      "missing api_key or access_token",
		}
	}

	upstreamRequest := req
	upstreamRequest.Model = target.RequestedModel
	upstreamRequest.Stream = true

	httpReq, err := c.newChatCompletionsRequest(ctx, upstreamRequest, credential)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute upstream request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	return resp.Body, nil
}

func (c *Client) credential() (string, error) {
	credential := strings.TrimSpace(c.provider.APIKey)
	if credential == "" {
		credential = strings.TrimSpace(c.provider.AccessToken)
	}
	if credential == "" {
		return "", fmt.Errorf("missing credential")
	}

	return credential, nil
}

func (c *Client) newChatCompletionsRequest(ctx context.Context, req openaiwire.ChatCompletionsRequest, credential string) (*http.Request, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode upstream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultBaseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+credential)

	return httpReq, nil
}

package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	adapterhttpclient "github.com/phamtanminhtien/goroute/internal/adapter/httpclient"
	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/protocoltranslator"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

const (
	defaultBaseURL          = "https://api.anthropic.com"
	defaultAnthropicVersion = "2023-06-01"
)

type Client struct {
	httpClient *http.Client
	connection connection.Record
	baseURL    string
}

func NewClient(httpClient *http.Client, connection connection.Record) *Client {
	return NewClientWithBaseURL(httpClient, connection, defaultBaseURL)
}

func NewClientWithBaseURL(httpClient *http.Client, connection connection.Record, baseURL string) *Client {
	if httpClient == nil {
		httpClient = adapterhttpclient.NewStreamingClient()
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{httpClient: httpClient, connection: connection, baseURL: baseURL}
}

func NewClientWithRuntimeSettings(connection connection.Record, settings config.ProviderRuntimeSettings) *Client {
	return NewClientWithBaseURL(adapterhttpclient.NewStreamingClientWithSettings(settings), connection, "")
}

func (c *Client) AnthropicMessages(ctx context.Context, req anthropicwire.MessagesRequest, target routing.Target) (anthropicwire.MessagesResponse, error) {
	credential, err := c.credential()
	if err != nil {
		return anthropicwire.MessagesResponse{}, c.configurationError()
	}
	payload, err := c.marshalRequest(req, target.RequestedModel, nil)
	if err != nil {
		return anthropicwire.MessagesResponse{}, fmt.Errorf("encode upstream request: %w", err)
	}
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetProviderRequestMode(false)
		recorder.SetTranslatedRequestBody(string(payload))
	}
	attemptIndex := chatcompletion.AttemptIndex(ctx)
	httpReq, err := c.newMessagesRequest(ctx, payload, credential)
	if err != nil {
		return anthropicwire.MessagesResponse{}, err
	}

	startedAt := time.Now().UTC()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, nil, nil, startedAt, time.Now().UTC(), err, attemptIndex)
		return anthropicwire.MessagesResponse{}, fmt.Errorf("execute upstream request: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	completedAt := time.Now().UTC()
	if readErr != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, nil, startedAt, completedAt, readErr, attemptIndex)
		return anthropicwire.MessagesResponse{}, fmt.Errorf("read upstream response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		upstreamErr := chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, upstreamErr, attemptIndex)
		return anthropicwire.MessagesResponse{}, upstreamErr
	}
	var out anthropicwire.MessagesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, err, attemptIndex)
		return anthropicwire.MessagesResponse{}, fmt.Errorf("decode upstream response: %w", err)
	}
	c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, nil, attemptIndex)
	return out, nil
}

func (c *Client) AnthropicMessagesStream(ctx context.Context, req anthropicwire.MessagesRequest, target routing.Target) (io.ReadCloser, error) {
	credential, err := c.credential()
	if err != nil {
		return nil, c.configurationError()
	}
	stream := true
	payload, err := c.marshalRequest(req, target.RequestedModel, &stream)
	if err != nil {
		return nil, fmt.Errorf("encode upstream request: %w", err)
	}
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetProviderRequestMode(true)
		recorder.SetTranslatedRequestBody(string(payload))
	}
	attemptIndex := chatcompletion.AttemptIndex(ctx)
	httpReq, err := c.newMessagesRequest(ctx, payload, credential)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	startedAt := time.Now().UTC()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, nil, nil, startedAt, time.Now().UTC(), err, attemptIndex)
		return nil, fmt.Errorf("execute upstream request: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		upstreamErr := chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, time.Now().UTC(), upstreamErr, attemptIndex)
		return nil, upstreamErr
	}

	return chatcompletion.CaptureStream(resp.Body, func(streamBody []byte, streamErr error) {
		completedAt := time.Now().UTC()
		if reconstructed, parseErr := protocoltranslator.ParseAnthropicSSE(streamBody); parseErr == nil {
			if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
				recorder.SetResponsesResponse(protocoltranslator.AnthropicToResponsesResponse(reconstructed), true)
			}
		}
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, streamBody, startedAt, completedAt, streamErr, attemptIndex)
	}), nil
}

func (c *Client) Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	translated, err := protocoltranslator.ResponsesToAnthropicRequest(req)
	if err != nil {
		return openaiwire.ResponsesResponse{}, err
	}
	response, err := c.AnthropicMessages(ctx, translated, target)
	if err != nil {
		return openaiwire.ResponsesResponse{}, err
	}
	return protocoltranslator.AnthropicToResponsesResponse(response), nil
}

func (c *Client) ResponsesStream(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error) {
	translated, err := protocoltranslator.ResponsesToAnthropicRequest(req)
	if err != nil {
		return nil, err
	}
	translated.Stream = true
	body, err := c.AnthropicMessagesStream(ctx, translated, target)
	if err != nil {
		return nil, err
	}
	return protocoltranslator.AnthropicStreamToResponses(body), nil
}

func (c *Client) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	translatedResponses, err := protocoltranslator.ChatCompletionsToResponses(req)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}
	response, err := c.Responses(ctx, translatedResponses, target)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}
	return protocoltranslator.ResponsesToChatCompletion(response), nil
}

func (c *Client) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	translatedResponses, err := protocoltranslator.ChatCompletionsToResponses(req)
	if err != nil {
		return nil, err
	}
	translatedResponses.Stream = true
	body, err := c.ResponsesStream(ctx, translatedResponses, target)
	if err != nil {
		return nil, err
	}
	return protocoltranslator.ResponsesStreamToChatCompletions(body), nil
}

func (c *Client) credential() (string, error) {
	credential := strings.TrimSpace(c.connection.APIKey)
	if credential == "" {
		return "", fmt.Errorf("missing credential")
	}
	return credential, nil
}

func (c *Client) configurationError() error {
	return chatcompletion.ConnectionConfigurationError{
		ConnectionID:   c.connection.ID,
		ConnectionName: c.connection.Name,
		Message:        "missing api_key",
	}
}

func (c *Client) marshalRequest(req anthropicwire.MessagesRequest, model string, forceStream *bool) ([]byte, error) {
	if len(req.RawBody) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(req.RawBody, &payload); err != nil {
			return nil, err
		}
		payload["model"] = model
		if forceStream != nil {
			payload["stream"] = *forceStream
		}
		return json.Marshal(payload)
	}
	upstreamRequest := req
	upstreamRequest.Model = model
	if forceStream != nil {
		upstreamRequest.Stream = *forceStream
	}
	upstreamRequest.RawBody = nil
	return json.Marshal(upstreamRequest)
}

func (c *Client) newMessagesRequest(ctx context.Context, payload []byte, credential string) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", credential)
	httpReq.Header.Set("anthropic-version", defaultAnthropicVersion)
	return httpReq, nil
}

func (c *Client) recordThirdPartyLog(ctx context.Context, target routing.Target, requestBody []byte, request *http.Request, response *http.Response, responseBody []byte, startedAt time.Time, completedAt time.Time, err error, attemptIndex int) {
	recorder := chatcompletion.FlowRecorderFromContext(ctx)
	if recorder == nil || request == nil {
		return
	}
	logRecord := chatcompletion.ThirdPartyLog{
		ProviderID:          target.ProviderID,
		ProviderName:        target.ProviderName,
		ConnectionID:        c.connection.ID,
		ConnectionName:      c.connection.Name,
		AttemptIndex:        attemptIndex,
		RequestMethod:       request.Method,
		RequestURL:          request.URL.String(),
		ProviderRequestMode: defaultProviderRequestMode(request),
		RequestHeaders:      chatcompletion.RedactHeadersForStorage(request.Header),
		RequestBody:         chatcompletion.RedactBodyForStorage(string(requestBody)),
		ResponseBody:        chatcompletion.ThirdPartyResponseBodyForStorage(responseHeader(response), string(responseBody)),
		StartedAt:           startedAt,
		CompletedAt:         completedAt,
	}
	if response != nil {
		logRecord.ResponseStatusCode = response.StatusCode
		logRecord.ResponseHeaders = chatcompletion.RedactHeadersForStorage(response.Header)
	}
	if err != nil {
		logRecord.ErrorType = thirdPartyErrorType(err)
		logRecord.ErrorMessage = err.Error()
	}
	recorder.AddThirdPartyLog(logRecord)
}

func thirdPartyErrorType(err error) string {
	if err == nil {
		return ""
	}
	var upstreamErr chatcompletion.UpstreamError
	if errors.As(err, &upstreamErr) {
		return "upstream_error"
	}
	return "request_error"
}

func responseHeader(response *http.Response) http.Header {
	if response == nil {
		return nil
	}
	return response.Header
}

func defaultProviderRequestMode(request *http.Request) string {
	if request == nil {
		return ""
	}
	if strings.Contains(strings.ToLower(request.Header.Get("Accept")), "text/event-stream") {
		return chatcompletion.RequestModeStream
	}
	return chatcompletion.RequestModeSync
}

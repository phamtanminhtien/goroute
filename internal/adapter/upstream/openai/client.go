package openai

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

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

const defaultBaseURL = "https://api.openai.com"

type Client struct {
	httpClient *http.Client
	connection connection.Record
}

func NewClient(httpClient *http.Client, connection connection.Record) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{httpClient: httpClient, connection: connection}
}

func (c *Client) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	credential, err := c.credential()
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, chatcompletion.ConnectionConfigurationError{
			ConnectionID:   c.connection.ID,
			ConnectionName: c.connection.Name,
			Message:        "missing api_key or access_token",
		}
	}

	upstreamRequest := req
	upstreamRequest.Model = target.RequestedModel
	payload, err := json.Marshal(upstreamRequest)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("encode upstream request: %w", err)
	}

	httpReq, err := c.newChatCompletionsRequest(ctx, payload, credential)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}

	startedAt := time.Now().UTC()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, nil, nil, startedAt, time.Now().UTC(), err, 0)
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("execute upstream request: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	completedAt := time.Now().UTC()
	if readErr != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, nil, startedAt, completedAt, readErr, 0)
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("read upstream response: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}, 0)
		return openaiwire.ChatCompletionsResponse{}, chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	var out openaiwire.ChatCompletionsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, err, 0)
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("decode upstream response: %w", err)
	}
	c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, nil, 0)

	return out, nil
}

func (c *Client) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	credential, err := c.credential()
	if err != nil {
		return nil, chatcompletion.ConnectionConfigurationError{
			ConnectionID:   c.connection.ID,
			ConnectionName: c.connection.Name,
			Message:        "missing api_key or access_token",
		}
	}

	upstreamRequest := req
	upstreamRequest.Model = target.RequestedModel
	upstreamRequest.Stream = true
	payload, err := json.Marshal(upstreamRequest)
	if err != nil {
		return nil, fmt.Errorf("encode upstream request: %w", err)
	}

	httpReq, err := c.newChatCompletionsRequest(ctx, payload, credential)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	startedAt := time.Now().UTC()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, nil, nil, startedAt, time.Now().UTC(), err, 0)
		return nil, fmt.Errorf("execute upstream request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, time.Now().UTC(), chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}, 0)
		return nil, chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	return chatcompletion.CaptureStream(resp.Body, func(streamBody []byte, streamErr error) {
		completedAt := time.Now().UTC()
		responseBody := ""
		if reconstructed, ok := chatcompletion.ReconstructOpenAIResponseFromSSE(streamBody, target.RequestedModel); ok {
			if payload, err := json.Marshal(reconstructed); err == nil {
				responseBody = string(payload)
			}
			if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
				recorder.SetFlowResponse(reconstructed, true)
			}
		}
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, []byte(responseBody), startedAt, completedAt, streamErr, 0)
	}), nil
}

func (c *Client) credential() (string, error) {
	credential := strings.TrimSpace(c.connection.APIKey)
	if credential == "" {
		credential = strings.TrimSpace(c.connection.AccessToken)
	}
	if credential == "" {
		return "", fmt.Errorf("missing credential")
	}

	return credential, nil
}

func (c *Client) newChatCompletionsRequest(ctx context.Context, payload []byte, credential string) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultBaseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+credential)

	return httpReq, nil
}

func (c *Client) recordThirdPartyLog(ctx context.Context, target routing.Target, requestBody []byte, request *http.Request, response *http.Response, responseBody []byte, startedAt time.Time, completedAt time.Time, err error, attemptIndex int) {
	recorder := chatcompletion.FlowRecorderFromContext(ctx)
	if recorder == nil || request == nil {
		return
	}

	logRecord := chatcompletion.ThirdPartyLog{
		ProviderID:     target.ProviderID,
		ProviderName:   target.ProviderName,
		ConnectionID:   c.connection.ID,
		ConnectionName: c.connection.Name,
		AttemptIndex:   attemptIndex,
		RequestMethod:  request.Method,
		RequestURL:     request.URL.String(),
		RequestHeaders: chatcompletion.RedactHeadersForStorage(request.Header),
		RequestBody:    chatcompletion.RedactBodyForStorage(string(requestBody)),
		ResponseBody:   chatcompletion.RedactBodyForStorage(string(responseBody)),
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
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

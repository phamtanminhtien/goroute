package codex

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

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/rtk"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	responsesusecase "github.com/phamtanminhtien/goroute/internal/usecase/responses"
)

var forwardedInboundResponseHeaders = map[string]struct{}{
	"accept":                {},
	"content-type":          {},
	"originator":            {},
	"session-id":            {},
	"session_id":            {},
	"thread-id":             {},
	"thread_id":             {},
	"user-agent":            {},
	"x-client-request-id":   {},
	"x-codex-beta-features": {},
	"x-codex-turn-metadata": {},
	"x-codex-window-id":     {},
}

func (c *Client) Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	resp, payload, httpReq, startedAt, attemptIndex, err := c.doResponsesRequest(ctx, req, target)
	if err != nil {
		return openaiwire.ResponsesResponse{}, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	completedAt := time.Now().UTC()
	if readErr != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, nil, startedAt, completedAt, readErr, attemptIndex)
		return openaiwire.ResponsesResponse{}, fmt.Errorf("read upstream response: %w", readErr)
	}

	reconstructed, parseErr := responsesusecase.ParseSSE(body)
	if parseErr != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, parseErr, attemptIndex)
		return openaiwire.ResponsesResponse{}, fmt.Errorf("reconstruct upstream response: %w", parseErr)
	}

	c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, nil, attemptIndex)
	return reconstructed, nil
}

func (c *Client) ResponsesStream(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error) {
	resp, payload, httpReq, startedAt, attemptIndex, err := c.doResponsesRequest(ctx, req, target)
	if err != nil {
		return nil, err
	}

	return chatcompletion.CaptureStream(resp.Body, func(streamBody []byte, streamErr error) {
		completedAt := time.Now().UTC()
		if reconstructed, parseErr := responsesusecase.ParseSSE(streamBody); parseErr == nil {
			if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
				recorder.SetResponsesResponse(reconstructed, true)
			}
		}
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, streamBody, startedAt, completedAt, streamErr, attemptIndex)
	}), nil
}

func (c *Client) doResponsesRequest(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (*http.Response, []byte, *http.Request, time.Time, int, error) {
	credential, err := c.resolveAccessToken(false)
	if err != nil {
		return nil, nil, nil, time.Time{}, 0, chatcompletion.ConnectionConfigurationError{
			ConnectionID:   c.connection.ID,
			ConnectionName: c.connection.Name,
			Message:        err.Error(),
		}
	}

	payload, err := c.marshalResponsesUpstreamRequest(ctx, req, target)
	if err != nil {
		return nil, nil, nil, time.Time{}, 0, fmt.Errorf("encode upstream request: %w", err)
	}
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetProviderRequestMode(true)
		recorder.SetTranslatedRequestBody(string(payload))
	}
	attemptIndex := chatcompletion.AttemptIndex(ctx)

	httpReq, err := c.newResponsesRequest(ctx, payload)
	if err != nil {
		return nil, nil, nil, time.Time{}, 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+credential)
	httpReq.Header.Set("Accept", "text/event-stream")

	startedAt := time.Now().UTC()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, nil, nil, startedAt, time.Now().UTC(), err, attemptIndex)
		return nil, nil, nil, time.Time{}, 0, fmt.Errorf("execute upstream request: %w", err)
	}

	if shouldRetryWithTokenRefresh(resp.StatusCode, c.connection) {
		resp.Body.Close()

		credential, err = c.resolveAccessToken(true)
		if err != nil {
			return nil, nil, nil, time.Time{}, 0, chatcompletion.ConnectionConfigurationError{
				ConnectionID:   c.connection.ID,
				ConnectionName: c.connection.Name,
				Message:        err.Error(),
			}
		}

		httpReq, err = c.newResponsesRequest(ctx, payload)
		if err != nil {
			return nil, nil, nil, time.Time{}, 0, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+credential)
		httpReq.Header.Set("Accept", "text/event-stream")

		startedAt = time.Now().UTC()
		resp, err = c.httpClient.Do(httpReq)
		if err != nil {
			c.recordThirdPartyLog(ctx, target, payload, httpReq, nil, nil, startedAt, time.Now().UTC(), err, attemptIndex)
			return nil, nil, nil, time.Time{}, 0, fmt.Errorf("execute upstream retry request: %w", err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		upstreamErr := chatcompletion.UpstreamError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, time.Now().UTC(), upstreamErr, attemptIndex)
		return nil, nil, nil, time.Time{}, 0, upstreamErr
	}

	return resp, payload, httpReq, startedAt, attemptIndex, nil
}

func (c *Client) marshalResponsesUpstreamRequest(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) ([]byte, error) {
	upstreamRequest := req
	if config.RTKEnabledFromContext(ctx) {
		compressed, summary := rtk.NewService().CompressResponses(upstreamRequest)
		upstreamRequest = compressed
		if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
			recorder.SetRTKSummary(summary)
		}
	}

	forceStream := true
	forceStore := false
	if len(upstreamRequest.RawBody) > 0 {
		payload, err := decodeResponsesRawPayload(upstreamRequest.RawBody)
		if err != nil {
			return nil, err
		}
		applyDefaultInstructionToRawPayload(payload)
		payload["model"] = target.RequestedModel
		payload["stream"] = forceStream
		payload["store"] = forceStore
		return json.Marshal(payload)
	}

	applyDefaultInstruction(responsesRequestBridge{instructions: &upstreamRequest.Instructions})
	upstreamRequest.Model = target.RequestedModel
	upstreamRequest.Stream = forceStream
	upstreamRequest.Store = forceStore
	upstreamRequest.RawBody = nil
	return json.Marshal(upstreamRequest)
}

func (c *Client) newResponsesRequest(ctx context.Context, payload []byte) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	forwardInboundHeaders(httpReq.Header, chatcompletion.InboundHeaders(ctx))
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if httpReq.Header.Get("originator") == "" {
		httpReq.Header.Set("originator", "codex-cli")
	}
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", defaultUserAgent)
	}
	return httpReq, nil
}

func forwardInboundHeaders(dst http.Header, src http.Header) {
	if len(src) == 0 {
		return
	}
	for key, values := range src {
		normalizedKey := strings.ToLower(key)
		if _, ok := forwardedInboundResponseHeaders[normalizedKey]; !ok {
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
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
		ProviderRequestMode: chatcompletion.RequestModeStream,
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

func responseHeader(response *http.Response) http.Header {
	if response == nil {
		return nil
	}
	return response.Header
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

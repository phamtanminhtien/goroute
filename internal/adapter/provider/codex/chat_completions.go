package codex

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	responsesusecase "github.com/phamtanminhtien/goroute/internal/usecase/responses"
)

func (c *Client) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	translated, err := translateChatCompletionsRequest(req)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}

	resp, payload, httpReq, startedAt, attemptIndex, err := c.doResponsesRequest(ctx, translated, target)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	completedAt := time.Now().UTC()
	if readErr != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, nil, startedAt, completedAt, readErr, attemptIndex)
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("read upstream response: %w", readErr)
	}

	reconstructed, parseErr := responsesusecase.ParseSSE(body)
	if parseErr != nil {
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, parseErr, attemptIndex)
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("reconstruct upstream response: %w", parseErr)
	}

	c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, body, startedAt, completedAt, nil, attemptIndex)
	return translateResponsesResponseToChatCompletion(reconstructed), nil
}

func (c *Client) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	translated, err := translateChatCompletionsRequest(req)
	if err != nil {
		return nil, err
	}

	resp, payload, httpReq, startedAt, attemptIndex, err := c.doResponsesRequest(ctx, translated, target)
	if err != nil {
		return nil, err
	}

	upstreamBody := chatcompletion.CaptureStream(resp.Body, func(streamBody []byte, streamErr error) {
		completedAt := time.Now().UTC()
		if reconstructed, parseErr := responsesusecase.ParseSSE(streamBody); parseErr == nil {
			if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
				recorder.SetFlowResponse(translateResponsesResponseToChatCompletion(reconstructed), true)
			}
		}
		c.recordThirdPartyLog(ctx, target, payload, httpReq, resp, streamBody, startedAt, completedAt, streamErr, attemptIndex)
	})

	return translateResponsesStreamToChatCompletions(upstreamBody), nil
}

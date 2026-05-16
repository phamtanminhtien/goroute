package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

type testProvider struct {
	response openaiwire.ChatCompletionsResponse
	err      error
	lastReq  openaiwire.ChatCompletionsRequest
}

func (p *testProvider) ChatCompletions(_ context.Context, req openaiwire.ChatCompletionsRequest, _ routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	p.lastReq = req
	return p.response, p.err
}

type streamingTestProvider struct {
	*testProvider
	body string
}

func (p streamingTestProvider) ChatCompletionsStream(_ context.Context, req openaiwire.ChatCompletionsRequest, _ routing.Target) (io.ReadCloser, error) {
	if p.testProvider != nil {
		p.testProvider.lastReq = req
	}
	return io.NopCloser(strings.NewReader(p.body)), p.err
}

type loggingTestProvider struct {
	testProvider
}

func (p *loggingTestProvider) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	response, err := p.testProvider.ChatCompletions(ctx, req, target)
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		if payload, err := json.Marshal(map[string]any{"model": target.RequestedModel}); err == nil {
			recorder.SetTranslatedRequestBody(string(payload))
		}
		recorder.AddThirdPartyLog(chatcompletion.ThirdPartyLog{
			ProviderID:          target.ProviderID,
			ProviderName:        target.ProviderName,
			ConnectionID:        "codex-1",
			ConnectionName:      "codex-user",
			AttemptIndex:        0,
			ProviderRequestMode: chatcompletion.RequestModeSync,
			RequestMethod:       "POST",
			RequestURL:          "https://provider.example/v1/chat/completions",
			RequestHeaders:      `{"Authorization":["[REDACTED]"]}`,
			RequestBody:         `{"model":"gpt-5.4"}`,
			ResponseStatusCode:  200,
			ResponseHeaders:     `{"Content-Type":["application/json"]}`,
			ResponseBody:        `{"id":"upstream-1"}`,
			StartedAt:           time.Now().UTC(),
			CompletedAt:         time.Now().UTC(),
		})
	}
	return response, err
}

type loggingStreamingTestProvider struct {
	*testProvider
	body string
}

func (p loggingStreamingTestProvider) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	if p.testProvider != nil {
		p.testProvider.lastReq = req
	}
	body := io.NopCloser(strings.NewReader(p.body))
	return chatcompletion.CaptureStream(body, func(streamBody []byte, _ error) {
		if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
			if payload, err := json.Marshal(map[string]any{"model": target.RequestedModel, "stream": true}); err == nil {
				recorder.SetTranslatedRequestBody(string(payload))
			}
			response := chatcompletion.BuildAssistantResponse(target.RequestedModel, chatcompletion.ExtractTextFromSSE(streamBody))
			recorder.SetFlowResponse(response, true)
			recorder.AddThirdPartyLog(chatcompletion.ThirdPartyLog{
				ProviderID:          target.ProviderID,
				ProviderName:        target.ProviderName,
				ConnectionID:        "codex-1",
				ConnectionName:      "codex-user",
				AttemptIndex:        0,
				RequestMode:         chatcompletion.RequestModeStream,
				ProviderRequestMode: chatcompletion.RequestModeStream,
				RequestMethod:       "POST",
				RequestURL:          "https://provider.example/v1/chat/completions",
				RequestHeaders:      `{"Authorization":["[REDACTED]"]}`,
				RequestBody:         `{"model":"gpt-5.4","stream":true}`,
				ResponseStatusCode:  200,
				ResponseHeaders:     `{"Content-Type":["text/event-stream"]}`,
				ResponseBody:        `{"object":"chat.completion","model":"gpt-5.4","choices":[{"index":0,"message":{"role":"assistant","content":"first"}}]}`,
				StartedAt:           time.Now().UTC(),
				CompletedAt:         time.Now().UTC(),
			})
		}
	}), p.err
}

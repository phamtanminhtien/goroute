package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/rtk"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

type testProvider struct {
	response            openaiwire.ChatCompletionsResponse
	err                 error
	lastReq             openaiwire.ChatCompletionsRequest
	lastTarget          routing.Target
	responsesResponse   openaiwire.ResponsesResponse
	responsesErr        error
	lastResponsesReq    openaiwire.ResponsesRequest
	lastResponsesTarget routing.Target
	anthropicResponse   anthropicwire.MessagesResponse
	anthropicErr        error
	lastAnthropicReq    anthropicwire.MessagesRequest
	lastAnthropicTarget routing.Target
}

func (p *testProvider) ChatCompletions(_ context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	p.lastReq = req
	p.lastTarget = target
	return p.response, p.err
}

func (p *testProvider) Responses(_ context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	p.lastResponsesReq = req
	p.lastResponsesTarget = target
	return p.responsesResponse, p.responsesErr
}

func (p *testProvider) AnthropicMessages(_ context.Context, req anthropicwire.MessagesRequest, target routing.Target) (anthropicwire.MessagesResponse, error) {
	p.lastAnthropicReq = req
	p.lastAnthropicTarget = target
	return p.anthropicResponse, p.anthropicErr
}

func (p *testProvider) ChatCompletionsStream(_ context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	p.lastReq = req
	p.lastTarget = target
	if p.err != nil {
		return nil, p.err
	}

	var content string
	if len(p.response.Choices) > 0 {
		content = p.response.Choices[0].Message.Content
	}
	payload := fmt.Sprintf("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":%q}}]}\n\ndata: [DONE]\n\n", content)
	return io.NopCloser(strings.NewReader(payload)), nil
}

func (p *testProvider) ResponsesStream(_ context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error) {
	p.lastResponsesReq = req
	p.lastResponsesTarget = target
	if p.responsesErr != nil {
		return nil, p.responsesErr
	}

	return io.NopCloser(strings.NewReader("data: {\"type\":\"response.created\"}\n\ndata: [DONE]\n\n")), nil
}

func (p *testProvider) AnthropicMessagesStream(_ context.Context, req anthropicwire.MessagesRequest, target routing.Target) (io.ReadCloser, error) {
	p.lastAnthropicReq = req
	p.lastAnthropicTarget = target
	if p.anthropicErr != nil {
		return nil, p.anthropicErr
	}
	return io.NopCloser(strings.NewReader("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")), nil
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

func (p streamingTestProvider) ResponsesStream(_ context.Context, req openaiwire.ResponsesRequest, _ routing.Target) (io.ReadCloser, error) {
	if p.testProvider != nil {
		p.testProvider.lastResponsesReq = req
	}
	return io.NopCloser(strings.NewReader(p.body)), p.responsesErr
}

type loggingTestProvider struct {
	testProvider
}

func (p *loggingTestProvider) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	response, err := p.testProvider.ChatCompletions(ctx, req, target)
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		upstreamReq := req
		upstreamReq.Model = target.RequestedModel
		if payload, err := json.Marshal(upstreamReq); err == nil {
			recorder.SetTranslatedRequestBody(string(payload))
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
				RequestBody:         chatcompletion.RedactBodyForStorage(string(payload)),
				ResponseStatusCode:  200,
				ResponseHeaders:     `{"Content-Type":["application/json"]}`,
				ResponseBody:        `{"id":"upstream-1"}`,
				StartedAt:           time.Now().UTC(),
				CompletedAt:         time.Now().UTC(),
			})
		}
	}
	return response, err
}

func (p *loggingTestProvider) Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	response, err := p.testProvider.Responses(ctx, req, target)
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		upstreamReq := req
		upstreamReq.Model = target.RequestedModel
		if payload, err := json.Marshal(upstreamReq); err == nil {
			recorder.SetTranslatedRequestBody(string(payload))
			recorder.AddThirdPartyLog(chatcompletion.ThirdPartyLog{
				ProviderID:          target.ProviderID,
				ProviderName:        target.ProviderName,
				ConnectionID:        "codex-1",
				ConnectionName:      "codex-user",
				AttemptIndex:        0,
				ProviderRequestMode: chatcompletion.RequestModeSync,
				RequestMethod:       "POST",
				RequestURL:          "https://provider.example/v1/responses",
				RequestHeaders:      `{"Authorization":["[REDACTED]"]}`,
				RequestBody:         chatcompletion.RedactBodyForStorage(string(payload)),
				ResponseStatusCode:  200,
				ResponseHeaders:     `{"Content-Type":["application/json"]}`,
				ResponseBody:        `{"id":"upstream-resp-1"}`,
				StartedAt:           time.Now().UTC(),
				CompletedAt:         time.Now().UTC(),
			})
		}
	}
	return response, err
}

type loggingStreamingTestProvider struct {
	*testProvider
	body string
}

type rtkLoggingTestProvider struct {
	loggingTestProvider
}

func (p *rtkLoggingTestProvider) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	if config.RTKEnabledFromContext(ctx) {
		compressed, summary := rtk.NewService().CompressChatCompletions(req)
		req = compressed
		if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
			recorder.SetRTKSummary(summary)
		}
	}

	return p.loggingTestProvider.ChatCompletions(ctx, req, target)
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

func (p loggingStreamingTestProvider) ResponsesStream(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error) {
	if p.testProvider != nil {
		p.testProvider.lastResponsesReq = req
	}
	body := io.NopCloser(strings.NewReader(p.body))
	return chatcompletion.CaptureStream(body, func(streamBody []byte, _ error) {
		if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
			if payload, err := json.Marshal(map[string]any{"model": target.RequestedModel, "stream": true}); err == nil {
				recorder.SetTranslatedRequestBody(string(payload))
			}
			recorder.SetResponsesResponse(openaiwire.ResponsesResponse{
				ID:        "resp_1",
				Object:    "response",
				CreatedAt: 1712345678,
				Model:     target.RequestedModel,
				Status:    openaiwire.ResponsesStatusCompleted,
				Output: []openaiwire.OutputItem{{
					Type: openaiwire.OutputItemTypeMessage,
					Role: string(openaiwire.ChatRoleAssistant),
					Content: []openaiwire.OutputContent{{
						Type: openaiwire.OutputContentTypeOutputText,
						Text: "first",
					}},
				}},
			}, true)
			recorder.AddThirdPartyLog(chatcompletion.ThirdPartyLog{
				ProviderID:          target.ProviderID,
				ProviderName:        target.ProviderName,
				ConnectionID:        "codex-1",
				ConnectionName:      "codex-user",
				AttemptIndex:        0,
				RequestMode:         chatcompletion.RequestModeStream,
				ProviderRequestMode: chatcompletion.RequestModeStream,
				RequestMethod:       "POST",
				RequestURL:          "https://provider.example/v1/responses",
				RequestHeaders:      `{"Authorization":["[REDACTED]"]}`,
				RequestBody:         `{"model":"gpt-5.4","stream":true}`,
				ResponseStatusCode:  200,
				ResponseHeaders:     `{"Content-Type":["text/event-stream"]}`,
				ResponseBody:        string(streamBody),
				StartedAt:           time.Now().UTC(),
				CompletedAt:         time.Now().UTC(),
			})
		}
	}), p.responsesErr
}

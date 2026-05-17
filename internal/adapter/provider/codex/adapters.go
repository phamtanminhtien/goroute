package codex

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	responsesusecase "github.com/phamtanminhtien/goroute/internal/usecase/responses"
)

type ResponsesAdapter struct {
	core *Client
}

type ChatCompletionsAdapter struct {
	responses *ResponsesAdapter
}

func NewProtocolConnections(connectionConfig connection.Record) chatcompletion.ProtocolConnections {
	return newProtocolConnectionsWithHTTPClient(nil, connectionConfig)
}

func newProtocolConnectionsWithHTTPClient(httpClient *http.Client, connectionConfig connection.Record) chatcompletion.ProtocolConnections {
	core := NewClientWithHTTPClient(httpClient, connectionConfig)
	responses := &ResponsesAdapter{core: core}

	return chatcompletion.ProtocolConnections{
		ChatCompletions: &ChatCompletionsAdapter{responses: responses},
		Responses:       responses,
	}
}

func (a *ResponsesAdapter) Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	streamBody, err := a.streamResponses(ctx, req, target, nil)
	if err != nil {
		return openaiwire.ResponsesResponse{}, err
	}
	defer streamBody.Close()

	captured, err := io.ReadAll(streamBody)
	if err != nil {
		return openaiwire.ResponsesResponse{}, fmt.Errorf("read codex responses stream: %w", err)
	}

	response, err := responsesusecase.ParseSSE(captured)
	if err != nil {
		return openaiwire.ResponsesResponse{}, err
	}
	return response, nil
}

func (a *ResponsesAdapter) ResponsesStream(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error) {
	return a.streamResponses(ctx, req, target, func(captured []byte, recorder *chatcompletion.FlowRecorder) {
		response, err := responsesusecase.ParseSSE(captured)
		if err != nil {
			return
		}
		recorder.SetResponsesResponse(response, true)
	})
}

func (a *ResponsesAdapter) streamResponses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target, capture func([]byte, *chatcompletion.FlowRecorder)) (io.ReadCloser, error) {
	body := normalizeResponsesRequest(req)
	body.Stream = true
	return a.core.executeResponsesRequest(ctx, body, target, capture)
}

func (a *ChatCompletionsAdapter) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	streamBody, err := a.streamChatCompletions(ctx, req, target)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, err
	}
	defer streamBody.Close()

	streamData, err := io.ReadAll(streamBody)
	if err != nil {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("read codex stream response: %w", err)
	}

	return reconstructStreamResponse(target, streamData), nil
}

func (a *ChatCompletionsAdapter) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	streamBody, err := a.streamChatCompletions(ctx, req, target)
	if err != nil {
		return nil, err
	}

	return transformResponsesToChatCompletionsStream(streamBody, target.Prefix+"/"+target.RequestedModel), nil
}

func (a *ChatCompletionsAdapter) streamChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	body := req
	body.Stream = true

	return a.responses.streamResponses(ctx, chatCompletionsToCodexResponses(body, target.RequestedModel), target, func(captured []byte, recorder *chatcompletion.FlowRecorder) {
		recorder.SetFlowResponse(reconstructStreamResponse(target, captured), true)
	})
}

func reconstructStreamResponse(target routing.Target, streamBody []byte) openaiwire.ChatCompletionsResponse {
	response, err := parseResponsesSSE(streamBody)
	if err != nil {
		text := chatcompletion.ExtractTextFromSSE(streamBody)
		return chatcompletion.BuildAssistantResponse(target.RequestedModel, text)
	}
	return responseToChatCompletion(response, target.RequestedModel)
}

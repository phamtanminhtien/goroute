package chatcompletion

import (
	"context"
	"io"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type ChatCompletionsConnection interface {
	ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error)
	ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error)
}

type ResponsesConnection interface {
	Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error)
	ResponsesStream(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error)
}

type AnthropicConnection interface {
	AnthropicMessages(ctx context.Context, req anthropicwire.MessagesRequest, target routing.Target) (anthropicwire.MessagesResponse, error)
	AnthropicMessagesStream(ctx context.Context, req anthropicwire.MessagesRequest, target routing.Target) (io.ReadCloser, error)
}

type ProtocolConnections struct {
	ChatCompletions ChatCompletionsConnection
	Responses       ResponsesConnection
	Anthropic       AnthropicConnection
}

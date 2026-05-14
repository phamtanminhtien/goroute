package chatcompletion

import (
	"context"
	"io"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type Provider interface {
	ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error)
}

type StreamingProvider interface {
	Provider
	ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error)
}

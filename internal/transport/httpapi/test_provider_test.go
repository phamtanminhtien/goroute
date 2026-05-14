package httpapi

import (
	"context"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type testProvider struct {
	response openaiwire.ChatCompletionsResponse
	err      error
}

func (p testProvider) ChatCompletions(_ context.Context, _ openaiwire.ChatCompletionsRequest, _ routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	return p.response, p.err
}

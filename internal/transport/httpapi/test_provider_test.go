package httpapi

import (
	"context"
	"io"
	"strings"

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

type streamingTestProvider struct {
	testProvider
	body string
}

func (p streamingTestProvider) ChatCompletionsStream(_ context.Context, _ openaiwire.ChatCompletionsRequest, _ routing.Target) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.body)), p.err
}

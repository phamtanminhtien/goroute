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

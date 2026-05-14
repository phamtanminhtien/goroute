package chatcompletion

import (
	"io"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type Input struct {
	Request openaiwire.ChatCompletionsRequest
}

type Output struct {
	Response openaiwire.ChatCompletionsResponse
}

type StreamOutput struct {
	Body io.ReadCloser
}

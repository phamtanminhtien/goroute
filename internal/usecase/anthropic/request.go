package anthropic

import (
	"io"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
)

type Input struct {
	Request anthropicwire.MessagesRequest
}

type Output struct {
	Response anthropicwire.MessagesResponse
}

type StreamOutput struct {
	Body io.ReadCloser
}

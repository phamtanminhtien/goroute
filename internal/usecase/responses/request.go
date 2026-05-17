package responses

import (
	"io"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type Input struct {
	Request openaiwire.ResponsesRequest
}

type Output struct {
	Response openaiwire.ResponsesResponse
}

type StreamOutput struct {
	Body io.ReadCloser
}

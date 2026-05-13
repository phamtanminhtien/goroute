package chatcompletion

import "github.com/phamtanminhtien/goroute/internal/openaiwire"

type Input struct {
	Request openaiwire.ChatCompletionsRequest
}

type Output struct {
	Response openaiwire.ChatCompletionsResponse
}

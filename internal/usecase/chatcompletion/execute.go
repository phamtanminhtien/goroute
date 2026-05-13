package chatcompletion

import (
	"context"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func Execute(_ context.Context, catalog driver.Catalog, input Input) (Output, error) {
	target, err := routing.ResolveModel(catalog, input.Request.Model)
	if err != nil {
		return Output{}, err
	}

	if len(input.Request.Messages) == 0 {
		return Output{}, fmt.Errorf("messages must contain at least one item")
	}

	return Output{
		Response: openaiwire.ChatCompletionsResponse{
			ID:     "chatcmpl-bootstrap",
			Object: "chat.completion",
			Model:  target.Prefix + "/" + target.RequestedModel,
			Choices: []openaiwire.ChatChoice{
				{
					Index: 0,
					Message: openaiwire.ChatMessage{
						Role:    "assistant",
						Content: "bootstrap response: upstream provider execution is not implemented yet",
					},
				},
			},
		},
	}, nil
}

package chatcompletion

import (
	"context"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
)

func Execute(ctx context.Context, catalog driver.Catalog, providerRegistry *ProviderRegistry, input Input) (Output, error) {
	target, err := routing.ResolveModel(catalog, input.Request.Model)
	if err != nil {
		return Output{}, err
	}

	if len(input.Request.Messages) == 0 {
		return Output{}, fmt.Errorf("messages must contain at least one item")
	}

	response, err := providerRegistry.ChatCompletions(ctx, input.Request, target)
	if err != nil {
		return Output{}, err
	}

	response.Model = target.Prefix + "/" + target.RequestedModel

	return Output{Response: response}, nil
}

func ExecuteStream(ctx context.Context, catalog driver.Catalog, providerRegistry *ProviderRegistry, input Input) (StreamOutput, error) {
	target, err := routing.ResolveModel(catalog, input.Request.Model)
	if err != nil {
		return StreamOutput{}, err
	}

	if len(input.Request.Messages) == 0 {
		return StreamOutput{}, fmt.Errorf("messages must contain at least one item")
	}

	body, err := providerRegistry.ChatCompletionsStream(ctx, input.Request, target)
	if err != nil {
		return StreamOutput{}, err
	}

	return StreamOutput{Body: body}, nil
}

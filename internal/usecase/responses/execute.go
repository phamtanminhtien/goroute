package responses

import (
	"context"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func Execute(ctx context.Context, catalog provider.Catalog, connectionRegistry *chatcompletion.ConnectionRegistry, input Input) (Output, error) {
	target, err := routing.ResolveModel(catalog, input.Request.Model)
	if err != nil {
		return Output{}, err
	}
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetRequestedModel(input.Request.Model)
		recorder.SetRequestMode(false)
		recorder.SetResolvedTarget(target)
	}

	response, err := connectionRegistry.Responses(ctx, input.Request, target)
	if err != nil {
		return Output{}, err
	}

	response.Model = target.Prefix + "/" + target.RequestedModel
	return Output{Response: response}, nil
}

func ExecuteStream(ctx context.Context, catalog provider.Catalog, connectionRegistry *chatcompletion.ConnectionRegistry, input Input) (StreamOutput, error) {
	target, err := routing.ResolveModel(catalog, input.Request.Model)
	if err != nil {
		return StreamOutput{}, err
	}
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetRequestedModel(input.Request.Model)
		recorder.SetRequestMode(true)
		recorder.SetResolvedTarget(target)
	}

	body, err := connectionRegistry.ResponsesStream(ctx, input.Request, target)
	if err != nil {
		return StreamOutput{}, err
	}

	return StreamOutput{Body: body}, nil
}

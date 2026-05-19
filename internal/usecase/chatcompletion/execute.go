package chatcompletion

import (
	"context"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/protocoltranslator"
)

func Execute(ctx context.Context, catalog provider.Catalog, combos []modelcombo.Combo, connectionRegistry *ConnectionRegistry, input Input) (Output, error) {
	plan, err := routing.ResolvePlan(catalog, combos, input.Request.Model)
	if err != nil {
		return Output{}, err
	}
	if recorder := FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetRequestedModel(input.Request.Model)
		recorder.SetRequestMode(false)
		recorder.SetResolvedTarget(plan.PrimaryTarget())
	}

	if len(input.Request.Messages) == 0 {
		return Output{}, fmt.Errorf("messages must contain at least one item")
	}
	if err := openaiwire.ValidateChatCompletionsRequest(input.Request); err != nil {
		return Output{}, err
	}

	response, err := connectionRegistry.ChatCompletionsTargets(ctx, input.Request, plan.Targets)
	if err != nil {
		return Output{}, err
	}

	response.Model = plan.ResponseModel

	return Output{Response: response}, nil
}

func ExecuteStream(ctx context.Context, catalog provider.Catalog, combos []modelcombo.Combo, connectionRegistry *ConnectionRegistry, input Input) (StreamOutput, error) {
	plan, err := routing.ResolvePlan(catalog, combos, input.Request.Model)
	if err != nil {
		return StreamOutput{}, err
	}
	if recorder := FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetRequestedModel(input.Request.Model)
		recorder.SetRequestMode(true)
		recorder.SetResolvedTarget(plan.PrimaryTarget())
	}

	if len(input.Request.Messages) == 0 {
		return StreamOutput{}, fmt.Errorf("messages must contain at least one item")
	}
	if err := openaiwire.ValidateChatCompletionsRequest(input.Request); err != nil {
		return StreamOutput{}, err
	}

	body, err := connectionRegistry.ChatCompletionsStreamTargets(ctx, input.Request, plan.Targets)
	if err != nil {
		return StreamOutput{}, err
	}

	return StreamOutput{Body: protocoltranslator.RewriteChatCompletionsStreamModel(body, plan.ResponseModel)}, nil
}

package responses

import (
	"context"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/protocoltranslator"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func Execute(ctx context.Context, catalog provider.Catalog, combos []modelcombo.Combo, connectionRegistry *chatcompletion.ConnectionRegistry, input Input) (Output, error) {
	plan, err := routing.ResolvePlan(catalog, combos, input.Request.Model)
	if err != nil {
		return Output{}, err
	}
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetRequestedModel(input.Request.Model)
		recorder.SetRequestMode(false)
		recorder.SetResolvedTarget(plan.PrimaryTarget())
	}

	response, err := connectionRegistry.ResponsesTargets(ctx, input.Request, plan.Targets)
	if err != nil {
		return Output{}, err
	}

	response.Model = plan.ResponseModel
	return Output{Response: response}, nil
}

func ExecuteStream(ctx context.Context, catalog provider.Catalog, combos []modelcombo.Combo, connectionRegistry *chatcompletion.ConnectionRegistry, input Input) (StreamOutput, error) {
	plan, err := routing.ResolvePlan(catalog, combos, input.Request.Model)
	if err != nil {
		return StreamOutput{}, err
	}
	if recorder := chatcompletion.FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetRequestedModel(input.Request.Model)
		recorder.SetRequestMode(true)
		recorder.SetResolvedTarget(plan.PrimaryTarget())
	}

	body, err := connectionRegistry.ResponsesStreamTargets(ctx, input.Request, plan.Targets)
	if err != nil {
		return StreamOutput{}, err
	}

	return StreamOutput{Body: protocoltranslator.RewriteResponsesStreamModel(body, plan.ResponseModel)}, nil
}

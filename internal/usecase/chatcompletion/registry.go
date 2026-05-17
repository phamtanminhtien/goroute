package chatcompletion

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/rs/zerolog"
)

type ConnectionEntry struct {
	ID         string
	Name       string
	ProviderID string
	ProtocolConnections
}

type ConnectionRegistry struct {
	mu          sync.RWMutex
	connections map[string][]ConnectionEntry
	logger      *zerolog.Logger
}

func NewConnectionRegistry(connections map[string][]ConnectionEntry) ConnectionRegistry {
	return NewConnectionRegistryWithEntries(connections, nil)
}

func NewConnectionRegistryWithEntries(connections map[string][]ConnectionEntry, logger *zerolog.Logger) ConnectionRegistry {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return ConnectionRegistry{
		connections: connections,
		logger:      logger,
	}
}

func (r *ConnectionRegistry) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		target,
		func(entry ConnectionEntry) bool { return entry.ChatCompletions != nil },
		func(entry ConnectionEntry) (openaiwire.ChatCompletionsResponse, error) {
			return entry.ChatCompletions.ChatCompletions(ctx, req, target)
		},
		"chat_completions_unsupported",
	)
}

func (r *ConnectionRegistry) Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		target,
		func(entry ConnectionEntry) bool { return entry.Responses != nil },
		func(entry ConnectionEntry) (openaiwire.ResponsesResponse, error) {
			return entry.Responses.Responses(ctx, req, target)
		},
		"responses_unsupported",
	)
}

func (r *ConnectionRegistry) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		target,
		func(entry ConnectionEntry) bool { return entry.ChatCompletions != nil },
		func(entry ConnectionEntry) (io.ReadCloser, error) {
			return entry.ChatCompletions.ChatCompletionsStream(ctx, req, target)
		},
		"chat_completions_unsupported",
	)
}

func (r *ConnectionRegistry) ResponsesStream(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		target,
		func(entry ConnectionEntry) bool { return entry.Responses != nil },
		func(entry ConnectionEntry) (io.ReadCloser, error) {
			return entry.Responses.ResponsesStream(ctx, req, target)
		},
		"responses_unsupported",
	)
}

func executeProtocol[T any](r *ConnectionRegistry, ctx context.Context, requestedModel string, target routing.Target, supported func(ConnectionEntry) bool, invoke func(ConnectionEntry) (T, error), unsupportedCategory string) (T, error) {
	connections := r.connectionsForProvider(target.ProviderID)
	requestID := RequestID(ctx)
	if len(connections) == 0 {
		return zeroValue[T](), fmt.Errorf("no executor configured for provider %q", target.ProviderID)
	}

	var lastErr error
	var lastPolicy FailurePolicy
	for i, connection := range connections {
		if !supported(connection) {
			lastErr = fmt.Errorf("connection %q does not support requested protocol", connection.Name)
			lastPolicy = FailurePolicy{
				Class:         FailureClassFallbackEligible,
				Category:      unsupportedCategory,
				AllowFallback: true,
			}
			r.logAttempt(ctx, requestID, requestedModel, target, connection, i, 0, string(lastPolicy.Class), lastPolicy.Category, true)
			continue
		}

		started := time.Now().UTC()
		response, err := invoke(connection)
		completedAt := time.Now().UTC()
		latency := completedAt.Sub(started)
		if err == nil {
			r.logAttempt(ctx, requestID, requestedModel, target, connection, i, latency, "success", "none", false)
			return response, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(ctx, requestID, requestedModel, target, connection, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		lastErr = err
		lastPolicy = policy
		if !policy.AllowFallback {
			r.logFinalFailure(ctx, requestID, requestedModel, target, policy.Category)
			return zeroValue[T](), err
		}
	}

	if lastErr != nil {
		r.logFinalFailure(ctx, requestID, requestedModel, target, lastPolicy.Category)
		return zeroValue[T](), lastErr
	}

	return zeroValue[T](), nil
}

func zeroValue[T any]() T {
	var zero T
	return zero
}

func (r *ConnectionRegistry) ReplaceConnections(connections map[string][]ConnectionEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.connections = connections
}

func (r *ConnectionRegistry) connectionsForProvider(providerID string) []ConnectionEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	connections := r.connections[providerID]
	if len(connections) == 0 {
		return nil
	}

	cloned := make([]ConnectionEntry, len(connections))
	copy(cloned, connections)
	return cloned
}

func (r *ConnectionRegistry) logAttempt(ctx context.Context, requestID string, requestedModel string, target routing.Target, connection ConnectionEntry, attempt int, latency time.Duration, outcome string, errorCategory string, willFallback bool) {
	if recorder := FlowRecorderFromContext(ctx); recorder != nil {
		recorder.RecordAttempt(target, connection, attempt, latency, outcome, errorCategory, willFallback)
	}
	r.logger.Info().
		Str("request_id", requestID).
		Str("requested_model", requestedModel).
		Str("resolved_target", target.Prefix+"/"+target.RequestedModel).
		Str("provider_id", target.ProviderID).
		Str("provider_name", target.ProviderName).
		Str("connection_id", connection.ID).
		Str("connection_name", connection.Name).
		Int("attempt_index", attempt).
		Str("outcome", outcome).
		Int64("latency_ms", latency.Milliseconds()).
		Str("error_category", errorCategory).
		Bool("will_fallback", willFallback).
		Msg("provider_request_attempt")
}

func (r *ConnectionRegistry) logFinalFailure(ctx context.Context, requestID string, requestedModel string, target routing.Target, finalCategory string) {
	if recorder := FlowRecorderFromContext(ctx); recorder != nil {
		recorder.SetFinalErrorCategory(finalCategory)
	}
	r.logger.Warn().
		Str("request_id", requestID).
		Str("requested_model", requestedModel).
		Str("resolved_target", target.Prefix+"/"+target.RequestedModel).
		Str("provider_id", target.ProviderID).
		Str("provider_name", target.ProviderName).
		Str("final_error_category", finalCategory).
		Msg("provider_request_result")
}

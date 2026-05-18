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
	ID                string
	Name              string
	ProviderID        string
	LastErrorMessage  string
	LastErrorCategory string
	LastErrorAt       int64
	RetryAfter        int64
	ProtocolConnections
}

type ConnectionRuntimeStateStore interface {
	RecordConnectionRuntimeError(id string, message string, category string, lastErrorAt int64, retryAfter int64) error
	ClearConnectionRuntimeError(id string) error
}

type ConnectionRegistry struct {
	mu          sync.RWMutex
	connections map[string][]ConnectionEntry
	logger      *zerolog.Logger
	stateStore  ConnectionRuntimeStateStore
}

var registryTimeNow = func() time.Time {
	return time.Now().UTC()
}

func NewConnectionRegistry(connections map[string][]ConnectionEntry) ConnectionRegistry {
	return NewConnectionRegistryWithEntries(connections, nil)
}

func NewConnectionRegistryWithEntries(connections map[string][]ConnectionEntry, logger *zerolog.Logger) ConnectionRegistry {
	return NewConnectionRegistryWithStateStore(connections, logger, nil)
}

func NewConnectionRegistryWithStateStore(connections map[string][]ConnectionEntry, logger *zerolog.Logger, stateStore ConnectionRuntimeStateStore) ConnectionRegistry {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return ConnectionRegistry{
		connections: connections,
		logger:      logger,
		stateStore:  stateStore,
	}
}

func (r *ConnectionRegistry) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	return r.ChatCompletionsTargets(ctx, req, []routing.Target{target})
}

func (r *ConnectionRegistry) ChatCompletionsTargets(ctx context.Context, req openaiwire.ChatCompletionsRequest, targets []routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		targets,
		func(entry ConnectionEntry) bool { return entry.ChatCompletions != nil },
		func(ctx context.Context, entry ConnectionEntry, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
			return entry.ChatCompletions.ChatCompletions(ctx, req, target)
		},
		"chat_completions_unsupported",
	)
}

func (r *ConnectionRegistry) Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	return r.ResponsesTargets(ctx, req, []routing.Target{target})
}

func (r *ConnectionRegistry) ResponsesTargets(ctx context.Context, req openaiwire.ResponsesRequest, targets []routing.Target) (openaiwire.ResponsesResponse, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		targets,
		func(entry ConnectionEntry) bool { return entry.Responses != nil },
		func(ctx context.Context, entry ConnectionEntry, target routing.Target) (openaiwire.ResponsesResponse, error) {
			return entry.Responses.Responses(ctx, req, target)
		},
		"responses_unsupported",
	)
}

func (r *ConnectionRegistry) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	return r.ChatCompletionsStreamTargets(ctx, req, []routing.Target{target})
}

func (r *ConnectionRegistry) ChatCompletionsStreamTargets(ctx context.Context, req openaiwire.ChatCompletionsRequest, targets []routing.Target) (io.ReadCloser, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		targets,
		func(entry ConnectionEntry) bool { return entry.ChatCompletions != nil },
		func(ctx context.Context, entry ConnectionEntry, target routing.Target) (io.ReadCloser, error) {
			return entry.ChatCompletions.ChatCompletionsStream(ctx, req, target)
		},
		"chat_completions_unsupported",
	)
}

func (r *ConnectionRegistry) ResponsesStream(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (io.ReadCloser, error) {
	return r.ResponsesStreamTargets(ctx, req, []routing.Target{target})
}

func (r *ConnectionRegistry) ResponsesStreamTargets(ctx context.Context, req openaiwire.ResponsesRequest, targets []routing.Target) (io.ReadCloser, error) {
	return executeProtocol(
		r,
		ctx,
		req.Model,
		targets,
		func(entry ConnectionEntry) bool { return entry.Responses != nil },
		func(ctx context.Context, entry ConnectionEntry, target routing.Target) (io.ReadCloser, error) {
			return entry.Responses.ResponsesStream(ctx, req, target)
		},
		"responses_unsupported",
	)
}

func executeProtocol[T any](r *ConnectionRegistry, ctx context.Context, requestedModel string, targets []routing.Target, supported func(ConnectionEntry) bool, invoke func(context.Context, ConnectionEntry, routing.Target) (T, error), unsupportedCategory string) (T, error) {
	requestID := RequestID(ctx)
	if len(targets) == 0 {
		return zeroValue[T](), fmt.Errorf("no routing targets configured")
	}

	var lastErr error
	var lastPolicy FailurePolicy
	attemptIndex := 0
	for _, target := range targets {
		connections := r.connectionsForTarget(target)
		if len(connections) == 0 {
			if target.ConnectionID != "" {
				lastErr = fmt.Errorf("no executor configured for connection %q on provider %q", target.ConnectionID, target.ProviderID)
			} else {
				lastErr = fmt.Errorf("no executor configured for provider %q", target.ProviderID)
			}
			lastPolicy = FailurePolicy{
				Class:         FailureClassFallbackEligible,
				Category:      "no_executor",
				AllowFallback: true,
			}
			continue
		}

		for _, connection := range connections {
			currentAttempt := attemptIndex
			attemptIndex++
			if retryAt, ok := connectionActiveCooldown(connection, registryTimeNow()); ok {
				lastErr = fmt.Errorf("connection %q is cooling down until %s", connection.Name, retryAt.Format(time.RFC3339))
				lastPolicy = FailurePolicy{
					Class:         FailureClassFallbackEligible,
					Category:      "connection_cooldown",
					AllowFallback: true,
				}
				r.logAttempt(ctx, requestID, requestedModel, target, connection, currentAttempt, 0, string(lastPolicy.Class), lastPolicy.Category, true)
				continue
			}
			if !supported(connection) {
				lastErr = fmt.Errorf("connection %q does not support requested protocol", connection.Name)
				lastPolicy = FailurePolicy{
					Class:         FailureClassFallbackEligible,
					Category:      unsupportedCategory,
					AllowFallback: true,
				}
				r.logAttempt(ctx, requestID, requestedModel, target, connection, currentAttempt, 0, string(lastPolicy.Class), lastPolicy.Category, true)
				continue
			}

			started := registryTimeNow()
			response, err := invoke(WithAttemptIndex(ctx, currentAttempt), connection, target)
			completedAt := registryTimeNow()
			latency := completedAt.Sub(started)
			if err == nil {
				r.clearConnectionRuntimeError(connection)
				r.logAttempt(ctx, requestID, requestedModel, target, connection, currentAttempt, latency, "success", "none", false)
				return response, nil
			}

			policy := ClassifyError(err)
			r.recordConnectionRuntimeError(connection, err, policy, completedAt)
			r.logAttempt(ctx, requestID, requestedModel, target, connection, currentAttempt, latency, string(policy.Class), policy.Category, policy.AllowFallback)
			lastErr = err
			lastPolicy = policy
			if !policy.AllowFallback {
				r.logFinalFailure(ctx, requestID, requestedModel, target, policy.Category)
				return zeroValue[T](), err
			}
		}
	}

	if lastErr != nil {
		r.logFinalFailure(ctx, requestID, requestedModel, targets[len(targets)-1], lastPolicy.Category)
		return zeroValue[T](), lastErr
	}

	return zeroValue[T](), nil
}

func connectionActiveCooldown(connection ConnectionEntry, now time.Time) (time.Time, bool) {
	if connection.RetryAfter <= 0 {
		return time.Time{}, false
	}

	retryAt := time.Unix(connection.RetryAfter, 0).UTC()
	return retryAt, retryAt.After(now)
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

func (r *ConnectionRegistry) connectionsForTarget(target routing.Target) []ConnectionEntry {
	connections := r.connectionsForProvider(target.ProviderID)
	if target.ConnectionID == "" || len(connections) == 0 {
		return connections
	}

	filtered := make([]ConnectionEntry, 0, 1)
	for _, connection := range connections {
		if connection.ID == target.ConnectionID {
			filtered = append(filtered, connection)
		}
	}

	return filtered
}

func (r *ConnectionRegistry) recordConnectionRuntimeError(connection ConnectionEntry, err error, policy FailurePolicy, occurredAt time.Time) {
	retryAfter := int64(0)
	if policy.Class == FailureClassRetryable {
		retryAfter = occurredAt.Add(RetryableConnectionCooldown).Unix()
	}

	message := err.Error()
	if r.stateStore != nil {
		if storeErr := r.stateStore.RecordConnectionRuntimeError(connection.ID, message, policy.Category, occurredAt.Unix(), retryAfter); storeErr != nil {
			r.logger.Error().
				Err(storeErr).
				Str("connection_id", connection.ID).
				Msg("connection_runtime_error_record_failed")
		}
	}
	r.updateConnectionRuntimeState(connection.ID, message, policy.Category, occurredAt.Unix(), retryAfter)
}

func (r *ConnectionRegistry) clearConnectionRuntimeError(connection ConnectionEntry) {
	if connection.LastErrorMessage == "" && connection.LastErrorCategory == "" && connection.LastErrorAt == 0 && connection.RetryAfter == 0 {
		return
	}

	if r.stateStore != nil {
		if err := r.stateStore.ClearConnectionRuntimeError(connection.ID); err != nil {
			r.logger.Error().
				Err(err).
				Str("connection_id", connection.ID).
				Msg("connection_runtime_error_clear_failed")
		}
	}
	r.updateConnectionRuntimeState(connection.ID, "", "", 0, 0)
}

func (r *ConnectionRegistry) updateConnectionRuntimeState(connectionID string, message string, category string, lastErrorAt int64, retryAfter int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for providerID, connections := range r.connections {
		for index := range connections {
			if connections[index].ID != connectionID {
				continue
			}
			connections[index].LastErrorMessage = message
			connections[index].LastErrorCategory = category
			connections[index].LastErrorAt = lastErrorAt
			connections[index].RetryAfter = retryAfter
			r.connections[providerID] = connections
			return
		}
	}
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

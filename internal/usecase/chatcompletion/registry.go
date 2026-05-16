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
	Connection Connection
}

type ConnectionRegistry struct {
	mu          sync.RWMutex
	connections map[string][]ConnectionEntry
	logger      *zerolog.Logger
}

func NewConnectionRegistry(connections map[string][]Connection) ConnectionRegistry {
	entries := make(map[string][]ConnectionEntry, len(connections))
	for providerID, configuredConnections := range connections {
		for i, connection := range configuredConnections {
			entries[providerID] = append(entries[providerID], ConnectionEntry{
				ID:         fmt.Sprintf("%s-%d", providerID, i+1),
				Name:       fmt.Sprintf("%s-%d", providerID, i+1),
				ProviderID: providerID,
				Connection: connection,
			})
		}
	}

	return NewConnectionRegistryWithEntries(entries, nil)
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
	connections := r.connectionsForProvider(target.ProviderID)
	requestID := RequestID(ctx)
	if len(connections) == 0 {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("no executor configured for provider %q", target.ProviderID)
	}

	var lastErr error
	var lastPolicy FailurePolicy
	for i, connection := range connections {
		started := time.Now().UTC()
		response, err := connection.Connection.ChatCompletions(ctx, req, target)
		completedAt := time.Now().UTC()
		latency := completedAt.Sub(started)
		if err == nil {
			r.logAttempt(ctx, requestID, req.Model, target, connection, i, latency, "success", "none", false)
			return response, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(ctx, requestID, req.Model, target, connection, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		lastErr = err
		lastPolicy = policy
		if !policy.AllowFallback {
			r.logFinalFailure(ctx, requestID, req.Model, target, policy.Category)
			return openaiwire.ChatCompletionsResponse{}, err
		}
	}

	if lastErr != nil {
		r.logFinalFailure(ctx, requestID, req.Model, target, lastPolicy.Category)
	}

	return openaiwire.ChatCompletionsResponse{}, lastErr
}

func (r *ConnectionRegistry) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	connections := r.connectionsForProvider(target.ProviderID)
	requestID := RequestID(ctx)
	if len(connections) == 0 {
		return nil, fmt.Errorf("no executor configured for provider %q", target.ProviderID)
	}

	var lastErr error
	var lastPolicy FailurePolicy
	for i, connection := range connections {
		streamingConnection, ok := connection.Connection.(StreamingConnection)
		if !ok {
			lastErr = fmt.Errorf("connection %q does not support streaming", connection.Name)
			lastPolicy = FailurePolicy{
				Class:         FailureClassFallbackEligible,
				Category:      "streaming_unsupported",
				AllowFallback: true,
			}
			r.logAttempt(ctx, requestID, req.Model, target, connection, i, 0, string(lastPolicy.Class), lastPolicy.Category, true)
			continue
		}

		started := time.Now().UTC()
		body, err := streamingConnection.ChatCompletionsStream(ctx, req, target)
		completedAt := time.Now().UTC()
		latency := completedAt.Sub(started)
		if err == nil {
			r.logAttempt(ctx, requestID, req.Model, target, connection, i, latency, "success", "none", false)
			return body, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(ctx, requestID, req.Model, target, connection, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		lastErr = err
		lastPolicy = policy
		if !policy.AllowFallback {
			r.logFinalFailure(ctx, requestID, req.Model, target, policy.Category)
			return nil, err
		}
	}

	if lastErr != nil {
		r.logFinalFailure(ctx, requestID, req.Model, target, lastPolicy.Category)
	}

	return nil, lastErr
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
		Msg("chat_completion_attempt")
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
		Msg("chat_completion_result")
}

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
	history     HistoryStore
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
	return NewConnectionRegistryWithEntriesAndHistory(connections, logger, nil)
}

func NewConnectionRegistryWithEntriesAndHistory(connections map[string][]ConnectionEntry, logger *zerolog.Logger, history HistoryStore) ConnectionRegistry {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}
	if history == nil {
		history = newMemoryHistoryStore(defaultHistoryLimit)
	}

	return ConnectionRegistry{
		connections: connections,
		logger:      logger,
		history:     history,
	}
}

func (r *ConnectionRegistry) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	connections := r.connectionsForProvider(target.ProviderID)
	requestID := RequestID(ctx)
	record := r.startRequestHistory(ctx, req, target, false)
	if len(connections) == 0 {
		finalized := r.finalizeRequestHistory(record, RequestStatusError, "connection_unavailable", fmt.Sprintf("no executor configured for provider %q", target.ProviderID))
		r.persistRequestHistoryUpdate(finalized)
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
			r.logAttempt(requestID, req.Model, target, connection, i, latency, "success", "none", false)
			record = r.appendAttempt(record, connection, i, "success", "none", "", false, started, completedAt)
			record = r.finalizeRequestHistory(record, RequestStatusSuccess, "", "")
			r.persistRequestHistoryUpdate(record)
			return response, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(requestID, req.Model, target, connection, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		record = r.appendAttempt(record, connection, i, string(policy.Class), policy.Category, err.Error(), policy.AllowFallback, started, completedAt)
		lastErr = err
		lastPolicy = policy
		if !policy.AllowFallback {
			r.logFinalFailure(requestID, req.Model, target, policy.Category)
			record = r.finalizeRequestHistory(record, RequestStatusError, policy.Category, err.Error())
			r.persistRequestHistoryUpdate(record)
			return openaiwire.ChatCompletionsResponse{}, err
		}
		record.Status = RequestStatusRetrying
		r.persistRequestHistoryUpdate(record)
	}

	if lastErr != nil {
		r.logFinalFailure(requestID, req.Model, target, lastPolicy.Category)
		record = r.finalizeRequestHistory(record, RequestStatusError, lastPolicy.Category, lastErr.Error())
		r.persistRequestHistoryUpdate(record)
	}

	return openaiwire.ChatCompletionsResponse{}, lastErr
}

func (r *ConnectionRegistry) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	connections := r.connectionsForProvider(target.ProviderID)
	requestID := RequestID(ctx)
	record := r.startRequestHistory(ctx, req, target, true)
	if len(connections) == 0 {
		finalized := r.finalizeRequestHistory(record, RequestStatusError, "connection_unavailable", fmt.Sprintf("no executor configured for provider %q", target.ProviderID))
		r.persistRequestHistoryUpdate(finalized)
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
			r.logAttempt(requestID, req.Model, target, connection, i, 0, string(lastPolicy.Class), lastPolicy.Category, true)
			now := time.Now().UTC()
			record = r.appendAttempt(record, connection, i, string(lastPolicy.Class), lastPolicy.Category, lastErr.Error(), true, now, now)
			record.Status = RequestStatusRetrying
			r.persistRequestHistoryUpdate(record)
			continue
		}

		started := time.Now().UTC()
		body, err := streamingConnection.ChatCompletionsStream(ctx, req, target)
		completedAt := time.Now().UTC()
		latency := completedAt.Sub(started)
		if err == nil {
			r.logAttempt(requestID, req.Model, target, connection, i, latency, "success", "none", false)
			record = r.appendAttempt(record, connection, i, "success", "none", "", false, started, completedAt)
			record = r.finalizeRequestHistory(record, RequestStatusSuccess, "", "")
			r.persistRequestHistoryUpdate(record)
			return body, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(requestID, req.Model, target, connection, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		record = r.appendAttempt(record, connection, i, string(policy.Class), policy.Category, err.Error(), policy.AllowFallback, started, completedAt)
		lastErr = err
		lastPolicy = policy
		if !policy.AllowFallback {
			r.logFinalFailure(requestID, req.Model, target, policy.Category)
			record = r.finalizeRequestHistory(record, RequestStatusError, policy.Category, err.Error())
			r.persistRequestHistoryUpdate(record)
			return nil, err
		}
		record.Status = RequestStatusRetrying
		r.persistRequestHistoryUpdate(record)
	}

	if lastErr != nil {
		r.logFinalFailure(requestID, req.Model, target, lastPolicy.Category)
		record = r.finalizeRequestHistory(record, RequestStatusError, lastPolicy.Category, lastErr.Error())
		r.persistRequestHistoryUpdate(record)
	}

	return nil, lastErr
}

func (r *ConnectionRegistry) RecentRequestAttempts(limit int) ([]RequestAttemptHistory, error) {
	if r.history == nil {
		return nil, nil
	}

	return r.history.RecentRequestAttempts(limit)
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

func (r *ConnectionRegistry) logAttempt(requestID string, requestedModel string, target routing.Target, connection ConnectionEntry, attempt int, latency time.Duration, outcome string, errorCategory string, willFallback bool) {
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

func (r *ConnectionRegistry) logFinalFailure(requestID string, requestedModel string, target routing.Target, finalCategory string) {
	r.logger.Warn().
		Str("request_id", requestID).
		Str("requested_model", requestedModel).
		Str("resolved_target", target.Prefix+"/"+target.RequestedModel).
		Str("provider_id", target.ProviderID).
		Str("provider_name", target.ProviderName).
		Str("final_error_category", finalCategory).
		Msg("chat_completion_result")
}

func (r *ConnectionRegistry) startRequestHistory(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target, stream bool) RequestAttemptHistory {
	now := time.Now().UTC()
	record := RequestAttemptHistory{
		RequestID:      RequestID(ctx),
		RequestPath:    "/v1/chat/completions",
		RequestedModel: req.Model,
		ResolvedTarget: target.Prefix + "/" + target.RequestedModel,
		ProviderID:     target.ProviderID,
		ProviderName:   target.ProviderName,
		Stream:         stream,
		Status:         RequestStatusStarted,
		MessageCount:   len(req.Messages),
		ToolCount:      len(req.Tools),
		StartedAt:      now,
		UpdatedAt:      now,
	}

	if r.history == nil {
		return record
	}

	created, err := r.history.CreateRequestAttemptHistory(record)
	if err != nil {
		r.logger.Error().Err(err).Msg("request_history_create_failed")
		return record
	}

	return created
}

func (r *ConnectionRegistry) appendAttempt(record RequestAttemptHistory, connection ConnectionEntry, attemptIndex int, outcome string, errorCategory string, errorMessage string, willFallback bool, startedAt time.Time, completedAt time.Time) RequestAttemptHistory {
	record.Attempts = append(record.Attempts, RequestAttempt{
		ConnectionID:   connection.ID,
		ConnectionName: connection.Name,
		AttemptIndex:   attemptIndex,
		Outcome:        outcome,
		ErrorCategory:  errorCategory,
		ErrorMessage:   errorMessage,
		WillFallback:   willFallback,
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		LatencyMillis:  completedAt.Sub(startedAt).Milliseconds(),
	})
	record.AttemptCount = len(record.Attempts)
	record.LastConnectionID = connection.ID
	record.LastConnectionName = connection.Name
	record.LastAttemptAt = completedAt
	record.UpdatedAt = completedAt
	record.LastErrorCategory = errorCategory
	record.LastErrorMessage = errorMessage
	if outcome == "success" {
		record.LastErrorCategory = ""
		record.LastErrorMessage = ""
	}
	return record
}

func (r *ConnectionRegistry) finalizeRequestHistory(record RequestAttemptHistory, status string, errorCategory string, errorMessage string) RequestAttemptHistory {
	now := time.Now().UTC()
	record.Status = status
	record.FinalStatus = status
	record.FinalErrorCategory = errorCategory
	record.LastErrorCategory = errorCategory
	record.LastErrorMessage = errorMessage
	record.CompletedAt = now
	record.UpdatedAt = now
	return record
}

func (r *ConnectionRegistry) persistRequestHistoryUpdate(record RequestAttemptHistory) {
	if r.history == nil || record.HistoryID == 0 {
		return
	}

	if err := r.history.UpdateRequestAttemptHistory(record); err != nil {
		r.logger.Error().Err(err).Msg("request_history_update_failed")
	}
}

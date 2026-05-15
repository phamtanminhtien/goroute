package chatcompletion

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
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
	logger      *log.Logger
	history     *requestHistoryStore
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

	return NewConnectionRegistryWithEntries(entries, log.Default())
}

func NewConnectionRegistryWithEntries(connections map[string][]ConnectionEntry, logger *log.Logger) ConnectionRegistry {
	if logger == nil {
		logger = log.Default()
	}

	return ConnectionRegistry{
		connections: connections,
		logger:      logger,
		history:     newRequestHistoryStore(defaultHistoryLimit),
	}
}

func (r *ConnectionRegistry) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	connections := r.connectionsForProvider(target.ProviderID)
	if len(connections) == 0 {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("no executor configured for provider %q", target.ProviderID)
	}

	record := RequestAttemptHistory{
		RequestID:      RequestID(ctx),
		RequestedModel: req.Model,
		ResolvedTarget: target.Prefix + "/" + target.RequestedModel,
		ProviderID:     target.ProviderID,
		ProviderName:   target.ProviderName,
		Stream:         false,
		StartedAt:      time.Now().UTC(),
	}
	defer func() {
		record.CompletedAt = time.Now().UTC()
		r.history.add(record)
	}()

	var lastErr error
	var lastPolicy FailurePolicy
	for i, connection := range connections {
		started := time.Now()
		response, err := connection.Connection.ChatCompletions(ctx, req, target)
		latency := time.Since(started)
		if err == nil {
			r.logAttempt(req.Model, target, connection, i, latency, "success", "none", false)
			record.Attempts = append(record.Attempts, RequestAttempt{
				ConnectionID:   connection.ID,
				ConnectionName: connection.Name,
				AttemptIndex:   i,
				Outcome:        "success",
				ErrorCategory:  "none",
				LatencyMillis:  latency.Milliseconds(),
			})
			record.FinalStatus = "success"
			return response, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(req.Model, target, connection, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		record.Attempts = append(record.Attempts, RequestAttempt{
			ConnectionID:   connection.ID,
			ConnectionName: connection.Name,
			AttemptIndex:   i,
			Outcome:        string(policy.Class),
			ErrorCategory:  policy.Category,
			LatencyMillis:  latency.Milliseconds(),
		})
		lastErr = err
		lastPolicy = policy
		if !policy.AllowFallback {
			r.logFinalFailure(req.Model, target, policy.Category)
			record.FinalStatus = "error"
			record.FinalErrorCategory = policy.Category
			return openaiwire.ChatCompletionsResponse{}, err
		}
	}

	if lastErr != nil {
		r.logFinalFailure(req.Model, target, lastPolicy.Category)
		record.FinalStatus = "error"
		record.FinalErrorCategory = lastPolicy.Category
	}

	return openaiwire.ChatCompletionsResponse{}, lastErr
}

func (r *ConnectionRegistry) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	connections := r.connectionsForProvider(target.ProviderID)
	if len(connections) == 0 {
		return nil, fmt.Errorf("no executor configured for provider %q", target.ProviderID)
	}

	record := RequestAttemptHistory{
		RequestID:      RequestID(ctx),
		RequestedModel: req.Model,
		ResolvedTarget: target.Prefix + "/" + target.RequestedModel,
		ProviderID:     target.ProviderID,
		ProviderName:   target.ProviderName,
		Stream:         true,
		StartedAt:      time.Now().UTC(),
	}
	defer func() {
		record.CompletedAt = time.Now().UTC()
		r.history.add(record)
	}()

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
			r.logAttempt(req.Model, target, connection, i, 0, string(lastPolicy.Class), lastPolicy.Category, true)
			record.Attempts = append(record.Attempts, RequestAttempt{
				ConnectionID:   connection.ID,
				ConnectionName: connection.Name,
				AttemptIndex:   i,
				Outcome:        string(lastPolicy.Class),
				ErrorCategory:  lastPolicy.Category,
			})
			continue
		}

		started := time.Now()
		body, err := streamingConnection.ChatCompletionsStream(ctx, req, target)
		latency := time.Since(started)
		if err == nil {
			r.logAttempt(req.Model, target, connection, i, latency, "success", "none", false)
			record.Attempts = append(record.Attempts, RequestAttempt{
				ConnectionID:   connection.ID,
				ConnectionName: connection.Name,
				AttemptIndex:   i,
				Outcome:        "success",
				ErrorCategory:  "none",
				LatencyMillis:  latency.Milliseconds(),
			})
			record.FinalStatus = "success"
			return body, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(req.Model, target, connection, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		record.Attempts = append(record.Attempts, RequestAttempt{
			ConnectionID:   connection.ID,
			ConnectionName: connection.Name,
			AttemptIndex:   i,
			Outcome:        string(policy.Class),
			ErrorCategory:  policy.Category,
			LatencyMillis:  latency.Milliseconds(),
		})
		lastErr = err
		lastPolicy = policy
		if !policy.AllowFallback {
			r.logFinalFailure(req.Model, target, policy.Category)
			record.FinalStatus = "error"
			record.FinalErrorCategory = policy.Category
			return nil, err
		}
	}

	if lastErr != nil {
		r.logFinalFailure(req.Model, target, lastPolicy.Category)
		record.FinalStatus = "error"
		record.FinalErrorCategory = lastPolicy.Category
	}

	return nil, lastErr
}

func (r *ConnectionRegistry) RecentRequestAttempts(limit int) []RequestAttemptHistory {
	if r.history == nil {
		return nil
	}

	return r.history.recent(limit)
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

func (r *ConnectionRegistry) logAttempt(requestedModel string, target routing.Target, connection ConnectionEntry, attempt int, latency time.Duration, outcome string, errorCategory string, willFallback bool) {
	r.logger.Printf(
		"chat_completion_attempt requested_model=%q resolved_target=%q provider_id=%q provider_name=%q connection_id=%q connection_name=%q attempt_index=%d outcome=%q latency=%s error_category=%q will_fallback=%t",
		requestedModel,
		target.Prefix+"/"+target.RequestedModel,
		target.ProviderID,
		target.ProviderName,
		connection.ID,
		connection.Name,
		attempt,
		outcome,
		latency,
		errorCategory,
		willFallback,
	)
}

func (r *ConnectionRegistry) logFinalFailure(requestedModel string, target routing.Target, finalCategory string) {
	r.logger.Printf(
		"chat_completion_result requested_model=%q resolved_target=%q provider_id=%q provider_name=%q final_error_category=%q",
		requestedModel,
		target.Prefix+"/"+target.RequestedModel,
		target.ProviderID,
		target.ProviderName,
		finalCategory,
	)
}

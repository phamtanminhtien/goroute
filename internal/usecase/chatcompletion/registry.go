package chatcompletion

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type ProviderEntry struct {
	ID       string
	Name     string
	Provider Provider
}

type ProviderRegistry struct {
	providers map[string][]ProviderEntry
	logger    *log.Logger
	history   *requestHistoryStore
}

func NewProviderRegistry(providers map[string][]Provider) ProviderRegistry {
	entries := make(map[string][]ProviderEntry, len(providers))
	for providerType, configuredProviders := range providers {
		for i, provider := range configuredProviders {
			entries[providerType] = append(entries[providerType], ProviderEntry{
				ID:       fmt.Sprintf("%s-%d", providerType, i+1),
				Name:     fmt.Sprintf("%s-%d", providerType, i+1),
				Provider: provider,
			})
		}
	}

	return NewProviderRegistryWithEntries(entries, log.Default())
}

func NewProviderRegistryWithEntries(providers map[string][]ProviderEntry, logger *log.Logger) ProviderRegistry {
	if logger == nil {
		logger = log.Default()
	}

	return ProviderRegistry{
		providers: providers,
		logger:    logger,
		history:   newRequestHistoryStore(defaultHistoryLimit),
	}
}

func (r ProviderRegistry) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	providers := r.providers[target.ProviderType]
	if len(providers) == 0 {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("no executor configured for provider type %q", target.ProviderType)
	}

	record := RequestAttemptHistory{
		RequestID:      RequestID(ctx),
		RequestedModel: req.Model,
		ResolvedTarget: target.Prefix + "/" + target.RequestedModel,
		ProviderType:   target.ProviderType,
		Stream:         false,
		StartedAt:      time.Now().UTC(),
	}
	defer func() {
		record.CompletedAt = time.Now().UTC()
		r.history.add(record)
	}()

	var lastErr error
	var lastPolicy FailurePolicy
	for i, provider := range providers {
		started := time.Now()
		response, err := provider.Provider.ChatCompletions(ctx, req, target)
		latency := time.Since(started)
		if err == nil {
			r.logAttempt(req.Model, target, provider, i, latency, "success", "none", false)
			record.Attempts = append(record.Attempts, RequestAttempt{
				ProviderID:    provider.ID,
				ProviderName:  provider.Name,
				AttemptIndex:  i,
				Outcome:       "success",
				ErrorCategory: "none",
				LatencyMillis: latency.Milliseconds(),
			})
			record.FinalStatus = "success"
			return response, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(req.Model, target, provider, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		record.Attempts = append(record.Attempts, RequestAttempt{
			ProviderID:    provider.ID,
			ProviderName:  provider.Name,
			AttemptIndex:  i,
			Outcome:       string(policy.Class),
			ErrorCategory: policy.Category,
			LatencyMillis: latency.Milliseconds(),
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

func (r ProviderRegistry) ChatCompletionsStream(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (io.ReadCloser, error) {
	providers := r.providers[target.ProviderType]
	if len(providers) == 0 {
		return nil, fmt.Errorf("no executor configured for provider type %q", target.ProviderType)
	}

	record := RequestAttemptHistory{
		RequestID:      RequestID(ctx),
		RequestedModel: req.Model,
		ResolvedTarget: target.Prefix + "/" + target.RequestedModel,
		ProviderType:   target.ProviderType,
		Stream:         true,
		StartedAt:      time.Now().UTC(),
	}
	defer func() {
		record.CompletedAt = time.Now().UTC()
		r.history.add(record)
	}()

	var lastErr error
	var lastPolicy FailurePolicy
	for i, provider := range providers {
		streamingProvider, ok := provider.Provider.(StreamingProvider)
		if !ok {
			lastErr = fmt.Errorf("provider %q does not support streaming", provider.Name)
			lastPolicy = FailurePolicy{
				Class:         FailureClassFallbackEligible,
				Category:      "streaming_unsupported",
				AllowFallback: true,
			}
			r.logAttempt(req.Model, target, provider, i, 0, string(lastPolicy.Class), lastPolicy.Category, true)
			record.Attempts = append(record.Attempts, RequestAttempt{
				ProviderID:    provider.ID,
				ProviderName:  provider.Name,
				AttemptIndex:  i,
				Outcome:       string(lastPolicy.Class),
				ErrorCategory: lastPolicy.Category,
			})
			continue
		}

		started := time.Now()
		body, err := streamingProvider.ChatCompletionsStream(ctx, req, target)
		latency := time.Since(started)
		if err == nil {
			r.logAttempt(req.Model, target, provider, i, latency, "success", "none", false)
			record.Attempts = append(record.Attempts, RequestAttempt{
				ProviderID:    provider.ID,
				ProviderName:  provider.Name,
				AttemptIndex:  i,
				Outcome:       "success",
				ErrorCategory: "none",
				LatencyMillis: latency.Milliseconds(),
			})
			record.FinalStatus = "success"
			return body, nil
		}

		policy := ClassifyError(err)
		r.logAttempt(req.Model, target, provider, i, latency, string(policy.Class), policy.Category, policy.AllowFallback)
		record.Attempts = append(record.Attempts, RequestAttempt{
			ProviderID:    provider.ID,
			ProviderName:  provider.Name,
			AttemptIndex:  i,
			Outcome:       string(policy.Class),
			ErrorCategory: policy.Category,
			LatencyMillis: latency.Milliseconds(),
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

func (r ProviderRegistry) RecentRequestAttempts(limit int) []RequestAttemptHistory {
	if r.history == nil {
		return nil
	}

	return r.history.recent(limit)
}

func (r ProviderRegistry) logAttempt(requestedModel string, target routing.Target, provider ProviderEntry, attempt int, latency time.Duration, outcome string, errorCategory string, willFallback bool) {
	r.logger.Printf(
		"chat_completion_attempt requested_model=%q resolved_target=%q provider_type=%q provider_id=%q provider_name=%q attempt_index=%d outcome=%q latency=%s error_category=%q will_fallback=%t",
		requestedModel,
		target.Prefix+"/"+target.RequestedModel,
		target.ProviderType,
		provider.ID,
		provider.Name,
		attempt,
		outcome,
		latency,
		errorCategory,
		willFallback,
	)
}

func (r ProviderRegistry) logFinalFailure(requestedModel string, target routing.Target, finalCategory string) {
	r.logger.Printf(
		"chat_completion_result requested_model=%q resolved_target=%q provider_type=%q final_error_category=%q",
		requestedModel,
		target.Prefix+"/"+target.RequestedModel,
		target.ProviderType,
		finalCategory,
	)
}

package chatcompletion

import (
	"context"
	"errors"
	"fmt"
	"net"
)

type UpstreamError struct {
	StatusCode int
	Message    string
}

func (e UpstreamError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("upstream returned status %d", e.StatusCode)
	}

	return fmt.Sprintf("upstream returned status %d: %s", e.StatusCode, e.Message)
}

type ConnectionConfigurationError struct {
	ConnectionID   string
	ConnectionName string
	Message        string
}

func (e ConnectionConfigurationError) Error() string {
	if e.ConnectionName != "" {
		return fmt.Sprintf("connection %q misconfigured: %s", e.ConnectionName, e.Message)
	}

	if e.ConnectionID != "" {
		return fmt.Sprintf("connection %q misconfigured: %s", e.ConnectionID, e.Message)
	}

	return fmt.Sprintf("connection misconfigured: %s", e.Message)
}

type FailureClass string

const (
	FailureClassRetryable        FailureClass = "retryable"
	FailureClassFallbackEligible FailureClass = "fallback_eligible"
	FailureClassTerminal         FailureClass = "terminal"
)

type FailurePolicy struct {
	Class         FailureClass
	Category      string
	AllowFallback bool
}

func ClassifyError(err error) FailurePolicy {
	var upstreamErr UpstreamError
	switch {
	case errors.As(err, &upstreamErr):
		return classifyUpstreamError(upstreamErr)
	case errors.As(err, new(ConnectionConfigurationError)):
		return FailurePolicy{
			Class:         FailureClassTerminal,
			Category:      "connection_config_error",
			AllowFallback: false,
		}
	case errors.Is(err, context.Canceled):
		return FailurePolicy{
			Class:         FailureClassTerminal,
			Category:      "context_canceled",
			AllowFallback: false,
		}
	case errors.Is(err, context.DeadlineExceeded):
		return FailurePolicy{
			Class:         FailureClassRetryable,
			Category:      "timeout",
			AllowFallback: true,
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return FailurePolicy{
			Class:         FailureClassRetryable,
			Category:      "network_error",
			AllowFallback: true,
		}
	}

	return FailurePolicy{
		Class:         FailureClassFallbackEligible,
		Category:      "unknown_error",
		AllowFallback: true,
	}
}

func classifyUpstreamError(err UpstreamError) FailurePolicy {
	switch {
	case err.StatusCode == 401 || err.StatusCode == 403:
		return FailurePolicy{
			Class:         FailureClassTerminal,
			Category:      "upstream_auth_error",
			AllowFallback: false,
		}
	case err.StatusCode == 400 || err.StatusCode == 404 || err.StatusCode == 405 || err.StatusCode == 409 || err.StatusCode == 410 || err.StatusCode == 422:
		return FailurePolicy{
			Class:         FailureClassTerminal,
			Category:      "upstream_client_error",
			AllowFallback: false,
		}
	case err.StatusCode == 408 || err.StatusCode == 425 || err.StatusCode == 429:
		return FailurePolicy{
			Class:         FailureClassRetryable,
			Category:      "upstream_retryable_error",
			AllowFallback: true,
		}
	case err.StatusCode >= 500:
		return FailurePolicy{
			Class:         FailureClassRetryable,
			Category:      "upstream_server_error",
			AllowFallback: true,
		}
	default:
		return FailurePolicy{
			Class:         FailureClassFallbackEligible,
			Category:      "upstream_unknown_error",
			AllowFallback: true,
		}
	}
}

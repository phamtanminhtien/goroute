package chatcompletion

import (
	"context"
	"testing"
)

func TestClassifyError(t *testing.T) {
	testCases := []struct {
		name         string
		err          error
		wantClass    FailureClass
		wantCategory string
		wantFallback bool
	}{
		{
			name:         "unauthorized upstream is terminal",
			err:          UpstreamError{StatusCode: 401, Message: "bad key"},
			wantClass:    FailureClassTerminal,
			wantCategory: "upstream_auth_error",
		},
		{
			name:         "bad request upstream is terminal",
			err:          UpstreamError{StatusCode: 400, Message: "bad model"},
			wantClass:    FailureClassTerminal,
			wantCategory: "upstream_client_error",
		},
		{
			name:         "rate limited upstream is retryable",
			err:          UpstreamError{StatusCode: 429, Message: "slow down"},
			wantClass:    FailureClassRetryable,
			wantCategory: "upstream_retryable_error",
			wantFallback: true,
		},
		{
			name:         "server error upstream is retryable",
			err:          UpstreamError{StatusCode: 503, Message: "unavailable"},
			wantClass:    FailureClassRetryable,
			wantCategory: "upstream_server_error",
			wantFallback: true,
		},
		{
			name:         "connection config error is terminal",
			err:          ConnectionConfigurationError{ConnectionID: "c1", Message: "missing token"},
			wantClass:    FailureClassTerminal,
			wantCategory: "connection_config_error",
		},
		{
			name:         "context cancellation is terminal",
			err:          context.Canceled,
			wantClass:    FailureClassTerminal,
			wantCategory: "context_canceled",
		},
		{
			name:         "deadline exceeded is retryable",
			err:          context.DeadlineExceeded,
			wantClass:    FailureClassRetryable,
			wantCategory: "timeout",
			wantFallback: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyError(tc.err)
			if got.Class != tc.wantClass {
				t.Fatalf("expected class %q, got %q", tc.wantClass, got.Class)
			}
			if got.Category != tc.wantCategory {
				t.Fatalf("expected category %q, got %q", tc.wantCategory, got.Category)
			}
			if got.AllowFallback != tc.wantFallback {
				t.Fatalf("expected allowFallback=%t, got %t", tc.wantFallback, got.AllowFallback)
			}
		})
	}
}

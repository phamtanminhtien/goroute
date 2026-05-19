package chatcompletion

import (
	"context"
	"net/http"
)

type contextKey string

const requestIDContextKey contextKey = "request_id"
const attemptIndexContextKey contextKey = "attempt_index"
const inboundHeadersContextKey contextKey = "inbound_headers"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey).(string)
	return value
}

func WithAttemptIndex(ctx context.Context, attemptIndex int) context.Context {
	return context.WithValue(ctx, attemptIndexContextKey, attemptIndex)
}

func AttemptIndex(ctx context.Context) int {
	value, _ := ctx.Value(attemptIndexContextKey).(int)
	return value
}

func WithInboundHeaders(ctx context.Context, headers http.Header) context.Context {
	if headers == nil {
		return context.WithValue(ctx, inboundHeadersContextKey, http.Header(nil))
	}
	return context.WithValue(ctx, inboundHeadersContextKey, headers.Clone())
}

func InboundHeaders(ctx context.Context) http.Header {
	value, _ := ctx.Value(inboundHeadersContextKey).(http.Header)
	if value == nil {
		return nil
	}
	return value.Clone()
}

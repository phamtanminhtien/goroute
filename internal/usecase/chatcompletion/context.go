package chatcompletion

import "context"

type contextKey string

const requestIDContextKey contextKey = "request_id"
const attemptIndexContextKey contextKey = "attempt_index"

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

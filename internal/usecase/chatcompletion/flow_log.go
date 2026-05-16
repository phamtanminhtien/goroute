package chatcompletion

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

const (
	RequestTypeCompletions = "completions"
	RequestModeSync        = "sync"
	RequestModeStream      = "stream"
)

type flowLogContextKey string

const flowRecorderContextKey flowLogContextKey = "flow_recorder"

type AttemptTrace struct {
	ConnectionID   string `json:"connection_id"`
	ConnectionName string `json:"connection_name"`
	ProviderID     string `json:"provider_id"`
	ProviderName   string `json:"provider_name"`
	AttemptIndex   int    `json:"attempt_index"`
	Outcome        string `json:"outcome"`
	ErrorCategory  string `json:"error_category"`
	WillFallback   bool   `json:"will_fallback"`
	LatencyMs      int64  `json:"latency_ms"`
}

type ThirdPartyLog struct {
	FlowRequestID       string
	Type                string
	RequestMode         string
	ProviderRequestMode string
	ProviderID          string
	ProviderName        string
	ConnectionID        string
	ConnectionName      string
	AttemptIndex        int
	RequestMethod       string
	RequestURL          string
	RequestHeaders      string
	RequestBody         string
	ResponseStatusCode  int
	ResponseHeaders     string
	ResponseBody        string
	ErrorType           string
	ErrorMessage        string
	StartedAt           time.Time
	CompletedAt         time.Time
}

type FlowRecorder struct {
	mu sync.Mutex

	requestID string
	startedAt time.Time

	requestType           string
	requestMode           string
	providerRequestMode   string
	method                string
	path                  string
	query                 string
	remoteAddr            string
	userAgent             string
	requestHeaders        string
	requestBody           string
	translatedRequestBody string

	requestedModel string
	resolvedModel  string
	providerID     string
	providerName   string

	finalConnectionID   string
	finalConnectionName string
	finalErrorCategory  string
	attemptTrace        []AttemptTrace

	responseStatusCode     int
	responseHeaders        string
	responseBody           string
	responseBodySet        bool
	translatedResponseBody string

	errorType    string
	errorMessage string

	promptTokens     int
	completionTokens int
	totalTokens      int

	thirdPartyLogs []ThirdPartyLog
}

func NewFlowRecorder(requestID string, startedAt time.Time) *FlowRecorder {
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	return &FlowRecorder{
		requestID:    requestID,
		startedAt:    startedAt,
		requestType:  RequestTypeCompletions,
		requestMode:  RequestModeSync,
		attemptTrace: make([]AttemptTrace, 0, 1),
	}
}

func WithFlowRecorder(ctx context.Context, recorder *FlowRecorder) context.Context {
	return context.WithValue(ctx, flowRecorderContextKey, recorder)
}

func FlowRecorderFromContext(ctx context.Context) *FlowRecorder {
	recorder, _ := ctx.Value(flowRecorderContextKey).(*FlowRecorder)
	return recorder
}

func (r *FlowRecorder) ConfigureInbound(req *http.Request, body []byte) {
	if r == nil || req == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.method = req.Method
	r.path = req.URL.Path
	r.query = req.URL.RawQuery
	r.remoteAddr = req.RemoteAddr
	r.userAgent = req.UserAgent()
	r.requestHeaders = marshalHeaders(req.Header, true)
	r.requestBody = redactBody(string(body))
}

func (r *FlowRecorder) SetRequestedModel(model string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.requestedModel = model
}

func (r *FlowRecorder) SetRequestMode(stream bool) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if stream {
		r.requestMode = RequestModeStream
		if r.providerRequestMode == "" {
			r.providerRequestMode = RequestModeStream
		}
		return
	}
	r.requestMode = RequestModeSync
	if r.providerRequestMode == "" {
		r.providerRequestMode = RequestModeSync
	}
}

func (r *FlowRecorder) SetProviderRequestMode(stream bool) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if stream {
		r.providerRequestMode = RequestModeStream
		return
	}
	r.providerRequestMode = RequestModeSync
}

func (r *FlowRecorder) SetResolvedTarget(target routing.Target) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolvedModel = resolvedModel(target)
	r.providerID = target.ProviderID
	r.providerName = target.ProviderName
}

func (r *FlowRecorder) RecordAttempt(target routing.Target, connection ConnectionEntry, attempt int, latency time.Duration, outcome string, errorCategory string, willFallback bool) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.providerID = target.ProviderID
	r.providerName = target.ProviderName
	r.resolvedModel = resolvedModel(target)
	r.attemptTrace = append(r.attemptTrace, AttemptTrace{
		ConnectionID:   connection.ID,
		ConnectionName: connection.Name,
		ProviderID:     target.ProviderID,
		ProviderName:   target.ProviderName,
		AttemptIndex:   attempt,
		Outcome:        outcome,
		ErrorCategory:  errorCategory,
		WillFallback:   willFallback,
		LatencyMs:      latency.Milliseconds(),
	})
	if outcome == "success" {
		r.finalConnectionID = connection.ID
		r.finalConnectionName = connection.Name
	}
}

func (r *FlowRecorder) SetFinalErrorCategory(category string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.finalErrorCategory = category
}

func (r *FlowRecorder) SetHTTPResponse(statusCode int, headers http.Header, body string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.responseStatusCode = statusCode
	r.responseHeaders = marshalHeaders(headers, false)
	if !r.responseBodySet {
		r.responseBody = responseBodyForStorage(r.requestMode, body)
	}
}

func (r *FlowRecorder) SetResponseBody(body string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.responseBody = body
	r.responseBodySet = true
}

func (r *FlowRecorder) SetTranslatedRequestBody(body string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.translatedRequestBody = redactBody(body)
}

func (r *FlowRecorder) SetTranslatedResponseBody(body string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.translatedResponseBody = redactBody(body)
}

func (r *FlowRecorder) SetError(errorType string, message string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.errorType = errorType
	r.errorMessage = message
}

func (r *FlowRecorder) SetUsage(usage *openaiwire.Usage) {
	if r == nil || usage == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.promptTokens = usage.PromptTokens
	r.completionTokens = usage.CompletionTokens
	r.totalTokens = usage.TotalTokens
}

func (r *FlowRecorder) SetFlowResponse(response openaiwire.ChatCompletionsResponse, normalizeModel bool) {
	if r == nil {
		return
	}

	if normalizeModel {
		if model := r.ResolvedModel(); model != "" {
			response.Model = model
		}
	}

	payload, err := json.Marshal(response)
	if err != nil {
		return
	}

	r.SetTranslatedResponseBody(string(payload))
	r.SetUsage(response.Usage)
}

func (r *FlowRecorder) AddThirdPartyLog(log ThirdPartyLog) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if log.FlowRequestID == "" {
		log.FlowRequestID = r.requestID
	}
	if log.Type == "" {
		log.Type = r.requestType
	}
	if log.RequestMode == "" {
		log.RequestMode = r.requestMode
	}
	if log.ProviderRequestMode == "" {
		if r.providerRequestMode != "" {
			log.ProviderRequestMode = r.providerRequestMode
		} else {
			log.ProviderRequestMode = log.RequestMode
		}
	}
	if r.providerRequestMode == "" && log.ProviderRequestMode != "" {
		r.providerRequestMode = log.ProviderRequestMode
	}
	r.thirdPartyLogs = append(r.thirdPartyLogs, log)
}

func (r *FlowRecorder) ResolvedModel() string {
	if r == nil {
		return ""
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.resolvedModel
}

func (r *FlowRecorder) Snapshot(completedAt time.Time) (airequestlog.RunRecord, airequestlog.FlowRecord, []airequestlog.ThirdPartyRequestLogRecord) {
	if r == nil {
		return airequestlog.RunRecord{}, airequestlog.FlowRecord{}, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}

	attemptTrace := marshalJSON(r.attemptTrace)
	durationMs := completedAt.Sub(r.startedAt).Milliseconds()
	run := airequestlog.RunRecord{
		RequestID:           r.requestID,
		Type:                r.requestType,
		RequestMode:         r.requestMode,
		ProviderRequestMode: defaultString(r.providerRequestMode, r.requestMode),
		Method:              r.method,
		Path:                r.path,
		RequestedModel:      r.requestedModel,
		ResolvedModel:       r.resolvedModel,
		ProviderID:          r.providerID,
		ProviderName:        r.providerName,
		FinalConnectionID:   r.finalConnectionID,
		FinalConnectionName: r.finalConnectionName,
		AttemptCount:        len(r.attemptTrace),
		StatusCode:          r.responseStatusCode,
		PromptTokens:        r.promptTokens,
		CompletionTokens:    r.completionTokens,
		TotalTokens:         r.totalTokens,
		ErrorType:           r.errorType,
		ErrorMessage:        r.errorMessage,
		FinalErrorCategory:  r.finalErrorCategory,
		StartedAt:           r.startedAt.UnixMilli(),
		CompletedAt:         completedAt.UnixMilli(),
		DurationMs:          durationMs,
	}

	flow := airequestlog.FlowRecord{
		RequestID:              r.requestID,
		Type:                   r.requestType,
		RequestMode:            r.requestMode,
		ProviderRequestMode:    defaultString(r.providerRequestMode, r.requestMode),
		Method:                 r.method,
		Path:                   r.path,
		Query:                  r.query,
		RemoteAddr:             r.remoteAddr,
		UserAgent:              r.userAgent,
		RequestHeaders:         r.requestHeaders,
		RequestBody:            r.requestBody,
		TranslatedRequestBody:  r.translatedRequestBody,
		RequestedModel:         r.requestedModel,
		ResolvedModel:          r.resolvedModel,
		ProviderID:             r.providerID,
		ProviderName:           r.providerName,
		AttemptTrace:           attemptTrace,
		ResponseStatusCode:     r.responseStatusCode,
		ResponseHeaders:        r.responseHeaders,
		ResponseBody:           r.responseBody,
		TranslatedResponseBody: r.translatedResponseBody,
		ErrorType:              r.errorType,
		ErrorMessage:           r.errorMessage,
		StartedAt:              r.startedAt.UnixMilli(),
		CompletedAt:            completedAt.UnixMilli(),
		DurationMs:             durationMs,
	}

	thirdPartyLogs := make([]airequestlog.ThirdPartyRequestLogRecord, 0, len(r.thirdPartyLogs))
	for _, current := range r.thirdPartyLogs {
		thirdPartyLogs = append(thirdPartyLogs, airequestlog.ThirdPartyRequestLogRecord{
			FlowRequestID:       current.FlowRequestID,
			Type:                current.Type,
			RequestMode:         current.RequestMode,
			ProviderRequestMode: defaultString(current.ProviderRequestMode, current.RequestMode),
			ProviderID:          current.ProviderID,
			ProviderName:        current.ProviderName,
			ConnectionID:        current.ConnectionID,
			ConnectionName:      current.ConnectionName,
			AttemptIndex:        current.AttemptIndex,
			RequestMethod:       current.RequestMethod,
			RequestURL:          current.RequestURL,
			RequestHeaders:      current.RequestHeaders,
			RequestBody:         current.RequestBody,
			ResponseStatusCode:  current.ResponseStatusCode,
			ResponseHeaders:     current.ResponseHeaders,
			ResponseBody:        current.ResponseBody,
			ErrorType:           current.ErrorType,
			ErrorMessage:        current.ErrorMessage,
			StartedAt:           current.StartedAt.UnixMilli(),
			CompletedAt:         current.CompletedAt.UnixMilli(),
			DurationMs:          current.CompletedAt.Sub(current.StartedAt).Milliseconds(),
		})
	}

	return run, flow, thirdPartyLogs
}

func CaptureStream(body io.ReadCloser, finalize func([]byte, error)) io.ReadCloser {
	if body == nil {
		return nil
	}

	return &capturedStreamBody{body: body, finalize: finalize}
}

type capturedStreamBody struct {
	body     io.ReadCloser
	finalize func([]byte, error)
	buf      []byte
	done     bool
}

func (b *capturedStreamBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if n > 0 {
		b.buf = append(b.buf, p[:n]...)
	}
	if err != nil {
		b.finish(err)
	}
	return n, err
}

func (b *capturedStreamBody) Close() error {
	err := b.body.Close()
	b.finish(err)
	return err
}

func (b *capturedStreamBody) finish(err error) {
	if b.done {
		return
	}
	b.done = true
	if b.finalize == nil {
		return
	}
	if err == io.EOF {
		err = nil
	}
	b.finalize(slices.Clone(b.buf), err)
}

func marshalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func marshalHeaders(headers http.Header, redact bool) string {
	if len(headers) == 0 {
		return "{}"
	}

	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		nextValues := append([]string(nil), values...)
		if redact && isSensitiveKey(key) {
			nextValues = []string{"[REDACTED]"}
		}
		cloned[key] = nextValues
	}

	return marshalJSON(cloned)
}

func redactBody(body string) string {
	if strings.TrimSpace(body) == "" {
		return body
	}

	var payload any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return body
	}

	return marshalJSON(redactValue(payload))
}

func RedactHeadersForStorage(headers http.Header) string {
	return marshalHeaders(headers, true)
}

func RedactBodyForStorage(body string) string {
	return redactBody(body)
}

func ThirdPartyResponseBodyForStorage(headers http.Header, body string) string {
	if isEventStream(headers) {
		return ThirdPartySSEResponseBodyForStorage(body)
	}
	return redactBody(body)
}

func ThirdPartySSEResponseBodyForStorage(body string) string {
	return redactBody(lastSSEEvent(body))
}

func responseBodyForStorage(requestMode string, body string) string {
	if requestMode != RequestModeStream {
		return body
	}

	return lastSSEEvent(body)
}

func lastSSEEvent(body string) string {
	if strings.TrimSpace(body) == "" {
		return body
	}

	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	parts := strings.Split(normalized, "\n\n")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		lines := strings.Split(part, "\n")
		dataLines := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data:") {
				continue
			}

			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}
			dataLines = append(dataLines, payload)
		}
		if len(dataLines) > 0 {
			return strings.Join(dataLines, "\n")
		}
	}

	return body
}

func isEventStream(headers http.Header) bool {
	if len(headers) == 0 {
		return false
	}
	return strings.Contains(strings.ToLower(headers.Get("Content-Type")), "text/event-stream")
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, current := range typed {
			if isSensitiveKey(key) {
				typed[key] = "[REDACTED]"
				continue
			}
			typed[key] = redactValue(current)
		}
		return typed
	case []any:
		for i := range typed {
			typed[i] = redactValue(typed[i])
		}
		return typed
	default:
		return value
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func isSensitiveKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "authorization", "cookie", "set-cookie", "api_key", "apikey", "access_token", "refreshtoken", "refresh_token":
		return true
	default:
		return false
	}
}

func resolvedModel(target routing.Target) string {
	if target.Prefix == "" || target.RequestedModel == "" {
		return target.RequestedModel
	}
	return target.Prefix + "/" + target.RequestedModel
}

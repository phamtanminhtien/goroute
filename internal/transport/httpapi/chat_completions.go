package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	"github.com/rs/zerolog"
)

func chatCompletionsHandler(catalog provider.Catalog, connectionRegistry *chatcompletion.ConnectionRegistry, requestLogRepo aiRequestLogRepository, logger *zerolog.Logger) http.Handler {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now().UTC()
		recorder := chatcompletion.NewFlowRecorder(chatcompletion.RequestID(r.Context()), startedAt)
		ctx := chatcompletion.WithFlowRecorder(r.Context(), recorder)
		r = r.WithContext(ctx)

		bodyWriter := newBodyCaptureResponseWriter(w)
		defer persistAIRequestLog(requestLogRepo, logger, recorder, bodyWriter)

		if r.Method != http.MethodPost {
			recorder.SetError("method_not_allowed", "method not allowed")
			writeError(r, bodyWriter, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		defer r.Body.Close()

		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			recorder.ConfigureInbound(r, nil)
			recorder.SetError("invalid_request", fmt.Sprintf("read request body: %v", err))
			writeError(r, bodyWriter, http.StatusBadRequest, "invalid_request", fmt.Sprintf("read request body: %v", err))
			return
		}
		recorder.ConfigureInbound(r, rawBody)

		var request openaiwire.ChatCompletionsRequest
		if err := json.Unmarshal(rawBody, &request); err != nil {
			recorder.SetError("invalid_request", fmt.Sprintf("invalid JSON body: %v", err))
			writeError(r, bodyWriter, http.StatusBadRequest, "invalid_request", fmt.Sprintf("invalid JSON body: %v", err))
			return
		}

		if request.Stream {
			recorder.SetRequestMode(true)
			output, err := chatcompletion.ExecuteStream(r.Context(), catalog, connectionRegistry, chatcompletion.Input{Request: request})
			if err != nil {
				var upstreamErr chatcompletion.UpstreamError
				switch {
				case errors.As(err, &upstreamErr):
					recorder.SetError("upstream_error", upstreamErr.Error())
					writeUpstreamError(bodyWriter, upstreamErr)
				default:
					recorder.SetError("invalid_request", err.Error())
					writeError(r, bodyWriter, http.StatusBadRequest, "invalid_request", err.Error())
				}
				return
			}
			defer output.Body.Close()

			bodyWriter.Header().Set("Content-Type", "text/event-stream")
			bodyWriter.Header().Set("Cache-Control", "no-cache")
			bodyWriter.Header().Set("Connection", "keep-alive")
			bodyWriter.Header().Set("Access-Control-Allow-Origin", "*")
			bodyWriter.WriteHeader(http.StatusOK)
			if _, err := io.Copy(bodyWriter, output.Body); err != nil {
				recorder.SetError("stream_error", err.Error())
			}
			return
		}

		recorder.SetRequestMode(false)
		output, err := chatcompletion.Execute(r.Context(), catalog, connectionRegistry, chatcompletion.Input{Request: request})
		if err != nil {
			var upstreamErr chatcompletion.UpstreamError
			switch {
			case errors.As(err, &upstreamErr):
				recorder.SetError("upstream_error", upstreamErr.Error())
				writeUpstreamError(bodyWriter, upstreamErr)
			default:
				recorder.SetError("invalid_request", err.Error())
				writeError(r, bodyWriter, http.StatusBadRequest, "invalid_request", err.Error())
			}
			return
		}

		recorder.SetFlowResponse(output.Response, false)
		writeJSON(bodyWriter, http.StatusOK, output.Response)
	})
}

func persistAIRequestLog(repo aiRequestLogRepository, logger *zerolog.Logger, recorder *chatcompletion.FlowRecorder, bodyWriter *bodyCaptureResponseWriter) {
	if repo == nil || recorder == nil || bodyWriter == nil {
		return
	}

	recorder.SetHTTPResponse(bodyWriter.statusCode, bodyWriter.headerSnapshot(), bodyWriter.bodyString())
	runRecord, flowRecord, thirdPartyLogs := recorder.Snapshot(time.Now().UTC())
	if err := repo.CreateAIRequestRun(runRecord); err != nil {
		logger.Error().Err(err).Str("request_id", runRecord.RequestID).Msg("persist_ai_request_run_failed")
	}
	if err := repo.CreateAIRequestFlow(flowRecord); err != nil {
		logger.Error().Err(err).Str("request_id", flowRecord.RequestID).Msg("persist_ai_request_flow_failed")
	}
	for _, current := range thirdPartyLogs {
		if err := repo.CreateThirdPartyRequestLog(current); err != nil {
			logger.Error().Err(err).Str("request_id", flowRecord.RequestID).Msg("persist_third_party_request_log_failed")
		}
	}
}

type bodyCaptureResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func newBodyCaptureResponseWriter(w http.ResponseWriter) *bodyCaptureResponseWriter {
	return &bodyCaptureResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (w *bodyCaptureResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *bodyCaptureResponseWriter) Write(body []byte) (int, error) {
	w.body.Write(body)
	return w.ResponseWriter.Write(body)
}

func (w *bodyCaptureResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *bodyCaptureResponseWriter) headerSnapshot() http.Header {
	return w.Header().Clone()
}

func (w *bodyCaptureResponseWriter) bodyString() string {
	return w.body.String()
}

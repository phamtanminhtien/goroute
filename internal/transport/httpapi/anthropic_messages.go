package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/config"
	anthropicusecase "github.com/phamtanminhtien/goroute/internal/usecase/anthropic"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	"github.com/rs/zerolog"
)

func anthropicMessagesHandler(catalog catalogSource, connectionRegistry *chatcompletion.ConnectionRegistry, requestLogRepo aiRequestLogRepository, modelRepo providerModelRepository, modelComboRepo modelComboRepository, settingsManager *config.SettingsManager, logger *zerolog.Logger) http.Handler {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now().UTC()
		recorder := chatcompletion.NewFlowRecorder(chatcompletion.RequestID(r.Context()), startedAt)
		ctx := chatcompletion.WithFlowRecorder(r.Context(), recorder)
		ctx = config.WithSettingsManager(ctx, settingsManager)
		r = r.WithContext(ctx)

		bodyWriter := newBodyCaptureResponseWriter(w)
		defer persistAIRequestLog(requestLogRepo, settingsManager, logger, recorder, bodyWriter)

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

		var request anthropicwire.MessagesRequest
		if err := json.Unmarshal(rawBody, &request); err != nil {
			recorder.SetError("invalid_request", fmt.Sprintf("invalid JSON body: %v", err))
			writeError(r, bodyWriter, http.StatusBadRequest, "invalid_request", fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
		request.RawBody = append(request.RawBody[:0], rawBody...)

		resolvedCatalog, combos, err := routingInputs(catalog.Catalog(), modelRepo, modelComboRepo)
		if err != nil {
			recorder.SetError("internal_error", err.Error())
			writeError(r, bodyWriter, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		if request.Stream {
			recorder.SetRequestMode(true)
			output, err := anthropicusecase.ExecuteStream(r.Context(), resolvedCatalog, combos, connectionRegistry, anthropicusecase.Input{Request: request})
			if err != nil {
				writeAnthropicExecutionError(r, bodyWriter, recorder, err)
				return
			}
			defer output.Body.Close()

			bodyWriter.Header().Set("Content-Type", "text/event-stream")
			bodyWriter.Header().Set("Cache-Control", "no-cache")
			bodyWriter.Header().Set("Connection", "keep-alive")
			bodyWriter.Header().Set("Access-Control-Allow-Origin", "*")
			bodyWriter.WriteHeader(http.StatusOK)
			if err := writeSSEStream(bodyWriter, output.Body); err != nil {
				recorder.SetError("stream_error", err.Error())
			} else {
				recorder.SetTranslatedSSEResponseBody(bodyWriter.bodyString())
			}
			return
		}

		recorder.SetRequestMode(false)
		output, err := anthropicusecase.Execute(r.Context(), resolvedCatalog, combos, connectionRegistry, anthropicusecase.Input{Request: request})
		if err != nil {
			writeAnthropicExecutionError(r, bodyWriter, recorder, err)
			return
		}

		writeJSON(bodyWriter, http.StatusOK, output.Response)
	})
}

func writeAnthropicExecutionError(r *http.Request, w http.ResponseWriter, recorder *chatcompletion.FlowRecorder, err error) {
	var upstreamErr chatcompletion.UpstreamError
	switch {
	case errors.As(err, &upstreamErr):
		recorder.SetError("upstream_error", upstreamErr.Error())
		writeUpstreamError(w, upstreamErr)
	default:
		recorder.SetError("invalid_request", err.Error())
		writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}

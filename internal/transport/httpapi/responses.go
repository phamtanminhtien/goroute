package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	responsesusecase "github.com/phamtanminhtien/goroute/internal/usecase/responses"
	"github.com/rs/zerolog"
)

func responsesHandler(catalog provider.Catalog, connectionRegistry *chatcompletion.ConnectionRegistry, requestLogRepo aiRequestLogRepository, modelRepo providerModelRepository, modelComboRepo modelComboRepository, settingsManager *config.SettingsManager, logger *zerolog.Logger) http.Handler {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now().UTC()
		recorder := chatcompletion.NewFlowRecorder(chatcompletion.RequestID(r.Context()), startedAt)
		recorder.SetRequestType(chatcompletion.RequestTypeResponses)
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

		var request openaiwire.ResponsesRequest
		if err := json.Unmarshal(rawBody, &request); err != nil {
			recorder.SetError("invalid_request", fmt.Sprintf("invalid JSON body: %v", err))
			writeError(r, bodyWriter, http.StatusBadRequest, "invalid_request", fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
		request.RawBody = append(request.RawBody[:0], rawBody...)

		if request.Stream {
			resolvedCatalog, combos, err := routingInputs(catalog, modelRepo, modelComboRepo)
			if err != nil {
				recorder.SetError("internal_error", err.Error())
				writeError(r, bodyWriter, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			recorder.SetRequestMode(true)
			output, err := responsesusecase.ExecuteStream(r.Context(), resolvedCatalog, combos, connectionRegistry, responsesusecase.Input{Request: request})
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

		resolvedCatalog, combos, err := routingInputs(catalog, modelRepo, modelComboRepo)
		if err != nil {
			recorder.SetError("internal_error", err.Error())
			writeError(r, bodyWriter, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		recorder.SetRequestMode(false)
		output, err := responsesusecase.Execute(r.Context(), resolvedCatalog, combos, connectionRegistry, responsesusecase.Input{Request: request})
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

		recorder.SetResponsesResponse(output.Response, true)
		writeJSON(bodyWriter, http.StatusOK, output.Response)
	})
}

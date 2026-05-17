package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/config"
)

type settingsResponse struct {
	Server     settingsServerResponse     `json:"server"`
	LLMLogging settingsLLMLoggingResponse `json:"llmLogging"`
}

type settingsServerResponse struct {
	Listen   string `json:"listen"`
	WebUIDir string `json:"web_ui_dir"`
}

type settingsLLMLoggingResponse struct {
	Enabled settingsLLMLoggingEnabledResponse `json:"enabled"`
}

type settingsLLMLoggingEnabledResponse struct {
	Flow       bool `json:"flow"`
	ThirdParty bool `json:"thirdParty"`
}

type updateSettingsRequest struct {
	LLMLogging *updateSettingsLLMLoggingRequest `json:"llmLogging"`
}

type updateSettingsLLMLoggingRequest struct {
	Enabled *updateSettingsLLMLoggingEnabledRequest `json:"enabled"`
}

type updateSettingsLLMLoggingEnabledRequest struct {
	Flow       *bool `json:"flow"`
	ThirdParty *bool `json:"thirdParty"`
}

func settingsHandler(settingsManager *config.SettingsManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if settingsManager == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "settings manager is not configured")
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, buildSettingsResponse(settingsManager.Snapshot()))
		case http.MethodPut:
			var input updateSettingsRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			if input.LLMLogging == nil || input.LLMLogging.Enabled == nil || input.LLMLogging.Enabled.Flow == nil || input.LLMLogging.Enabled.ThirdParty == nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "llmLogging.enabled.flow and llmLogging.enabled.thirdParty are required")
				return
			}

			cfg, err := settingsManager.UpdateLLMLogging(config.LLMLoggingState{
				FlowEnabled:       *input.LLMLogging.Enabled.Flow,
				ThirdPartyEnabled: *input.LLMLogging.Enabled.ThirdParty,
			})
			if err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}

			writeJSON(w, http.StatusOK, buildSettingsResponse(cfg))
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func buildSettingsResponse(cfg config.Config) settingsResponse {
	return settingsResponse{
		Server: settingsServerResponse{
			Listen:   cfg.Server.Listen,
			WebUIDir: cfg.Server.WebUIDir,
		},
		LLMLogging: settingsLLMLoggingResponse{
			Enabled: settingsLLMLoggingEnabledResponse{
				Flow:       cfg.LLMLogging.Flow,
				ThirdParty: cfg.LLMLogging.ThirdParty,
			},
		},
	}
}

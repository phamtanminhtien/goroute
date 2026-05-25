package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
	providersusecase "github.com/phamtanminhtien/goroute/internal/usecase/providers"
)

type adminProvidersListResponse struct {
	Object string              `json:"object"`
	Data   []adminProviderItem `json:"data"`
}

type adminProviderItem struct {
	ID                     string                          `json:"id"`
	Name                   string                          `json:"name"`
	AuthType               provider.AuthType               `json:"auth_type"`
	Category               string                          `json:"category"`
	AdapterType            provider.AdapterType            `json:"adapter_type,omitempty"`
	BaseURL                string                          `json:"base_url,omitempty"`
	ConnectionCount        int                             `json:"connection_count"`
	EnabledConnectionCount int                             `json:"enabled_connection_count"`
	DefaultModel           string                          `json:"default_model"`
	Models                 []provider.Model                `json:"models"`
	Connections            []connectionsusecase.Item       `json:"connections"`
	RuntimeSettings        providerRuntimeSettingsResponse `json:"runtime_settings"`
}

type providerRuntimeSettingsResponse struct {
	DialTimeoutMs           int `json:"dial_timeout_ms"`
	TLSHandshakeTimeoutMs   int `json:"tls_handshake_timeout_ms"`
	ResponseHeaderTimeoutMs int `json:"response_header_timeout_ms"`
	TimeoutRetryCount       int `json:"timeout_retry_count"`
	RetryableCooldownMs     int `json:"retryable_cooldown_ms"`
}

type providerModelRepository interface {
	ListProviderModels() ([]provider.ModelRecord, error)
	CreateProviderModel(provider.ModelRecord) error
	UpdateProviderModel(providerID string, modelID string, record provider.ModelRecord) error
	DeleteProviderModel(providerID string, modelID string) error
}

type providerMutationService interface {
	Create(providersusecase.MutationInput) (provider.Provider, error)
	Update(string, providersusecase.MutationInput) (provider.Provider, error)
	Delete(string) error
}

type providerMutationInput struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	AdapterType  provider.AdapterType `json:"adapter_type"`
	BaseURL      string               `json:"base_url"`
	DefaultModel string               `json:"default_model"`
	APIKey       string               `json:"api_key"`
	Enabled      *bool                `json:"enabled"`
}

type providerConnectionsEnabledRequest struct {
	Enabled *bool `json:"enabled"`
}

type providerRuntimeSettingsRequest struct {
	DialTimeoutMs           *int `json:"dial_timeout_ms"`
	TLSHandshakeTimeoutMs   *int `json:"tls_handshake_timeout_ms"`
	ResponseHeaderTimeoutMs *int `json:"response_header_timeout_ms"`
	TimeoutRetryCount       *int `json:"timeout_retry_count"`
	RetryableCooldownMs     *int `json:"retryable_cooldown_ms"`
}

func providersHandler(
	catalog catalogSource,
	connectionService *connectionsusecase.Service,
	providerService providerMutationService,
	modelRepo providerModelRepository,
	settingsManager *config.SettingsManager,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			resolvedCatalog, err := catalogWithCustomModels(catalog.Catalog(), modelRepo)
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}

			writeJSON(w, http.StatusOK, adminProvidersListResponse{
				Object: "list",
				Data:   buildAdminProviderItems(resolvedCatalog, connectionService.List(), settingsManager),
			})
		case http.MethodPost:
			if providerService == nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", "provider service is not configured")
				return
			}
			var input providerMutationInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}
			enabled := true
			if input.Enabled != nil {
				enabled = *input.Enabled
			}
			created, err := providerService.Create(providersusecase.MutationInput{
				ID:           input.ID,
				Name:         input.Name,
				AdapterType:  input.AdapterType,
				BaseURL:      input.BaseURL,
				DefaultModel: input.DefaultModel,
				APIKey:       input.APIKey,
				Enabled:      enabled,
			})
			if err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, created)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func providerByIDHandler(providerService providerMutationService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerID := strings.TrimSpace(chi.URLParam(r, "id"))
		if providerID == "" || strings.Contains(providerID, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}

		switch r.Method {
		case http.MethodPut:
			if providerService == nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", "provider service is not configured")
				return
			}
			var input providerMutationInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}
			enabled := true
			if input.Enabled != nil {
				enabled = *input.Enabled
			}
			updated, err := providerService.Update(providerID, providersusecase.MutationInput{
				ID:           providerID,
				Name:         input.Name,
				AdapterType:  input.AdapterType,
				BaseURL:      input.BaseURL,
				DefaultModel: input.DefaultModel,
				APIKey:       input.APIKey,
				Enabled:      enabled,
			})
			if err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			if providerService == nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", "provider service is not configured")
				return
			}
			if err := providerService.Delete(providerID); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func providerConnectionsEnabledHandler(
	catalog catalogSource,
	connectionService *connectionsusecase.Service,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		providerID := strings.TrimSpace(chi.URLParam(r, "id"))
		if providerID == "" || strings.Contains(providerID, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		resolvedCatalog := catalog.Catalog()
		if resolvedProvider, ok := resolvedCatalog.FindByID(providerID); !ok {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		} else if resolvedProvider.Category == "custom" {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "connections for custom providers are managed with the provider")
			return
		}

		var input providerConnectionsEnabledRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}
		if input.Enabled == nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "enabled is required")
			return
		}

		connections, err := connectionService.SetProviderConnectionsEnabled(providerID, *input.Enabled)
		if err != nil {
			writeConnectionMutationError(r, w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data":   connections,
		})
	})
}

func providerRuntimeSettingsHandler(
	catalog catalogSource,
	connectionService *connectionsusecase.Service,
	settingsManager *config.SettingsManager,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodDelete {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		if settingsManager == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "settings manager is not configured")
			return
		}

		providerID := strings.TrimSpace(chi.URLParam(r, "id"))
		if providerID == "" || strings.Contains(providerID, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if _, ok := catalog.Catalog().FindByID(providerID); !ok {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}

		var cfg config.Config
		var err error
		switch r.Method {
		case http.MethodPut:
			var input providerRuntimeSettingsRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}
			if input.DialTimeoutMs == nil || input.TLSHandshakeTimeoutMs == nil || input.ResponseHeaderTimeoutMs == nil || input.TimeoutRetryCount == nil || input.RetryableCooldownMs == nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "dial_timeout_ms, tls_handshake_timeout_ms, response_header_timeout_ms, timeout_retry_count, and retryable_cooldown_ms are required")
				return
			}
			cfg, err = settingsManager.UpdateProviderRuntime(providerID, config.ProviderRuntimeState{
				DialTimeoutMs:           *input.DialTimeoutMs,
				TLSHandshakeTimeoutMs:   *input.TLSHandshakeTimeoutMs,
				ResponseHeaderTimeoutMs: *input.ResponseHeaderTimeoutMs,
				TimeoutRetryCount:       *input.TimeoutRetryCount,
				RetryableCooldownMs:     *input.RetryableCooldownMs,
			})
		case http.MethodDelete:
			cfg, err = settingsManager.ResetProviderRuntime(providerID)
		}
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if connectionService != nil {
			if err := connectionService.ReloadRuntime(); err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
		}

		writeJSON(w, http.StatusOK, buildProviderRuntimeSettingsResponse(config.EffectiveProviderRuntimeSettings(cfg, providerID)))
	})
}

type providerModelCreateRequest struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	InputPricePerMillionUSD  float64 `json:"input_price_per_million_usd"`
	OutputPricePerMillionUSD float64 `json:"output_price_per_million_usd"`
}

func providerModelsHandler(catalog catalogSource, modelRepo providerModelRepository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		if modelRepo == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "provider model repository is not configured")
			return
		}

		providerID := strings.TrimSpace(chi.URLParam(r, "id"))
		if providerID == "" || strings.Contains(providerID, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if _, ok := catalog.Catalog().FindByID(providerID); !ok {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}

		var input providerModelCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}

		record, err := buildProviderModelRecord(providerID, input)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if err := modelRepo.CreateProviderModel(record); err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, record)
	})
}

func providerModelByIDHandler(catalog catalogSource, modelRepo providerModelRepository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodDelete {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		if modelRepo == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "provider model repository is not configured")
			return
		}

		providerID := strings.TrimSpace(chi.URLParam(r, "id"))
		modelID := strings.Trim(strings.TrimSpace(chi.URLParam(r, "*")), "/")
		if providerID == "" || strings.Contains(providerID, "/") || modelID == "" {
			writeError(r, w, http.StatusNotFound, "not_found", "provider model not found")
			return
		}
		if _, ok := catalog.Catalog().FindByID(providerID); !ok {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}
		if !customModelExists(modelRepo, providerID, modelID) {
			writeError(r, w, http.StatusNotFound, "not_found", "custom model not found")
			return
		}

		switch r.Method {
		case http.MethodPut:
			var input providerModelCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			record, err := buildProviderModelRecord(providerID, input)
			if err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			if err := modelRepo.UpdateProviderModel(providerID, modelID, record); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}

			writeJSON(w, http.StatusOK, record)
		case http.MethodDelete:
			if err := modelRepo.DeleteProviderModel(providerID, modelID); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	})
}

type providerOAuthURLResponse struct {
	ProviderID string `json:"provider_id"`
	SessionID  string `json:"session_id"`
	URL        string `json:"url"`
}

type providerOAuthStarter interface {
	StartOAuth(providerID string) (connectionsusecase.OAuthStartResult, error)
}

func providerOAuthURLHandler(starter providerOAuthStarter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		providerID := strings.TrimSpace(chi.URLParam(r, "id"))
		if providerID == "" || strings.Contains(providerID, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}

		started, err := starter.StartOAuth(providerID)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, providerOAuthURLResponse{
			ProviderID: providerID,
			SessionID:  started.SessionID,
			URL:        started.AuthorizationURL,
		})
	})
}

type providerModelTester interface {
	ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error)
}

type providerModelTestRequest struct {
	Model string `json:"model"`
}

type providerModelTestResponse struct {
	ProviderID string `json:"provider_id"`
	Model      string `json:"model"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	OutputText string `json:"output_text,omitempty"`
}

func providerModelTestHandler(
	catalog catalogSource,
	tester providerModelTester,
	modelRepo providerModelRepository,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		providerID := strings.TrimSpace(chi.URLParam(r, "id"))
		if providerID == "" || strings.Contains(providerID, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}

		resolvedCatalog, err := catalogWithCustomModels(catalog.Catalog(), modelRepo)
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		resolvedProvider, ok := resolvedCatalog.FindByID(providerID)
		if !ok {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}

		var input providerModelTestRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}

		modelID := strings.TrimSpace(input.Model)
		if modelID == "" {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "model is required")
			return
		}
		if !providerHasModel(resolvedProvider, modelID) {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "model does not belong to provider")
			return
		}
		target, err := routing.ResolveModel(resolvedCatalog, modelID)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if target.ProviderID != providerID {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "model does not belong to provider")
			return
		}

		response, err := tester.ChatCompletions(r.Context(), openaiwire.ChatCompletionsRequest{
			Model: modelID,
			Messages: []openaiwire.ChatMessage{{
				Role:    openaiwire.ChatRoleUser,
				Content: openaiwire.TextContent("Reply with OK if this test request reaches the model."),
			}},
		}, target)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		outputText := firstCompletionText(response)
		writeJSON(w, http.StatusOK, providerModelTestResponse{
			ProviderID: resolvedProvider.ID,
			Model:      modelID,
			Status:     "success",
			Message:    "Model test succeeded.",
			OutputText: outputText,
		})
	})
}

func providerHasModel(resolvedProvider provider.Provider, modelID string) bool {
	if resolvedProvider.DefaultModel == modelID {
		return true
	}

	for _, model := range resolvedProvider.Models {
		if model.ID == modelID {
			return true
		}
	}

	return false
}

func catalogWithCustomModels(catalog provider.Catalog, modelRepo providerModelRepository) (provider.Catalog, error) {
	if modelRepo == nil {
		return catalog, nil
	}

	records, err := modelRepo.ListProviderModels()
	if err != nil {
		return provider.Catalog{}, err
	}

	return catalog.WithModelRecords(records), nil
}

func customModelExists(modelRepo providerModelRepository, providerID string, modelID string) bool {
	records, err := modelRepo.ListProviderModels()
	if err != nil {
		return false
	}

	for _, record := range records {
		if record.ProviderID == providerID && record.ID == modelID {
			return true
		}
	}

	return false
}

func buildProviderModelRecord(providerID string, input providerModelCreateRequest) (provider.ModelRecord, error) {
	modelID := strings.TrimSpace(input.ID)
	if modelID == "" {
		return provider.ModelRecord{}, fmt.Errorf("model id is required")
	}
	prefix := providerID + "/"
	if !strings.HasPrefix(modelID, prefix) || strings.TrimSpace(strings.TrimPrefix(modelID, prefix)) == "" {
		return provider.ModelRecord{}, fmt.Errorf("model id must start with provider prefix %q", prefix)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = strings.TrimPrefix(modelID, prefix)
	}

	return provider.ModelRecord{
		ID:                       modelID,
		ProviderID:               providerID,
		Name:                     name,
		Description:              strings.TrimSpace(input.Description),
		InputPricePerMillionUSD:  input.InputPricePerMillionUSD,
		OutputPricePerMillionUSD: input.OutputPricePerMillionUSD,
	}, nil
}

func firstCompletionText(response openaiwire.ChatCompletionsResponse) string {
	if len(response.Choices) == 0 {
		return ""
	}

	return strings.TrimSpace(response.Choices[0].Message.Content)
}

func buildAdminProviderItems(
	catalog provider.Catalog,
	connections []connectionsusecase.Item,
	settingsManager *config.SettingsManager,
) []adminProviderItem {
	groupedConnections := make(map[string][]connectionsusecase.Item, len(connections))
	for _, connection := range connections {
		groupedConnections[connection.ProviderID] = append(
			groupedConnections[connection.ProviderID],
			connection,
		)
	}

	items := make([]adminProviderItem, 0, len(catalog.Providers))
	for _, providerItem := range catalog.Providers {
		providerConnections := groupedConnections[providerItem.ID]
		enabledConnectionCount := 0
		for _, connection := range providerConnections {
			if connection.Enabled {
				enabledConnectionCount++
			}
		}
		models := modelsWithSource(providerItem.Models)
		if len(models) == 0 && providerItem.DefaultModel != "" {
			models = []provider.Model{{
				ID:     providerItem.DefaultModel,
				Name:   strings.TrimPrefix(providerItem.DefaultModel, providerItem.ID+"/"),
				Source: "system",
			}}
		}
		items = append(items, adminProviderItem{
			ID:                     providerItem.ID,
			Name:                   providerItem.Name,
			AuthType:               providerItem.AuthType,
			Category:               providerItem.Category,
			AdapterType:            providerItem.AdapterType,
			BaseURL:                providerItem.BaseURL,
			ConnectionCount:        len(providerConnections),
			EnabledConnectionCount: enabledConnectionCount,
			DefaultModel:           providerItem.DefaultModel,
			Models:                 models,
			Connections:            providerConnections,
			RuntimeSettings:        buildProviderRuntimeSettingsResponse(effectiveProviderRuntimeSettings(settingsManager, providerItem.ID)),
		})
	}

	return items
}

func effectiveProviderRuntimeSettings(settingsManager *config.SettingsManager, providerID string) config.ProviderRuntimeSettings {
	if settingsManager == nil {
		return config.DefaultProviderRuntimeSettings()
	}
	return config.EffectiveProviderRuntimeSettings(settingsManager.Snapshot(), providerID)
}

func buildProviderRuntimeSettingsResponse(settings config.ProviderRuntimeSettings) providerRuntimeSettingsResponse {
	return providerRuntimeSettingsResponse{
		DialTimeoutMs:           settings.DialTimeoutMs,
		TLSHandshakeTimeoutMs:   settings.TLSHandshakeTimeoutMs,
		ResponseHeaderTimeoutMs: settings.ResponseHeaderTimeoutMs,
		TimeoutRetryCount:       settings.TimeoutRetryCount,
		RetryableCooldownMs:     settings.RetryableCooldownMs,
	}
}

func modelsWithSource(models []provider.Model) []provider.Model {
	resolved := make([]provider.Model, len(models))
	for index, model := range models {
		if model.Source == "" {
			model.Source = "system"
		}
		resolved[index] = model
	}

	return resolved
}

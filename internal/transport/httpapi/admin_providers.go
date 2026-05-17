package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

type adminProvidersListResponse struct {
	Object string              `json:"object"`
	Data   []adminProviderItem `json:"data"`
}

type adminProviderItem struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	AuthType        provider.AuthType         `json:"auth_type"`
	Category        string                    `json:"category"`
	ConnectionCount int                       `json:"connection_count"`
	DefaultModel    string                    `json:"default_model"`
	Models          []provider.Model          `json:"models"`
	Connections     []connectionsusecase.Item `json:"connections"`
}

func providersHandler(
	catalog provider.Catalog,
	connectionService *connectionsusecase.Service,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		writeJSON(w, http.StatusOK, adminProvidersListResponse{
			Object: "list",
			Data:   buildAdminProviderItems(catalog, connectionService.List()),
		})
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
	catalog provider.Catalog,
	tester providerModelTester,
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

		resolvedProvider, ok := catalog.FindByID(providerID)
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

		response, err := tester.ChatCompletions(r.Context(), openaiwire.ChatCompletionsRequest{
			Model: modelID,
			Messages: []openaiwire.ChatMessage{{
				Role:    openaiwire.ChatRoleUser,
				Content: openaiwire.TextContent("Reply with OK if this test request reaches the model."),
			}},
		}, routing.Target{
			Prefix:         providerID,
			RequestedModel: modelID,
			ProviderID:     resolvedProvider.ID,
			ProviderName:   resolvedProvider.Name,
		})
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

func firstCompletionText(response openaiwire.ChatCompletionsResponse) string {
	if len(response.Choices) == 0 {
		return ""
	}

	return strings.TrimSpace(response.Choices[0].Message.Content)
}

func buildAdminProviderItems(
	catalog provider.Catalog,
	connections []connectionsusecase.Item,
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
		items = append(items, adminProviderItem{
			ID:              providerItem.ID,
			Name:            providerItem.Name,
			AuthType:        providerItem.AuthType,
			Category:        providerItem.Category,
			ConnectionCount: len(providerConnections),
			DefaultModel:    providerItem.DefaultModel,
			Models:          providerItem.Models,
			Connections:     providerConnections,
		})
	}

	return items
}

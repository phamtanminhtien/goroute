package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
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

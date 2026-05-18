package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
	"gorm.io/gorm"
)

type modelComboRepository interface {
	ListModelCombos() ([]modelcombo.Combo, error)
	GetModelCombo(alias string) (modelcombo.Combo, bool, error)
	CreateModelCombo(modelcombo.Combo) error
	UpdateModelCombo(alias string, combo modelcombo.Combo) error
	DeleteModelCombo(alias string) error
}

type modelCombosListResponse struct {
	Object string             `json:"object"`
	Data   []modelcombo.Combo `json:"data"`
}

type modelComboPayload struct {
	Alias       string                    `json:"alias"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Enabled     *bool                     `json:"enabled"`
	Targets     []modelComboTargetPayload `json:"targets"`
}

type modelComboTargetPayload struct {
	ProviderID   string `json:"provider_id"`
	ConnectionID string `json:"connection_id"`
	ModelID      string `json:"model_id"`
	Enabled      *bool  `json:"enabled"`
}

func modelCombosHandler(catalog provider.Catalog, modelRepo providerModelRepository, comboRepo modelComboRepository, connectionService *connectionsusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if comboRepo == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "model combo repository is not configured")
			return
		}

		switch r.Method {
		case http.MethodGet:
			combos, err := comboRepo.ListModelCombos()
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, modelCombosListResponse{Object: "list", Data: combos})
		case http.MethodPost:
			resolvedCatalog, err := catalogWithCustomModels(catalog, modelRepo)
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			var payload modelComboPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}
			combo, err := buildModelCombo(resolvedCatalog, connectionService.List(), payload)
			if err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			if _, exists, err := comboRepo.GetModelCombo(combo.Alias); err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			} else if exists {
				writeError(r, w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("model combo alias %q already exists", combo.Alias))
				return
			}
			if err := comboRepo.CreateModelCombo(combo); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, combo)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func modelComboByAliasHandler(catalog provider.Catalog, modelRepo providerModelRepository, comboRepo modelComboRepository, connectionService *connectionsusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if comboRepo == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "model combo repository is not configured")
			return
		}

		alias := strings.TrimSpace(chi.URLParam(r, "*"))
		if alias == "" {
			alias = strings.TrimSpace(chi.URLParam(r, "alias"))
		}
		if alias == "" {
			writeError(r, w, http.StatusNotFound, "not_found", "model combo not found")
			return
		}

		switch r.Method {
		case http.MethodPut:
			resolvedCatalog, err := catalogWithCustomModels(catalog, modelRepo)
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			var payload modelComboPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}
			combo, err := buildModelCombo(resolvedCatalog, connectionService.List(), payload)
			if err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			if combo.Alias != alias {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "model combo alias cannot be changed")
				return
			}
			if err := comboRepo.UpdateModelCombo(alias, combo); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					writeError(r, w, http.StatusNotFound, "not_found", "model combo not found")
					return
				}
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, combo)
		case http.MethodDelete:
			if err := comboRepo.DeleteModelCombo(alias); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					writeError(r, w, http.StatusNotFound, "not_found", "model combo not found")
					return
				}
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func routingInputs(catalog provider.Catalog, modelRepo providerModelRepository, comboRepo modelComboRepository) (provider.Catalog, []modelcombo.Combo, error) {
	resolvedCatalog, err := catalogWithCustomModels(catalog, modelRepo)
	if err != nil {
		return provider.Catalog{}, nil, err
	}
	combos, err := listModelCombos(comboRepo)
	if err != nil {
		return provider.Catalog{}, nil, err
	}

	return resolvedCatalog, combos, nil
}

func listModelCombos(comboRepo modelComboRepository) ([]modelcombo.Combo, error) {
	if comboRepo == nil {
		return nil, nil
	}

	return comboRepo.ListModelCombos()
}

func buildModelCombo(catalog provider.Catalog, connections []connectionsusecase.Item, payload modelComboPayload) (modelcombo.Combo, error) {
	alias := strings.TrimSpace(payload.Alias)
	if alias == "" {
		return modelcombo.Combo{}, fmt.Errorf("alias is required")
	}
	if len(payload.Targets) == 0 {
		return modelcombo.Combo{}, fmt.Errorf("model combo must include at least one target")
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = alias
	}
	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}

	targets := make([]modelcombo.Target, 0, len(payload.Targets))
	for index, inputTarget := range payload.Targets {
		providerID := strings.TrimSpace(inputTarget.ProviderID)
		connectionID := strings.TrimSpace(inputTarget.ConnectionID)
		modelID := strings.TrimSpace(inputTarget.ModelID)
		if providerID == "" {
			return modelcombo.Combo{}, fmt.Errorf("target %d provider_id is required", index+1)
		}
		if modelID == "" {
			return modelcombo.Combo{}, fmt.Errorf("target %d model_id is required", index+1)
		}
		resolvedProvider, ok := catalog.FindByID(providerID)
		if !ok {
			return modelcombo.Combo{}, fmt.Errorf("target %d provider %q not found", index+1, providerID)
		}
		if !providerHasModel(resolvedProvider, modelID) {
			return modelcombo.Combo{}, fmt.Errorf("target %d model %q does not belong to provider %q", index+1, modelID, providerID)
		}
		if connectionID != "" && !connectionBelongsToProvider(connections, connectionID, providerID) {
			return modelcombo.Combo{}, fmt.Errorf("target %d connection %q does not belong to provider %q", index+1, connectionID, providerID)
		}
		enabled := true
		if inputTarget.Enabled != nil {
			enabled = *inputTarget.Enabled
		}
		targets = append(targets, modelcombo.Target{
			ComboAlias:   alias,
			ProviderID:   providerID,
			ConnectionID: connectionID,
			ModelID:      modelID,
			Enabled:      enabled,
			Priority:     index,
		})
	}
	sort.Sort(modelcombo.Targets(targets))

	return modelcombo.Combo{
		Alias:       alias,
		Name:        name,
		Description: strings.TrimSpace(payload.Description),
		Enabled:     enabled,
		Targets:     targets,
	}, nil
}

func connectionBelongsToProvider(connections []connectionsusecase.Item, connectionID string, providerID string) bool {
	for _, connection := range connections {
		if connection.ID == connectionID && connection.ProviderID == providerID {
			return true
		}
	}

	return false
}

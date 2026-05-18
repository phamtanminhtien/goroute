package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phamtanminhtien/goroute/internal/domain/systemapikey"
	"gorm.io/gorm"
)

type systemAPIKeyRepository interface {
	ListSystemAPIKeys() ([]systemapikey.Record, error)
	CreateSystemAPIKey(systemapikey.Record) error
	GetSystemAPIKey(string) (systemapikey.Record, bool, error)
	UpdateSystemAPIKey(systemapikey.Record) error
	DeleteSystemAPIKey(string) error
}

type systemAPIKeyAuthRepository interface {
	HasSystemAPIKeys() (bool, error)
	AuthenticateSystemAPIKey(string) (systemapikey.Record, bool, error)
}

type systemAPIKeyRequest struct {
	Name    *string `json:"name"`
	Enabled *bool   `json:"enabled"`
}

type systemAPIKeyListResponse struct {
	Object string                `json:"object"`
	Data   []systemapikey.Record `json:"data"`
}

func systemAPIKeysHandler(repo systemAPIKeyRepository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if repo == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "system api key repository is not configured")
			return
		}

		switch r.Method {
		case http.MethodGet:
			records, err := repo.ListSystemAPIKeys()
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, systemAPIKeyListResponse{
				Object: "list",
				Data:   records,
			})
		case http.MethodPost:
			var input systemAPIKeyRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			name := ""
			if input.Name != nil {
				name = strings.TrimSpace(*input.Name)
			}
			if name == "" {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "name is required")
				return
			}

			record, err := newSystemAPIKeyRecord(name)
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			if err := repo.CreateSystemAPIKey(record); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}

			writeJSON(w, http.StatusCreated, record)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func systemAPIKeyByIDHandler(repo systemAPIKeyRepository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if repo == nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "system api key repository is not configured")
			return
		}

		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "id is required")
			return
		}

		switch r.Method {
		case http.MethodPut:
			existing, ok, err := repo.GetSystemAPIKey(id)
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			if !ok {
				writeError(r, w, http.StatusNotFound, "not_found", "system api key not found")
				return
			}

			var input systemAPIKeyRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			if input.Name != nil {
				existing.Name = strings.TrimSpace(*input.Name)
			}
			if existing.Name == "" {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "name is required")
				return
			}
			if input.Enabled != nil {
				existing.Enabled = *input.Enabled
			}

			if err := repo.UpdateSystemAPIKey(existing); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					writeError(r, w, http.StatusNotFound, "not_found", "system api key not found")
					return
				}
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}

			updated, ok, err := repo.GetSystemAPIKey(id)
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			if !ok {
				writeError(r, w, http.StatusNotFound, "not_found", "system api key not found")
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			if err := repo.DeleteSystemAPIKey(id); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					writeError(r, w, http.StatusNotFound, "not_found", "system api key not found")
					return
				}
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func newSystemAPIKeyRecord(name string) (systemapikey.Record, error) {
	token, err := randomBase64URLString(24)
	if err != nil {
		return systemapikey.Record{}, fmt.Errorf("generate system api key: %w", err)
	}

	return systemapikey.Record{
		ID:      "sak_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		Name:    name,
		Key:     "sk-goroute-" + token,
		Enabled: true,
	}, nil
}

func randomBase64URLString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

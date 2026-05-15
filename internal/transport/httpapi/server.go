package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/health"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
	"github.com/rs/zerolog"
)

func NewServer(catalog provider.Catalog, connectionRegistry *chatcompletion.ConnectionRegistry, connectionService *connectionsusecase.Service, adminAuthToken string, logger *zerolog.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(requestIDMiddleware, loggingMiddleware(logger))

	router.Handle("/healthz", health.Handler())
	router.Handle("/v1/models", modelsHandler(catalog))
	router.Handle("/v1/chat/completions", chatCompletionsHandler(catalog, connectionRegistry))

	router.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return authMiddleware(adminAuthToken, next)
		})
		r.Handle("/admin/api/providers", providersHandler(catalog, connectionService))
		r.Handle("/admin/api/connections", connectionsHandler(connectionService))
		r.Handle("/admin/api/connections/{id}", connectionByIDHandler(connectionService))
		r.Handle("/debug/requests", requestHistoryHandler(connectionRegistry))
	})

	return router
}

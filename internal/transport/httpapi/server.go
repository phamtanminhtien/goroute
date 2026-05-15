package httpapi

import (
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/health"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

func NewServer(catalog provider.Catalog, connectionRegistry *chatcompletion.ConnectionRegistry, connectionService *connectionsusecase.Service, adminAuthToken string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", health.Handler())
	mux.Handle("/v1/models", modelsHandler(catalog))
	mux.Handle("/v1/chat/completions", chatCompletionsHandler(catalog, connectionRegistry))
	mux.Handle("/admin/api/providers", authMiddleware(adminAuthToken, providersHandler(catalog, connectionService)))
	mux.Handle("/admin/api/connections", authMiddleware(adminAuthToken, connectionsHandler(connectionService)))
	mux.Handle("/admin/api/connections/", authMiddleware(adminAuthToken, connectionByIDHandler(connectionService)))
	mux.Handle("/debug/requests", authMiddleware(adminAuthToken, requestHistoryHandler(connectionRegistry)))

	return requestIDMiddleware(loggingMiddleware(mux))
}

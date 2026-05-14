package httpapi

import (
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/health"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	providersusecase "github.com/phamtanminhtien/goroute/internal/usecase/providers"
)

func NewServer(catalog driver.Catalog, providerRegistry *chatcompletion.ProviderRegistry, providerService *providersusecase.Service, adminAuthToken string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", health.Handler())
	mux.Handle("/v1/models", modelsHandler(catalog))
	mux.Handle("/v1/chat/completions", chatCompletionsHandler(catalog, providerRegistry))
	mux.Handle("/admin/api/providers", authMiddleware(adminAuthToken, providersHandler(providerService)))
	mux.Handle("/admin/api/providers/", authMiddleware(adminAuthToken, providerByIDHandler(providerService)))
	mux.Handle("/debug/requests", authMiddleware(adminAuthToken, requestHistoryHandler(providerRegistry)))

	return requestIDMiddleware(loggingMiddleware(mux))
}

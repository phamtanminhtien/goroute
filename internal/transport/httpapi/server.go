package httpapi

import (
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/health"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func NewServer(authToken string, catalog driver.Catalog, providerRegistry chatcompletion.ProviderRegistry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", health.Handler())
	mux.Handle("/v1/models", authMiddleware(authToken, modelsHandler(catalog)))
	mux.Handle("/v1/chat/completions", authMiddleware(authToken, chatCompletionsHandler(catalog, providerRegistry)))

	return requestIDMiddleware(loggingMiddleware(mux))
}

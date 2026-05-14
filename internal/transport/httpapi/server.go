package httpapi

import (
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/health"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func NewServer(catalog driver.Catalog, providerRegistry chatcompletion.ProviderRegistry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", health.Handler())
	mux.Handle("/v1/models", modelsHandler(catalog))
	mux.Handle("/v1/chat/completions", chatCompletionsHandler(catalog, providerRegistry))

	return requestIDMiddleware(loggingMiddleware(mux))
}

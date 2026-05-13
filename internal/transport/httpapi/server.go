package httpapi

import (
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/health"
)

func NewServer(authToken string, catalog driver.Catalog) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", health.Handler())
	mux.Handle("/v1/models", authMiddleware(authToken, modelsHandler(catalog)))
	mux.Handle("/v1/chat/completions", authMiddleware(authToken, chatCompletionsHandler(catalog)))

	return requestIDMiddleware(loggingMiddleware(mux))
}

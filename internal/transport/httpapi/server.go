package httpapi

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/health"
	"github.com/phamtanminhtien/goroute/internal/usecase/analytics"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
	"github.com/rs/zerolog"
)

type catalogSource interface {
	Catalog() provider.Catalog
}

type aiRequestLogRepository interface {
	CreateAIRequestRun(record *airequestlog.RunRecord) error
	CreateAIRequestFlow(record airequestlog.FlowRecord) error
	CreateThirdPartyRequestLog(record airequestlog.ThirdPartyRequestLogRecord) error
	CreateRTKRecord(record *airequestlog.RTKRecord) error
	analytics.Repository
}

func NewServer(catalog catalogSource, connectionRegistry *chatcompletion.ConnectionRegistry, connectionService *connectionsusecase.Service, providerService providerMutationService, requestLogRepo aiRequestLogRepository, modelRepo providerModelRepository, modelComboRepo modelComboRepository, systemKeyRepo interface {
	systemAPIKeyRepository
	systemAPIKeyAuthRepository
}, settingsManager *config.SettingsManager, adminAuthToken string, webUIRoot fs.FS, logger *zerolog.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(requestIDMiddleware, loggingMiddleware(logger))
	analyticsService := analytics.NewServiceWithPricingSource(requestLogRepo, catalog, modelRepo)

	router.Handle("/healthz", health.Handler())
	router.Handle("/v1/models", openAICompatibleAuthMiddleware(settingsManager, systemKeyRepo, false, modelsHandler(catalog, modelRepo, modelComboRepo)))
	router.Handle("/v1/chat/completions", openAICompatibleAuthMiddleware(settingsManager, systemKeyRepo, true, chatCompletionsHandler(catalog, connectionRegistry, requestLogRepo, modelRepo, modelComboRepo, settingsManager, logger)))
	router.Handle("/v1/responses", openAICompatibleAuthMiddleware(settingsManager, systemKeyRepo, true, responsesHandler(catalog, connectionRegistry, requestLogRepo, modelRepo, modelComboRepo, settingsManager, logger)))
	router.Handle("/v1/messages", anthropicCompatibleAuthMiddleware(settingsManager, systemKeyRepo, true, anthropicMessagesHandler(catalog, connectionRegistry, requestLogRepo, modelRepo, modelComboRepo, settingsManager, logger)))

	router.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return authMiddleware(adminAuthToken, next)
		})
		r.Handle("/admin/api/providers", providersHandler(catalog, connectionService, providerService, modelRepo))
		r.Handle("/admin/api/providers/{id}", providerByIDHandler(providerService))
		r.Handle("/admin/api/providers/{id}/connections/enabled", providerConnectionsEnabledHandler(catalog, connectionService))
		r.Handle("/admin/api/providers/{id}/models", providerModelsHandler(catalog, modelRepo))
		r.Handle("/admin/api/providers/{id}/models/*", providerModelByIDHandler(catalog, modelRepo))
		r.Handle("/admin/api/providers/{id}/oauth-url", providerOAuthURLHandler(connectionService))
		r.Handle("/admin/api/providers/{id}/test", providerModelTestHandler(catalog, connectionRegistry, modelRepo))
		r.Handle("/admin/api/model-combos", modelCombosHandler(catalog, modelRepo, modelComboRepo, connectionService))
		r.Handle("/admin/api/model-combos/*", modelComboByAliasHandler(catalog, modelRepo, modelComboRepo, connectionService))
		r.Handle("/admin/api/connections", connectionsHandler(connectionService))
		r.Handle("/admin/api/connections/{id}", connectionByIDHandler(connectionService))
		r.Handle("/admin/api/connections/{id}/usage", connectionUsageHandler(connectionService))
		r.Handle("/admin/api/connections/oauth", connectionOAuthHandler(connectionService))
		r.Handle("/admin/api/system-api-keys", systemAPIKeysHandler(systemKeyRepo))
		r.Handle("/admin/api/system-api-keys/{id}", systemAPIKeyByIDHandler(systemKeyRepo))
		r.Handle("/admin/api/settings", settingsHandler(settingsManager))
		r.Handle("/admin/api/analytics/usage/summary", analyticsUsageSummaryHandler(analyticsService))
		r.Handle("/admin/api/analytics/usage/timeseries", analyticsUsageTimeseriesHandler(analyticsService))
		r.Handle("/admin/api/analytics/usage/provider-breakdown", analyticsUsageProviderBreakdownHandler(analyticsService))
		r.Handle("/admin/api/analytics/usage/recent-requests", analyticsUsageRecentRequestsHandler(analyticsService))
		r.Handle("/admin/api/analytics/usage/requests", analyticsUsageRequestsHandler(analyticsService))
		r.Handle("/admin/api/analytics/usage/requests/{request_id}", analyticsUsageRequestDetailHandler(analyticsService))
		r.Handle("/admin/api/logs/stream", logsStreamHandler())
	})

	if webUIRoot != nil {
		router.NotFound(webUIHandler(webUIRoot).ServeHTTP)
	}

	return router
}

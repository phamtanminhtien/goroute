package app

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	adapteropenaicompatible "github.com/phamtanminhtien/goroute/internal/adapter/openaicompatible"
	providercodex "github.com/phamtanminhtien/goroute/internal/adapter/provider/codex"
	provideropenai "github.com/phamtanminhtien/goroute/internal/adapter/provider/openai"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/storage/gormsqlite"
	"github.com/phamtanminhtien/goroute/internal/transport/httpapi"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	"github.com/phamtanminhtien/goroute/internal/usecase/connections"
	providersusecase "github.com/phamtanminhtien/goroute/internal/usecase/providers"
	"github.com/rs/zerolog"
)

type App struct {
	server *http.Server
	logger zerolog.Logger
	repo   *gormsqlite.Repository
}

func New(logger zerolog.Logger) (*App, error) {
	configPath, err := config.ResolvePath()
	if err != nil {
		return nil, fmt.Errorf("resolve user config path: %w", err)
	}

	cfg, createdConfig, err := config.LoadOrCreatePath(configPath)
	if err != nil {
		return nil, fmt.Errorf("load user config: %w", err)
	}
	if createdConfig {
		logger.Info().Str("config_path", configPath).Msg("user_config_created")
	}
	settingsManager := config.NewSettingsManager(configPath, cfg)
	databasePath, err := config.ResolveDatabasePath()
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite repository: %w", err)
	}

	providerRuntime := &providerRuntime{repo: repo}
	if err := providerRuntime.ReloadProviders(); err != nil {
		repo.Close()
		return nil, fmt.Errorf("build provider registry: %w", err)
	}

	appLogger := logger.With().Str("component", "app").Logger()
	connectionRegistryLogger := logger.With().Str("component", "connection_registry").Logger()
	connectionServiceLogger := logger.With().Str("component", "connections_service").Logger()
	httpLogger := logger.With().Str("component", "http").Logger()

	connectionRuntime := &connectionRuntime{
		repo:            repo,
		providerRuntime: providerRuntime,
		logger:          &connectionRegistryLogger,
	}
	connectionRegistry, err := connectionRuntime.BuildRegistry()
	if err != nil {
		repo.Close()
		return nil, err
	}

	connectionRuntime.registry = connectionRegistry

	connectionService := connections.NewService(repo, connectionRuntime, providerRuntime, &connectionServiceLogger)
	providerService := providersusecase.NewService(repo, &combinedRuntime{
		providerRuntime:   providerRuntime,
		connectionRuntime: connectionRuntime,
	})

	webUIRoot, webUIDir := resolveWebUIRoot(cfg.Server.WebUIDir)
	if webUIRoot == nil {
		appLogger.Warn().Str("web_ui_dir", webUIDir).Msg("web_ui_disabled")
	} else {
		appLogger.Info().Str("web_ui_dir", webUIDir).Msg("web_ui_enabled")
	}

	handler := httpapi.NewServer(providerRuntime, connectionRegistry, connectionService, providerService, repo, repo, repo, repo, settingsManager, cfg.Server.AuthToken, webUIRoot, &httpLogger)
	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{server: server, logger: appLogger, repo: repo}, nil
}

func buildProviderRegistry() (providerregistry.Registry, error) {
	return providerregistry.New(
		providercodex.Registration(),
		provideropenai.Registration(),
	)
}

func buildProviderRegistryWithCustom(customProviders []provider.Record) (providerregistry.Registry, error) {
	registrations := []providerregistry.Registration{
		providercodex.Registration(),
		provideropenai.Registration(),
	}
	for _, customProvider := range customProviders {
		registrations = append(registrations, adapteropenaicompatible.Registration(customProvider.Provider()))
	}

	return providerregistry.New(registrations...)
}

func buildConnectionRegistryWithLogger(connectionConfigs []connection.Record, providers providerregistry.Registry, logger *zerolog.Logger) (*chatcompletion.ConnectionRegistry, error) {
	entries, err := buildConnectionEntries(connectionConfigs, providers, logger)
	if err != nil {
		return nil, err
	}

	registry := chatcompletion.NewConnectionRegistryWithEntries(entries, logger)
	return &registry, nil
}

func buildConnectionEntries(connectionConfigs []connection.Record, providers providerregistry.Registry, logger *zerolog.Logger) (map[string][]chatcompletion.ConnectionEntry, error) {
	connectionsByProvider := make(map[string][]chatcompletion.ConnectionEntry, len(connectionConfigs))
	for _, connectionConfig := range connectionConfigs {
		logConnectionDiagnostic(logger, providers, connectionConfig)
		if !connectionConfig.Enabled {
			continue
		}

		connections, err := providers.BuildConnection(connectionConfig)
		if err != nil {
			return nil, err
		}
		connectionsByProvider[connectionConfig.ProviderID] = append(connectionsByProvider[connectionConfig.ProviderID], chatcompletion.ConnectionEntry{
			ID:                  connectionConfig.ID,
			Name:                connectionConfig.Name,
			ProviderID:          connectionConfig.ProviderID,
			LastErrorMessage:    connectionConfig.LastErrorMessage,
			LastErrorCategory:   connectionConfig.LastErrorCategory,
			LastErrorAt:         connectionConfig.LastErrorAt,
			RetryAfter:          connectionConfig.RetryAfter,
			ProtocolConnections: connections,
		})
	}

	return connectionsByProvider, nil
}

func logConnectionDiagnostic(logger *zerolog.Logger, providers providerregistry.Registry, connection connection.Record) {
	if logger == nil {
		return
	}

	problems := providers.ValidateConnection(connection)
	status := "ready"
	if len(problems) > 0 {
		status = "misconfigured"
	}

	logger.Info().
		Str("connection_id", connection.ID).
		Str("provider_id", connection.ProviderID).
		Str("connection_name", connection.Name).
		Str("status", status).
		Strs("problems", problems).
		Msg("connection_diagnostic")
}

type connectionRuntime struct {
	repo interface {
		ListConnections() ([]connection.Record, error)
		RecordConnectionRuntimeError(id string, message string, category string, lastErrorAt int64, retryAfter int64) error
		ClearConnectionRuntimeError(id string) error
	}
	providerRuntime *providerRuntime
	registry        *chatcompletion.ConnectionRegistry
	logger          *zerolog.Logger
}

func (r *connectionRuntime) BuildRegistry() (*chatcompletion.ConnectionRegistry, error) {
	connectionConfigs, err := r.repo.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("load runtime connections: %w", err)
	}

	entries, err := buildConnectionEntries(connectionConfigs, r.providerRuntime.Registry(), r.logger)
	if err != nil {
		return nil, err
	}

	registry := chatcompletion.NewConnectionRegistryWithStateStore(entries, r.logger, r.repo)
	return &registry, nil
}

func (r *connectionRuntime) ReloadConnections() error {
	connectionConfigs, err := r.repo.ListConnections()
	if err != nil {
		return fmt.Errorf("load runtime connections: %w", err)
	}

	entries, err := buildConnectionEntries(connectionConfigs, r.providerRuntime.Registry(), r.logger)
	if err != nil {
		return err
	}

	r.registry.ReplaceConnections(entries)
	return nil
}

type providerRepository interface {
	ListProviders() ([]provider.Record, error)
}

type providerRuntime struct {
	mu       sync.RWMutex
	repo     providerRepository
	registry providerregistry.Registry
	catalog  provider.Catalog
}

func (r *providerRuntime) ReloadProviders() error {
	customProviders, err := r.repo.ListProviders()
	if err != nil {
		return fmt.Errorf("load custom providers: %w", err)
	}
	registry, err := buildProviderRegistryWithCustom(customProviders)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.registry = registry
	r.catalog = registry.Catalog()
	return nil
}

func (r *providerRuntime) Catalog() provider.Catalog {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.catalog
}

func (r *providerRuntime) Registry() providerregistry.Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.registry
}

func (r *providerRuntime) IsSystemProvider(providerID string) bool {
	return r.Registry().IsSystemProvider(providerID)
}

func (r *providerRuntime) IsCustomProvider(providerID string) bool {
	return r.Registry().IsCustomProvider(providerID)
}

func (r *providerRuntime) ValidateConnection(connection connection.Record) []string {
	return r.Registry().ValidateConnection(connection)
}

func (r *providerRuntime) GetUsage(ctx context.Context, connection connection.Record) (providerregistry.UsageInfo, error) {
	return r.Registry().GetUsage(ctx, connection)
}

func (r *providerRuntime) GenerateOAuthURL(connection connection.Record) (string, error) {
	return r.Registry().GenerateOAuthURL(connection)
}

func (r *providerRuntime) StartOAuth(connection connection.Record) (providerregistry.OAuthSession, error) {
	return r.Registry().StartOAuth(connection)
}

func (r *providerRuntime) CompleteOAuth(connection connection.Record, pending map[string]string, callbackURL string) (providerregistry.OAuthResult, error) {
	return r.Registry().CompleteOAuth(connection, pending, callbackURL)
}

type combinedRuntime struct {
	providerRuntime   *providerRuntime
	connectionRuntime *connectionRuntime
}

func (r *combinedRuntime) ReloadProviders() error {
	return r.providerRuntime.ReloadProviders()
}

func (r *combinedRuntime) ReloadConnections() error {
	return r.connectionRuntime.ReloadConnections()
}

func (r *combinedRuntime) IsSystemProvider(providerID string) bool {
	return r.providerRuntime.IsSystemProvider(providerID)
}

func (r *combinedRuntime) IsCustomProvider(providerID string) bool {
	return r.providerRuntime.IsCustomProvider(providerID)
}

func resolveWebUIRoot(webUIDir string) (fs.FS, string) {
	resolvedPath, err := filepath.Abs(webUIDir)
	if err != nil {
		return nil, webUIDir
	}

	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return nil, resolvedPath
	}

	return os.DirFS(resolvedPath), resolvedPath
}

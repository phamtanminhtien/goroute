package app

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	providercodex "github.com/phamtanminhtien/goroute/internal/adapter/provider/codex"
	provideropenai "github.com/phamtanminhtien/goroute/internal/adapter/provider/openai"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/storage/gormsqlite"
	"github.com/phamtanminhtien/goroute/internal/transport/httpapi"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	"github.com/phamtanminhtien/goroute/internal/usecase/connections"
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

	cfg, err := config.LoadPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("load user config: %w", err)
	}
	databasePath, err := config.ResolveDatabasePath()
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite repository: %w", err)
	}

	providers, err := buildProviderRegistry()
	if err != nil {
		repo.Close()
		return nil, fmt.Errorf("build provider registry: %w", err)
	}
	catalog := providers.Catalog()

	appLogger := logger.With().Str("component", "app").Logger()
	connectionRegistryLogger := logger.With().Str("component", "connection_registry").Logger()
	connectionServiceLogger := logger.With().Str("component", "connections_service").Logger()
	httpLogger := logger.With().Str("component", "http").Logger()

	connectionRuntime := &connectionRuntime{
		repo:      repo,
		providers: providers,
		logger:    &connectionRegistryLogger,
	}
	connectionRegistry, err := connectionRuntime.BuildRegistry()
	if err != nil {
		repo.Close()
		return nil, err
	}

	connectionRuntime.registry = connectionRegistry

	connectionService := connections.NewService(repo, connectionRuntime, providers, &connectionServiceLogger)

	webUIRoot, webUIDir := resolveWebUIRoot(cfg.Server.WebUIDir)
	if webUIRoot == nil {
		appLogger.Warn().Str("web_ui_dir", webUIDir).Msg("web_ui_disabled")
	} else {
		appLogger.Info().Str("web_ui_dir", webUIDir).Msg("web_ui_enabled")
	}

	handler := httpapi.NewServer(catalog, connectionRegistry, connectionService, cfg.Server.AuthToken, webUIRoot, &httpLogger)
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

		connection, err := providers.BuildConnection(connectionConfig)
		if err != nil {
			return nil, err
		}
		connectionsByProvider[connectionConfig.ProviderID] = append(connectionsByProvider[connectionConfig.ProviderID], chatcompletion.ConnectionEntry{
			ID:         connectionConfig.ID,
			Name:       connectionConfig.Name,
			ProviderID: connectionConfig.ProviderID,
			Connection: connection,
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
	}
	providers providerregistry.Registry
	registry  *chatcompletion.ConnectionRegistry
	logger    *zerolog.Logger
}

func (r *connectionRuntime) BuildRegistry() (*chatcompletion.ConnectionRegistry, error) {
	connectionConfigs, err := r.repo.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("load runtime connections: %w", err)
	}

	entries, err := buildConnectionEntries(connectionConfigs, r.providers, r.logger)
	if err != nil {
		return nil, err
	}

	registry := chatcompletion.NewConnectionRegistryWithEntries(entries, r.logger)
	return &registry, nil
}

func (r *connectionRuntime) ReloadConnections() error {
	connectionConfigs, err := r.repo.ListConnections()
	if err != nil {
		return fmt.Errorf("load runtime connections: %w", err)
	}

	entries, err := buildConnectionEntries(connectionConfigs, r.providers, r.logger)
	if err != nil {
		return err
	}

	r.registry.ReplaceConnections(entries)
	return nil
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

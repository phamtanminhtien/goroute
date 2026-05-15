package app

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	providercodex "github.com/phamtanminhtien/goroute/internal/adapter/provider/codex"
	provideropenai "github.com/phamtanminhtien/goroute/internal/adapter/provider/openai"
	"github.com/phamtanminhtien/goroute/internal/adapter/systemdata"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/transport/httpapi"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	"github.com/phamtanminhtien/goroute/internal/usecase/connections"
	"github.com/rs/zerolog"
)

type App struct {
	server *http.Server
	logger zerolog.Logger
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

	catalog, err := systemdata.LoadFile(filepath.Join("data", "system-providers.json"))
	if err != nil {
		return nil, fmt.Errorf("load system provider data: %w", err)
	}

	appLogger := logger.With().Str("component", "app").Logger()
	connectionRegistryLogger := logger.With().Str("component", "connection_registry").Logger()
	connectionServiceLogger := logger.With().Str("component", "connections_service").Logger()
	httpLogger := logger.With().Str("component", "http").Logger()

	connectionRegistry, err := buildConnectionRegistryWithLogger(cfg.Connections, &connectionRegistryLogger)
	if err != nil {
		return nil, err
	}

	connectionService := connections.NewService(configPath, cfg, &connectionRuntime{
		registry: connectionRegistry,
		logger:   &connectionRegistryLogger,
	}, &connectionServiceLogger)

	handler := httpapi.NewServer(catalog, connectionRegistry, connectionService, cfg.Server.AuthToken, &httpLogger)
	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{server: server, logger: appLogger}, nil
}

func buildConnectionRegistryWithLogger(connectionConfigs []config.ConnectionConfig, logger *zerolog.Logger) (*chatcompletion.ConnectionRegistry, error) {
	entries, err := buildConnectionEntries(connectionConfigs, logger)
	if err != nil {
		return nil, err
	}

	registry := chatcompletion.NewConnectionRegistryWithEntries(entries, logger)
	return &registry, nil
}

func buildConnectionEntries(connectionConfigs []config.ConnectionConfig, logger *zerolog.Logger) (map[string][]chatcompletion.ConnectionEntry, error) {
	connectionsByProvider := make(map[string][]chatcompletion.ConnectionEntry, len(connectionConfigs))
	for _, connectionConfig := range connectionConfigs {
		logConnectionDiagnostic(logger, connectionConfig)

		var connection chatcompletion.Connection
		switch connectionConfig.ProviderID {
		case "cx":
			connection = providercodex.NewClient(connectionConfig)
		case "openai":
			connection = provideropenai.NewClient(nil, connectionConfig)
		default:
			return nil, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
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

func logConnectionDiagnostic(logger *zerolog.Logger, connection config.ConnectionConfig) {
	if logger == nil {
		return
	}

	problems := make([]string, 0, 2)
	switch connection.ProviderID {
	case "cx":
		if strings.TrimSpace(connection.AccessToken) == "" && strings.TrimSpace(connection.APIKey) == "" {
			problems = append(problems, "missing access_token or api_key")
		}
	case "openai":
		if strings.TrimSpace(connection.APIKey) == "" && strings.TrimSpace(connection.AccessToken) == "" {
			problems = append(problems, "missing api_key or access_token")
		}
	default:
		problems = append(problems, "unsupported provider")
	}

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
	registry *chatcompletion.ConnectionRegistry
	logger   *zerolog.Logger
}

func (r *connectionRuntime) ReplaceConnections(connectionConfigs []config.ConnectionConfig) error {
	entries, err := buildConnectionEntries(connectionConfigs, r.logger)
	if err != nil {
		return err
	}

	r.registry.ReplaceConnections(entries)
	return nil
}

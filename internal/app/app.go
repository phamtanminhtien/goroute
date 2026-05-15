package app

import (
	"fmt"
	"log"
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
)

type App struct {
	server *http.Server
}

func New() (*App, error) {
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

	connectionRegistry, err := buildConnectionRegistry(cfg.Connections)
	if err != nil {
		return nil, err
	}

	connectionService := connections.NewService(configPath, cfg, &connectionRuntime{
		registry: connectionRegistry,
		logger:   log.Default(),
	}, log.Default())

	handler := httpapi.NewServer(catalog, connectionRegistry, connectionService, cfg.Server.AuthToken)
	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{server: server}, nil
}

func buildConnectionRegistry(connectionConfigs []config.ConnectionConfig) (*chatcompletion.ConnectionRegistry, error) {
	return buildConnectionRegistryWithLogger(connectionConfigs, log.Default())
}

func buildConnectionRegistryWithLogger(connectionConfigs []config.ConnectionConfig, logger *log.Logger) (*chatcompletion.ConnectionRegistry, error) {
	entries, err := buildConnectionEntries(connectionConfigs, logger)
	if err != nil {
		return nil, err
	}

	registry := chatcompletion.NewConnectionRegistryWithEntries(entries, logger)
	return &registry, nil
}

func buildConnectionEntries(connectionConfigs []config.ConnectionConfig, logger *log.Logger) (map[string][]chatcompletion.ConnectionEntry, error) {
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

func logConnectionDiagnostic(logger *log.Logger, connection config.ConnectionConfig) {
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

	logger.Printf(
		"connection_diagnostic connection_id=%q provider_id=%q connection_name=%q status=%q problems=%q",
		connection.ID,
		connection.ProviderID,
		connection.Name,
		status,
		strings.Join(problems, ", "),
	)
}

type connectionRuntime struct {
	registry *chatcompletion.ConnectionRegistry
	logger   *log.Logger
}

func (r *connectionRuntime) ReplaceConnections(connectionConfigs []config.ConnectionConfig) error {
	entries, err := buildConnectionEntries(connectionConfigs, r.logger)
	if err != nil {
		return err
	}

	r.registry.ReplaceConnections(entries)
	return nil
}

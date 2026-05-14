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
)

type App struct {
	server *http.Server
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load user config: %w", err)
	}

	catalog, err := systemdata.LoadFile(filepath.Join("data", "system-drivers.json"))
	if err != nil {
		return nil, fmt.Errorf("load system driver data: %w", err)
	}

	providerRegistry, err := buildProviderRegistry(cfg.Providers)
	if err != nil {
		return nil, err
	}

	handler := httpapi.NewServer(catalog, providerRegistry, cfg.Server.AuthToken)
	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{server: server}, nil
}

func buildProviderRegistry(providerConfigs []config.ProviderConfig) (chatcompletion.ProviderRegistry, error) {
	return buildProviderRegistryWithLogger(providerConfigs, log.Default())
}

func buildProviderRegistryWithLogger(providerConfigs []config.ProviderConfig, logger *log.Logger) (chatcompletion.ProviderRegistry, error) {
	providers := make(map[string][]chatcompletion.ProviderEntry, len(providerConfigs))
	for _, providerConfig := range providerConfigs {
		logProviderDiagnostic(logger, providerConfig)

		var provider chatcompletion.Provider
		switch providerConfig.Type {
		case "codex":
			provider = providercodex.NewClient(providerConfig)
		case "openai":
			provider = provideropenai.NewClient(nil, providerConfig)
		default:
			return chatcompletion.ProviderRegistry{}, fmt.Errorf("unsupported provider type %q", providerConfig.Type)
		}
		providers[providerConfig.Type] = append(providers[providerConfig.Type], chatcompletion.ProviderEntry{
			ID:       providerConfig.ID,
			Name:     providerConfig.Name,
			Provider: provider,
		})
	}

	return chatcompletion.NewProviderRegistryWithEntries(providers, logger), nil
}

func logProviderDiagnostic(logger *log.Logger, provider config.ProviderConfig) {
	if logger == nil {
		return
	}

	problems := make([]string, 0, 2)
	switch provider.Type {
	case "codex":
		if strings.TrimSpace(provider.AccessToken) == "" && strings.TrimSpace(provider.APIKey) == "" {
			problems = append(problems, "missing access_token or api_key")
		}
	case "openai":
		if strings.TrimSpace(provider.APIKey) == "" && strings.TrimSpace(provider.AccessToken) == "" {
			problems = append(problems, "missing api_key or access_token")
		}
	default:
		problems = append(problems, "unsupported provider type")
	}

	status := "ready"
	if len(problems) > 0 {
		status = "misconfigured"
	}

	logger.Printf(
		"provider_diagnostic provider_id=%q provider_name=%q provider_type=%q status=%q problems=%q",
		provider.ID,
		provider.Name,
		provider.Type,
		status,
		strings.Join(problems, ", "),
	)
}

package app

import (
	"fmt"
	"net/http"
	"path/filepath"
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

	handler := httpapi.NewServer(catalog, providerRegistry)
	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{server: server}, nil
}

func buildProviderRegistry(providerConfigs []config.ProviderConfig) (chatcompletion.ProviderRegistry, error) {
	providers := make(map[string][]chatcompletion.Provider, len(providerConfigs))
	for _, providerConfig := range providerConfigs {
		var provider chatcompletion.Provider
		switch providerConfig.Type {
		case "codex":
			provider = providercodex.NewClient(providerConfig)
		case "openai":
			provider = provideropenai.NewClient(nil, providerConfig)
		default:
			return chatcompletion.ProviderRegistry{}, fmt.Errorf("unsupported provider type %q", providerConfig.Type)
		}
		providers[providerConfig.Type] = append(providers[providerConfig.Type], provider)
	}

	return chatcompletion.NewProviderRegistry(providers), nil
}

package app

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/phamtanminhtien/goroute/internal/adapter/systemdata"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/transport/httpapi"
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

	handler := httpapi.NewServer(cfg.Server.AuthToken, catalog)
	server := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{server: server}, nil
}

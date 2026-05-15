package main

import (
	"os"

	"github.com/phamtanminhtien/goroute/internal/app"
	"github.com/phamtanminhtien/goroute/internal/logging"
)

func main() {
	logger := logging.New(os.Getenv("GOROUTE_ENV"))

	application, err := app.New(logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("build_app_failed")
	}

	if err := application.Run(); err != nil {
		logger.Fatal().Err(err).Msg("run_app_failed")
	}
}

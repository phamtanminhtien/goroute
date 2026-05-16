package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (a *App) Run() error {
	a.logger.Info().Str("listen_addr", a.server.Addr).Msg("server_listening")

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		a.logger.Info().Str("signal", sig.String()).Msg("shutdown_signal_received")
	case err := <-errCh:
		a.logger.Error().Err(err).Msg("server_stopped_unexpectedly")
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error().Err(err).Msg("server_shutdown_failed")
		return err
	}
	if err := a.repo.Close(); err != nil {
		a.logger.Error().Err(err).Msg("repository_close_failed")
		return err
	}

	a.logger.Info().Msg("server_shutdown_complete")
	return nil
}

package gormsqlite

import (
	"fmt"
	"strings"

	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

func normalizeWriteError(err error, connectionID, action string) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return connectionsusecase.ErrConflict{ConnectionID: connectionID}
	}

	if connectionID == "" {
		return fmt.Errorf("%s connections: %w", action, err)
	}

	return fmt.Errorf("%s connection %q: %w", action, connectionID, err)
}

func normalizeProviderWriteError(err error, providerID, action string) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("provider %q already exists", providerID)
	}

	if providerID == "" {
		return fmt.Errorf("%s providers: %w", action, err)
	}

	return fmt.Errorf("%s provider %q: %w", action, providerID, err)
}

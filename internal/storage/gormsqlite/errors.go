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

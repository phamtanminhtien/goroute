package connections

import "fmt"

type ErrNotFound struct {
	ConnectionID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("connection %q not found", e.ConnectionID)
}

type ErrConflict struct {
	ConnectionID string
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("connection %q already exists", e.ConnectionID)
}

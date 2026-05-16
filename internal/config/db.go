package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const DefaultDatabaseName = "goroute.db"

func ResolveDatabasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".goroute", DefaultDatabaseName), nil
}

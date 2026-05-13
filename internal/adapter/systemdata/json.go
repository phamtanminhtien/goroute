package systemdata

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
)

type filePayload struct {
	Drivers []driver.Driver `json:"drivers"`
}

func LoadFile(path string) (driver.Catalog, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return driver.Catalog{}, fmt.Errorf("read system data %q: %w", path, err)
	}

	var payload filePayload
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return driver.Catalog{}, fmt.Errorf("decode system data %q: %w", path, err)
	}

	if len(payload.Drivers) == 0 {
		return driver.Catalog{}, fmt.Errorf("system data must contain at least one driver")
	}

	return driver.Catalog{Drivers: payload.Drivers}, nil
}

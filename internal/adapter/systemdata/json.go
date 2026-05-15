package systemdata

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

type filePayload struct {
	Providers []provider.Provider `json:"providers"`
}

func LoadFile(path string) (provider.Catalog, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return provider.Catalog{}, fmt.Errorf("read system data %q: %w", path, err)
	}

	var payload filePayload
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return provider.Catalog{}, fmt.Errorf("decode system data %q: %w", path, err)
	}

	if len(payload.Providers) == 0 {
		return provider.Catalog{}, fmt.Errorf("system data must contain at least one provider")
	}

	return provider.Catalog{Providers: payload.Providers}, nil
}

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type RTKConfig struct {
	Enabled bool
	present bool
}

type rtkJSON struct {
	Enabled bool `json:"enabled"`
}

func (c RTKConfig) MarshalJSON() ([]byte, error) {
	if !c.Enabled {
		return []byte("false"), nil
	}

	return json.Marshal(rtkJSON{Enabled: true})
}

func (c *RTKConfig) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("value is required")
	}

	switch trimmed[0] {
	case 'f', 't':
		var enabled bool
		if err := json.Unmarshal(trimmed, &enabled); err != nil {
			return err
		}

		*c = RTKConfig{
			Enabled: enabled,
			present: true,
		}
		return nil
	case '{':
		var decoded rtkJSON
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return err
		}

		*c = RTKConfig{
			Enabled: decoded.Enabled,
			present: true,
		}
		return nil
	default:
		return fmt.Errorf("expected boolean or object")
	}
}

func (c RTKConfig) IsPresent() bool {
	return c.present
}

func NewRTKConfig(enabled bool) RTKConfig {
	return RTKConfig{
		Enabled: enabled,
		present: true,
	}
}

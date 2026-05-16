package config

import (
	"encoding/json"
	"testing"
)

func TestServerConfigJSONShapeMatchesConfigFile(t *testing.T) {
	bytes, err := json.Marshal(ServerConfig{
		Listen:    ":2232",
		AuthToken: "change-me",
		WebUIDir:  "web/dist",
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	const want = `{"listen":":2232","auth_token":"change-me","web_ui_dir":"web/dist"}`
	if string(bytes) != want {
		t.Fatalf("expected JSON shape %s, got %s", want, string(bytes))
	}
}

package config

import (
	"encoding/json"
	"testing"
)

func TestConnectionConfigJSONShapeMatchesConfigFile(t *testing.T) {
	bytes, err := json.Marshal(ConnectionConfig{
		ID:                   "connection-1",
		ProviderID:           "cx",
		AccessToken:          "access-token",
		RefreshToken:         "refresh-token",
		TokenType:            "Bearer",
		ExpiresIn:            3600,
		AccessTokenExpiresAt: 1710003600,
		Name:                 "user@example.com",
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	const want = `{"id":"connection-1","provider_id":"cx","access_token":"access-token","refresh_token":"refresh-token","token_type":"Bearer","expires_in":3600,"access_token_expires_at":1710003600,"name":"user@example.com"}`
	if string(bytes) != want {
		t.Fatalf("expected JSON shape %s, got %s", want, string(bytes))
	}
}

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

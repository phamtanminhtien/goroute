package config

import (
	"encoding/json"
	"testing"
)

func TestProviderConfigJSONShapeMatchesConfigFile(t *testing.T) {
	bytes, err := json.Marshal(ProviderConfig{
		ID:           "provider-1",
		Type:         "codex",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Name:         "user@example.com",
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	const want = `{"id":"provider-1","type":"codex","access_token":"access-token","refresh_token":"refresh-token","name":"user@example.com"}`
	if string(bytes) != want {
		t.Fatalf("expected JSON shape %s, got %s", want, string(bytes))
	}
}

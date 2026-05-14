package config

import "testing"

func TestValidateRequiresServerAuthToken(t *testing.T) {
	err := Validate(Config{
		Server: ServerConfig{
			Listen:    ":2232",
			AuthToken: "   ",
		},
		Providers: []ProviderConfig{
			{ID: "provider-1", Type: "openai", Name: "user@example.com"},
		},
	})
	if err == nil {
		t.Fatal("expected validation error for missing auth token")
	}
	if got := err.Error(); got != "config.server.auth_token is required" {
		t.Fatalf("expected auth token validation error, got %q", got)
	}
}

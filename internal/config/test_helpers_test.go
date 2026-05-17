package config

import (
	"os"
	"testing"
)

func writeConfigFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func assertConfigFile(t *testing.T, path string, want string) {
	t.Helper()

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(bytes) != want {
		t.Fatalf("expected config file %q, got %q", want, string(bytes))
	}
}

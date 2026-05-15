package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewWithWriterPrettyPrintsOutsideProduction(t *testing.T) {
	var output bytes.Buffer
	logger := NewWithWriter("dev", &output)

	logger.Info().Str("request_id", "req-1").Msg("hello")

	logLine := output.String()
	if strings.HasPrefix(strings.TrimSpace(logLine), "{") {
		t.Fatalf("expected pretty output, got %q", logLine)
	}
	if !strings.Contains(logLine, "hello") {
		t.Fatalf("expected message in pretty output, got %q", logLine)
	}
	if !strings.Contains(logLine, "request_id=req-1") {
		t.Fatalf("expected request_id field in pretty output, got %q", logLine)
	}
}

func TestNewWithWriterEmitsJSONInProduction(t *testing.T) {
	var output bytes.Buffer
	logger := NewWithWriter("prod", &output)

	logger.Info().Str("request_id", "req-1").Msg("hello")

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON log, got error: %v", err)
	}
	if payload["message"] != "hello" {
		t.Fatalf("expected message field, got %#v", payload["message"])
	}
	if payload["request_id"] != "req-1" {
		t.Fatalf("expected request_id field, got %#v", payload["request_id"])
	}
	if payload["service"] != serviceName {
		t.Fatalf("expected service field, got %#v", payload["service"])
	}
}

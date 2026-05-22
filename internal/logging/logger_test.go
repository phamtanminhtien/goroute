package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
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

func TestStreamBroadcasterReceivesPublishedLines(t *testing.T) {
	broadcaster := NewStreamBroadcaster(2)
	stream, unsubscribe := broadcaster.Subscribe()
	defer unsubscribe()

	if _, err := io.WriteString(broadcaster.Writer(), "first line\nsecond line\n"); err != nil {
		t.Fatalf("write stream lines: %v", err)
	}

	select {
	case line := <-stream:
		if line != "first line" {
			t.Fatalf("expected first line, got %q", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first stream line")
	}

	select {
	case line := <-stream:
		if line != "second line" {
			t.Fatalf("expected second line, got %q", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second stream line")
	}
}

func TestStreamBroadcasterUnsubscribeClosesChannel(t *testing.T) {
	broadcaster := NewStreamBroadcaster(1)
	stream, unsubscribe := broadcaster.Subscribe()

	unsubscribe()

	select {
	case _, ok := <-stream:
		if ok {
			t.Fatal("expected stream to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream closure")
	}
}

func TestStreamBroadcasterDropsLinesForSlowSubscribers(t *testing.T) {
	broadcaster := NewStreamBroadcaster(1)
	stream, unsubscribe := broadcaster.Subscribe()
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for index := 0; index < 1000; index++ {
			broadcaster.Publish("busy")
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected publish loop not to block")
	}

	select {
	case <-stream:
	case <-time.After(time.Second):
		t.Fatal("expected at least one buffered line for subscriber")
	}
}

func TestStreamBroadcasterReplaysRecentLinesToNewSubscribers(t *testing.T) {
	broadcaster := NewStreamBroadcaster(2)

	broadcaster.Publish("first line")
	broadcaster.Publish("second line")
	broadcaster.Publish("third line")

	stream, unsubscribe := broadcaster.Subscribe()
	defer unsubscribe()

	for _, expected := range []string{"second line", "third line"} {
		select {
		case line := <-stream:
			if line != expected {
				t.Fatalf("expected replayed line %q, got %q", expected, line)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for replayed line %q", expected)
		}
	}
}

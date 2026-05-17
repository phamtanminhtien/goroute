package rtk

import (
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestCompressTextDedupsRepeatedLogs(t *testing.T) {
	service := NewService()

	result := service.CompressText(strings.Repeat("warn: retrying\n", 120), Hint{
		RequestType: "chat_completions",
		Role:        string(openaiwire.ChatRoleUser),
		FieldPath:   "messages[0].content",
	})

	if !result.Applied {
		t.Fatal("expected compression to apply")
	}
	if !strings.Contains(result.Text, "... (119 duplicate lines)") {
		t.Fatalf("expected duplicate marker, got %q", result.Text)
	}
	if result.BytesAfter >= result.BytesBefore {
		t.Fatalf("expected bytes to shrink, got before=%d after=%d", result.BytesBefore, result.BytesAfter)
	}
}

func TestCompressTextSkipsAssistantContent(t *testing.T) {
	service := NewService()

	result := service.CompressText(strings.Repeat("warn: retrying\n", 120), Hint{
		RequestType: "chat_completions",
		Role:        string(openaiwire.ChatRoleAssistant),
		FieldPath:   "messages[1].content",
	})

	if result.Applied {
		t.Fatalf("expected assistant content to be skipped, got %#v", result)
	}
}

func TestCompressTextSkipsDataURLs(t *testing.T) {
	service := NewService()

	result := service.CompressText(strings.Repeat("data:image/png;base64,abc\n", 120), Hint{
		RequestType: "responses",
		Role:        "tool",
		FieldPath:   "input[0].output",
	})

	if result.Applied {
		t.Fatalf("expected data url content to be skipped, got %#v", result)
	}
}

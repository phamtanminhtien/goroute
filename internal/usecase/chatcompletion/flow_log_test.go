package chatcompletion

import (
	"net/http"
	"testing"
)

func TestThirdPartyResponseBodyForStorageUsesLastSSEPayloadBeforeDone(t *testing.T) {
	headers := http.Header{"Content-Type": []string{"text/event-stream"}}
	body := "data: {\"first\":true}\n\ndata: {\"text\":\"final\"}\n\ndata: [DONE]\n\n"

	got := ThirdPartyResponseBodyForStorage(headers, body)

	if got != `{"text":"final"}` {
		t.Fatalf("unexpected stored response body %q", got)
	}
}

package chatcompletion

import (
	"net/http"
	"testing"
	"time"
)

func TestThirdPartyResponseBodyForStorageUsesLastSSEPayloadBeforeDone(t *testing.T) {
	headers := http.Header{"Content-Type": []string{"text/event-stream"}}
	body := "data: {\"first\":true}\n\ndata: {\"text\":\"final\"}\n\ndata: [DONE]\n\n"

	got := ThirdPartyResponseBodyForStorage(headers, body)

	if got != `{"text":"final"}` {
		t.Fatalf("unexpected stored response body %q", got)
	}
}

func TestSetTranslatedSSEResponseBodyUsesLastSSEPayloadBeforeDone(t *testing.T) {
	recorder := NewFlowRecorder("req-1", time.Time{})
	recorder.SetTranslatedSSEResponseBody("data: {\"first\":true}\n\ndata: {\"type\":\"response.completed\",\"sequence_number\":88}\n\ndata: [DONE]\n\n")

	flow, _ := recorder.SnapshotDetails(time.Time{}, 1)
	if flow.TranslatedResponseBody != `{"sequence_number":88,"type":"response.completed"}` {
		t.Fatalf("unexpected translated response body %q", flow.TranslatedResponseBody)
	}
}

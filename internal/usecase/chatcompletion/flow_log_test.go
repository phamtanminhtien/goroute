package chatcompletion

import (
	"net/http"
	"strings"
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
	if !stringPointerContains(flow.TranslatedResponseBody, `{"sequence_number":88,"type":"response.completed"}`) {
		t.Fatalf("unexpected translated response body %#v", flow.TranslatedResponseBody)
	}
}

func TestSnapshotDetailsNullsStreamBodiesByMode(t *testing.T) {
	recorder := NewFlowRecorder("req-1", time.Time{})
	recorder.SetRequestMode(true)
	recorder.SetProviderRequestMode(true)
	recorder.SetResponseBody(`{"client":true}`)
	recorder.SetTranslatedResponseBody(`{"provider":true}`)
	recorder.AddThirdPartyLog(ThirdPartyLog{
		RequestMode:         RequestModeStream,
		ProviderRequestMode: RequestModeStream,
		ResponseBody:        `{"provider":true}`,
	})

	flow, thirdPartyLogs := recorder.SnapshotDetails(time.Time{}, 1)
	if flow.ResponseBody != nil {
		t.Fatalf("expected stream flow response body to be nil, got %q", *flow.ResponseBody)
	}
	if flow.TranslatedResponseBody != nil {
		t.Fatalf("expected stream translated response body to be nil, got %q", *flow.TranslatedResponseBody)
	}
	if len(thirdPartyLogs) != 1 || thirdPartyLogs[0].ResponseBody != nil {
		t.Fatalf("expected stream third-party response body to be nil, got %#v", thirdPartyLogs)
	}
}

func TestSnapshotDetailsKeepsSyncBodiesByMode(t *testing.T) {
	recorder := NewFlowRecorder("req-1", time.Time{})
	recorder.SetRequestMode(false)
	recorder.SetProviderRequestMode(false)
	recorder.SetResponseBody(`{"client":true}`)
	recorder.SetTranslatedResponseBody(`{"provider":true}`)
	recorder.AddThirdPartyLog(ThirdPartyLog{
		RequestMode:         RequestModeSync,
		ProviderRequestMode: RequestModeSync,
		ResponseBody:        `{"provider":true}`,
	})
	recorder.SetResponseBody(`{"client":true}`)

	flow, thirdPartyLogs := recorder.SnapshotDetails(time.Time{}, 1)
	if !stringPointerContains(flow.ResponseBody, `"client":true`) {
		t.Fatalf("expected sync flow response body, got %#v", flow.ResponseBody)
	}
	if !stringPointerContains(flow.TranslatedResponseBody, `"provider":true`) {
		t.Fatalf("expected sync translated response body, got %#v", flow.TranslatedResponseBody)
	}
	if len(thirdPartyLogs) != 1 || !stringPointerContains(thirdPartyLogs[0].ResponseBody, `"provider":true`) {
		t.Fatalf("expected sync third-party response body, got %#v", thirdPartyLogs)
	}
}

func TestSnapshotDetailsUsesRequestModeForTranslatedResponseAndProviderModeForResponse(t *testing.T) {
	recorder := NewFlowRecorder("req-1", time.Time{})
	recorder.SetRequestMode(true)
	recorder.SetProviderRequestMode(false)
	recorder.SetResponseBody(`{"provider":true}`)
	recorder.SetTranslatedResponseBody(`{"client":true}`)
	recorder.AddThirdPartyLog(ThirdPartyLog{
		RequestMode:         RequestModeStream,
		ProviderRequestMode: RequestModeSync,
		ResponseBody:        `{"provider":true}`,
	})

	flow, thirdPartyLogs := recorder.SnapshotDetails(time.Time{}, 1)
	if !stringPointerContains(flow.ResponseBody, `"provider":true`) {
		t.Fatalf("expected response body to follow provider_request_mode, got %#v", flow.ResponseBody)
	}
	if flow.TranslatedResponseBody != nil {
		t.Fatalf("expected translated response body to follow request_mode, got %#v", flow.TranslatedResponseBody)
	}
	if len(thirdPartyLogs) != 1 || !stringPointerContains(thirdPartyLogs[0].ResponseBody, `"provider":true`) {
		t.Fatalf("expected third-party response body to follow provider_request_mode, got %#v", thirdPartyLogs)
	}
}

func stringPointerContains(value *string, needle string) bool {
	return value != nil && strings.Contains(*value, needle)
}

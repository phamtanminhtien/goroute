package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func TestRecordSSEStreamResultIgnoresClientClose(t *testing.T) {
	recorder := chatcompletion.NewFlowRecorder("req_1", time.Now().UTC())
	writer := newBodyCaptureResponseWriter(httptest.NewRecorder())

	recordSSEStreamResult(recorder, writer, context.Canceled)

	run := recorder.SnapshotRun(time.Now().UTC())
	if run.ErrorType != "" || run.ErrorMessage != "" {
		t.Fatalf("expected client close to be ignored, got error_type=%q error_message=%q", run.ErrorType, run.ErrorMessage)
	}
}

func TestRecordSSEStreamResultRecordsUnexpectedErrors(t *testing.T) {
	recorder := chatcompletion.NewFlowRecorder("req_1", time.Now().UTC())
	writer := newBodyCaptureResponseWriter(httptest.NewRecorder())

	recordSSEStreamResult(recorder, writer, errors.New("upstream read failed"))

	run := recorder.SnapshotRun(time.Now().UTC())
	if run.ErrorType != "stream_error" || run.ErrorMessage != "upstream read failed" {
		t.Fatalf("expected stream_error, got error_type=%q error_message=%q", run.ErrorType, run.ErrorMessage)
	}
}

package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func writeSSEStream(w *bodyCaptureResponseWriter, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, writeErr := w.Write(line); writeErr != nil {
				return writeErr
			}
			if bytes.Equal(line, []byte("\n")) || bytes.Equal(line, []byte("\r\n")) {
				w.Flush()
			}
		}

		if err == nil {
			continue
		}
		if err == io.EOF {
			if len(line) > 0 {
				w.Flush()
			}
			return nil
		}
		return err
	}
}

func recordSSEStreamResult(recorder *chatcompletion.FlowRecorder, w *bodyCaptureResponseWriter, err error) {
	if err == nil {
		recorder.SetTranslatedSSEResponseBody(w.bodyString())
		return
	}
	if isClientStreamCloseError(err) {
		return
	}
	recorder.SetError("stream_error", err.Error())
}

func isClientStreamCloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "context canceled") ||
		strings.Contains(message, "broken pipe") ||
		strings.Contains(message, "connection reset by peer") ||
		strings.Contains(message, "client disconnected")
}

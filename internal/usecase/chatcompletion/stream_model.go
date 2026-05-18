package chatcompletion

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func RewriteChatCompletionsStreamModel(body io.ReadCloser, model string) io.ReadCloser {
	return rewriteSSEData(body, func(payload string) (string, bool) {
		if payload == "[DONE]" {
			return payload, true
		}

		var chunk openaiwire.ChatCompletionsStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return payload, false
		}
		chunk.Model = model

		encoded, err := json.Marshal(chunk)
		if err != nil {
			return payload, false
		}
		return string(encoded), true
	})
}

func RewriteResponsesStreamModel(body io.ReadCloser, model string) io.ReadCloser {
	return rewriteSSEData(body, func(payload string) (string, bool) {
		if payload == "[DONE]" {
			return payload, true
		}

		var event openaiwire.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return payload, false
		}
		if event.Response != nil {
			event.Response.Model = model
		}

		encoded, err := json.Marshal(event)
		if err != nil {
			return payload, false
		}
		return string(encoded), true
	})
}

func rewriteSSEData(body io.ReadCloser, rewrite func(string) (string, bool)) io.ReadCloser {
	reader, writer := io.Pipe()

	go func() {
		defer body.Close()

		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 0, 1024), 10*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if nextPayload, ok := rewrite(payload); ok {
					line = "data: " + nextPayload
				}
			}
			if _, err := fmt.Fprintln(writer, line); err != nil {
				_ = writer.CloseWithError(err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_ = writer.Close()
	}()

	return reader
}

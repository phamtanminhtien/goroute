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

		var event map[string]json.RawMessage
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return payload, false
		}

		rawResponse, ok := event["response"]
		if !ok {
			return payload, true
		}

		var response map[string]json.RawMessage
		if err := json.Unmarshal(rawResponse, &response); err != nil {
			return payload, false
		}

		encodedModel, err := json.Marshal(model)
		if err != nil {
			return payload, false
		}
		response["model"] = encodedModel

		encodedResponse, err := json.Marshal(response)
		if err != nil {
			return payload, false
		}
		event["response"] = encodedResponse

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
		defer writer.Close()

		buffered := bufio.NewReader(body)
		for {
			line, err := buffered.ReadString('\n')
			if err != nil && err != io.EOF {
				_ = writer.CloseWithError(err)
				return
			}
			if line == "" && err == io.EOF {
				return
			}

			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
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

			if err == io.EOF {
				return
			}
		}
	}()

	return reader
}

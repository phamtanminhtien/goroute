package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type chatStreamState struct {
	ID          string
	Created     int64
	Model       string
	SawToolCall bool
	SentRole    bool
	Usage       *openaiwire.CompletionUsage
}

func transformResponsesToChatCompletionsStream(body io.ReadCloser, model string) io.ReadCloser {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		defer body.Close()
		defer pipeWriter.Close()

		state := chatStreamState{
			Created: time.Now().Unix(),
			Model:   model,
		}
		if err := streamResponsesAsChatChunks(body, pipeWriter, &state); err != nil {
			_ = pipeWriter.CloseWithError(err)
		}
	}()

	return pipeReader
}

func streamResponsesAsChatChunks(input io.Reader, output io.Writer, state *chatStreamState) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	var builder strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if builder.Len() > 0 {
				if err := handleResponsesStreamEvent(strings.TrimSuffix(builder.String(), "\n"), output, state); err != nil {
					return err
				}
				builder.Reset()
			}
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		builder.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		builder.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read responses stream: %w", err)
	}
	if builder.Len() > 0 {
		if err := handleResponsesStreamEvent(strings.TrimSuffix(builder.String(), "\n"), output, state); err != nil {
			return err
		}
	}
	return nil
}

func handleResponsesStreamEvent(raw string, output io.Writer, state *chatStreamState) error {
	if raw == "" || raw == "[DONE]" {
		return nil
	}

	var event openaiwire.ResponsesStreamEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return fmt.Errorf("decode responses stream event: %w", err)
	}

	if event.Response != nil {
		if event.Response.ID != "" {
			state.ID = chatCompletionID(event.Response.ID)
		}
		if event.Response.CreatedAt != 0 {
			state.Created = event.Response.CreatedAt
		}
	}

	switch event.Type {
	case openaiwire.ResponsesStreamEventTypeCreated:
		return nil
	case openaiwire.ResponsesStreamEventTypeOutputTextDelta:
		delta := event.Delta
		if delta == "" {
			delta = event.Text
		}
		if delta == "" {
			return nil
		}

		chunk := openaiwire.ChatCompletionsStreamChunk{
			ID:      defaultString(state.ID, fmt.Sprintf("chatcmpl-%d", state.Created)),
			Object:  "chat.completion.chunk",
			Created: state.Created,
			Model:   state.Model,
			Choices: []openaiwire.ChatCompletionsStreamChoice{{
				Index: 0,
				Delta: assistantDelta(delta, state),
			}},
		}
		return writeSSEChunk(output, chunk)
	case openaiwire.ResponsesStreamEventTypeOutputItemDone:
		if event.Item == nil || event.Item.Type != openaiwire.OutputItemTypeFunctionCall {
			return nil
		}
		state.SawToolCall = true
		chunk := openaiwire.ChatCompletionsStreamChunk{
			ID:      defaultString(state.ID, fmt.Sprintf("chatcmpl-%d", state.Created)),
			Object:  "chat.completion.chunk",
			Created: state.Created,
			Model:   state.Model,
			Choices: []openaiwire.ChatCompletionsStreamChoice{{
				Index: 0,
				Delta: assistantToolCallDelta(*event.Item, event.OutputIndex, state),
			}},
		}
		return writeSSEChunk(output, chunk)
	case openaiwire.ResponsesStreamEventTypeCompleted:
		if event.Response != nil && event.Response.Usage != nil {
			state.Usage = &openaiwire.CompletionUsage{
				PromptTokens:     event.Response.Usage.InputTokens,
				CompletionTokens: event.Response.Usage.OutputTokens,
				TotalTokens:      event.Response.Usage.TotalTokens,
			}
			if event.Response.Usage.InputTokensDetails != nil {
				state.Usage.PromptTokensDetails = &openaiwire.PromptTokensDetails{
					CachedTokens: event.Response.Usage.InputTokensDetails.CachedTokens,
				}
			}
		}
		chunk := openaiwire.ChatCompletionsStreamChunk{
			ID:      defaultString(state.ID, fmt.Sprintf("chatcmpl-%d", state.Created)),
			Object:  "chat.completion.chunk",
			Created: state.Created,
			Model:   state.Model,
			Choices: []openaiwire.ChatCompletionsStreamChoice{{
				Index:        0,
				Delta:        openaiwire.Message{},
				FinishReason: finishReason(state),
			}},
			Usage: state.Usage,
		}
		if err := writeSSEChunk(output, chunk); err != nil {
			return err
		}
		_, err := io.WriteString(output, "data: [DONE]\n\n")
		return err
	case openaiwire.ResponsesStreamEventTypeFailed:
		return fmt.Errorf("codex response failed")
	default:
		return nil
	}
}

func assistantDelta(text string, state *chatStreamState) openaiwire.Message {
	delta := openaiwire.Message{Content: text}
	if !state.SentRole {
		delta.Role = openaiwire.ChatRoleAssistant
		state.SentRole = true
	}
	return delta
}

func assistantToolCallDelta(item openaiwire.OutputItem, index int, state *chatStreamState) openaiwire.Message {
	delta := openaiwire.Message{
		ToolCalls: []openaiwire.ToolCall{{
			ID:   defaultString(item.CallID, fmt.Sprintf("call_%d", index)),
			Type: openaiwire.ToolTypeFunction,
			Function: openaiwire.ToolCallFunction{
				Name:      item.Name,
				Arguments: defaultString(item.Arguments, "{}"),
			},
		}},
	}
	if !state.SentRole {
		delta.Role = openaiwire.ChatRoleAssistant
		state.SentRole = true
	}
	return delta
}

func finishReason(state *chatStreamState) openaiwire.FinishReason {
	if state != nil && state.SawToolCall {
		return openaiwire.FinishReasonToolCalls
	}
	return openaiwire.FinishReasonStop
}

func writeSSEChunk(output io.Writer, chunk openaiwire.ChatCompletionsStreamChunk) error {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("encode chat completion chunk: %w", err)
	}
	_, err = fmt.Fprintf(output, "data: %s\n\n", payload)
	return err
}

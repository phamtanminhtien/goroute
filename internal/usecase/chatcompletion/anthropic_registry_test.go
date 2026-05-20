package chatcompletion

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type precedenceConnection struct {
	anthropicCalls int
	responsesCalls int
	chatCalls      int
}

func (c *precedenceConnection) AnthropicMessages(context.Context, anthropicwire.MessagesRequest, routing.Target) (anthropicwire.MessagesResponse, error) {
	c.anthropicCalls++
	return anthropicwire.MessagesResponse{ID: "msg_1", Type: "message", Role: "assistant", Model: "claude", Content: []anthropicwire.ContentBlock{{Type: "text", Text: "native"}}}, nil
}

func (c *precedenceConnection) AnthropicMessagesStream(context.Context, anthropicwire.MessagesRequest, routing.Target) (io.ReadCloser, error) {
	c.anthropicCalls++
	return io.NopCloser(strings.NewReader("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")), nil
}

func (c *precedenceConnection) Responses(context.Context, openaiwire.ResponsesRequest, routing.Target) (openaiwire.ResponsesResponse, error) {
	c.responsesCalls++
	return openaiwire.ResponsesResponse{ID: "resp_1", Object: "response", Model: "gpt", Status: openaiwire.ResponsesStatusCompleted, Output: []openaiwire.OutputItem{{Type: openaiwire.OutputItemTypeMessage, Role: "assistant", Content: []openaiwire.OutputContent{{Type: openaiwire.OutputContentTypeOutputText, Text: "responses"}}}}}, nil
}

func (c *precedenceConnection) ResponsesStream(context.Context, openaiwire.ResponsesRequest, routing.Target) (io.ReadCloser, error) {
	c.responsesCalls++
	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func (c *precedenceConnection) ChatCompletions(context.Context, openaiwire.ChatCompletionsRequest, routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	c.chatCalls++
	return openaiwire.ChatCompletionsResponse{ID: "chatcmpl-1", Object: "chat.completion", Model: "gpt", Choices: []openaiwire.ChatCompletionChoice{{Index: 0, Message: openaiwire.Message{Role: openaiwire.ChatRoleAssistant, Content: "chat"}, FinishReason: openaiwire.FinishReasonStop}}}, nil
}

func (c *precedenceConnection) ChatCompletionsStream(context.Context, openaiwire.ChatCompletionsRequest, routing.Target) (io.ReadCloser, error) {
	c.chatCalls++
	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func TestAnthropicMessagesPrefersNativeThenResponsesThenChatCompletions(t *testing.T) {
	req := anthropicwire.MessagesRequest{Model: "cx/gpt-5.4", Messages: []anthropicwire.MessageParam{{Role: "user", Content: anthropicwire.TextContent("hello")}}}
	target := routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "gpt-5.4"}

	native := &precedenceConnection{}
	registry := NewConnectionRegistry(map[string][]ConnectionEntry{"cx": {{ID: "native", ProviderID: "cx", ProtocolConnections: ProtocolConnections{Anthropic: native, Responses: native, ChatCompletions: native}}}})
	if _, err := registry.AnthropicMessages(context.Background(), req, target); err != nil {
		t.Fatalf("native route: %v", err)
	}
	if native.anthropicCalls != 1 || native.responsesCalls != 0 || native.chatCalls != 0 {
		t.Fatalf("expected native only, got %#v", native)
	}

	responses := &precedenceConnection{}
	registry = NewConnectionRegistry(map[string][]ConnectionEntry{"cx": {{ID: "responses", ProviderID: "cx", ProtocolConnections: ProtocolConnections{Responses: responses, ChatCompletions: responses}}}})
	if _, err := registry.AnthropicMessages(context.Background(), req, target); err != nil {
		t.Fatalf("responses route: %v", err)
	}
	if responses.responsesCalls != 1 || responses.chatCalls != 0 || responses.anthropicCalls != 0 {
		t.Fatalf("expected responses only, got %#v", responses)
	}

	chat := &precedenceConnection{}
	registry = NewConnectionRegistry(map[string][]ConnectionEntry{"cx": {{ID: "chat", ProviderID: "cx", ProtocolConnections: ProtocolConnections{ChatCompletions: chat}}}})
	if _, err := registry.AnthropicMessages(context.Background(), req, target); err != nil {
		t.Fatalf("chat route: %v", err)
	}
	if chat.chatCalls != 1 || chat.responsesCalls != 0 || chat.anthropicCalls != 0 {
		t.Fatalf("expected chat only, got %#v", chat)
	}
}

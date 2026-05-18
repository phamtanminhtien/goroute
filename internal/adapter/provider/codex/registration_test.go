package codex

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
)

func TestRegistrationBuildConnectionProvidesResponsesClient(t *testing.T) {
	registration := Registration()
	connectionConfig := connection.Record{
		ID:         "cx-1",
		ProviderID: "cx",
		Name:       "Codex Primary",
		APIKey:     "token",
	}

	protocols, err := registration.BuildConnection(connectionConfig)
	if err != nil {
		t.Fatalf("BuildConnection returned error: %v", err)
	}
	if protocols.Responses == nil {
		t.Fatal("expected responses capability to be registered")
	}
	if protocols.ChatCompletions != nil {
		t.Fatal("expected chat completions capability to remain unset")
	}

	client, ok := protocols.Responses.(*Client)
	if !ok {
		t.Fatalf("expected responses client to be *Client, got %T", protocols.Responses)
	}
	if client.connection.ID != connectionConfig.ID {
		t.Fatalf("expected connection id %q, got %q", connectionConfig.ID, client.connection.ID)
	}
	if client.connection.APIKey != connectionConfig.APIKey {
		t.Fatalf("expected api key to be preserved, got %q", client.connection.APIKey)
	}
}

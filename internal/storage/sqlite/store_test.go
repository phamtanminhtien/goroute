package sqlite

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func TestOpenMigratesEmptyDatabase(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "goroute.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	connections, err := store.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections returned error: %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("expected empty connection list, got %#v", connections)
	}
}

func TestStoreConnectionCRUD(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "goroute.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	created := connection.Record{
		ID:          "openai-1",
		ProviderID:  "openai",
		Name:        "openai-user",
		APIKey:      "token-1",
		AccessToken: "access-1",
	}
	if err := store.CreateConnection(created); err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}

	got, ok, err := store.GetConnection("openai-1")
	if err != nil {
		t.Fatalf("GetConnection returned error: %v", err)
	}
	if !ok || got.Name != "openai-user" || got.APIKey != "token-1" {
		t.Fatalf("unexpected stored connection: ok=%v value=%#v", ok, got)
	}

	updated := created
	updated.ID = "openai-renamed"
	updated.Name = "openai-admin"
	updated.APIKey = ""
	if err := store.UpdateConnection("openai-1", updated); err != nil {
		t.Fatalf("UpdateConnection returned error: %v", err)
	}

	got, ok, err = store.GetConnection("openai-renamed")
	if err != nil {
		t.Fatalf("GetConnection after update returned error: %v", err)
	}
	if !ok || got.Name != "openai-admin" || got.AccessToken != "access-1" {
		t.Fatalf("unexpected updated connection: ok=%v value=%#v", ok, got)
	}

	listed, err := store.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections after update returned error: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "openai-renamed" {
		t.Fatalf("unexpected list output: %#v", listed)
	}

	if err := store.DeleteConnection("openai-renamed"); err != nil {
		t.Fatalf("DeleteConnection returned error: %v", err)
	}

	_, ok, err = store.GetConnection("openai-renamed")
	if err != nil {
		t.Fatalf("GetConnection after delete returned error: %v", err)
	}
	if ok {
		t.Fatal("expected deleted connection to be missing")
	}
}

func TestStoreEnforcesUniqueConnectionIDs(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "goroute.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	record := connection.Record{ID: "cx-1", ProviderID: "cx", Name: "codex-user", AccessToken: "token"}
	if err := store.CreateConnection(record); err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}
	if err := store.CreateConnection(record); err == nil || !strings.Contains(err.Error(), "UNIQUE") {
		t.Fatalf("expected unique constraint error, got %v", err)
	}
}

func TestRequestHistoryPersistsAcrossReopen(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	store, err := Open(databasePath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	startedAt := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(1500 * time.Millisecond)
	record := chatcompletion.RequestAttemptHistory{
		RequestID:      "req-1",
		RequestPath:    "/v1/chat/completions",
		RequestedModel: "cx/gpt-5.4",
		ResolvedTarget: "cx/gpt-5.4",
		ProviderID:     "cx",
		ProviderName:   "Codex",
		Stream:         false,
		Status:         chatcompletion.RequestStatusStarted,
		MessageCount:   1,
		ToolCount:      0,
		StartedAt:      startedAt,
		UpdatedAt:      startedAt,
	}
	record, err = store.CreateRequestAttemptHistory(record)
	if err != nil {
		t.Fatalf("CreateRequestAttemptHistory returned error: %v", err)
	}

	record.Status = chatcompletion.RequestStatusSuccess
	record.FinalStatus = chatcompletion.RequestStatusSuccess
	record.AttemptCount = 1
	record.LastConnectionID = "codex-1"
	record.LastConnectionName = "codex-user"
	record.LastAttemptAt = completedAt
	record.CompletedAt = completedAt
	record.UpdatedAt = completedAt
	record.Attempts = []chatcompletion.RequestAttempt{{
		ConnectionID:   "codex-1",
		ConnectionName: "codex-user",
		AttemptIndex:   0,
		Outcome:        "success",
		ErrorCategory:  "none",
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		LatencyMillis:  1500,
	}}
	if err := store.UpdateRequestAttemptHistory(record); err != nil {
		t.Fatalf("UpdateRequestAttemptHistory returned error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	reopened, err := Open(databasePath)
	if err != nil {
		t.Fatalf("reopen store returned error: %v", err)
	}
	defer reopened.Close()

	history, err := reopened.RecentRequestAttempts(1)
	if err != nil {
		t.Fatalf("RecentRequestAttempts returned error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected one history record, got %#v", history)
	}
	if history[0].RequestedModel != "cx/gpt-5.4" || history[0].FinalStatus != "success" || history[0].Status != "success" {
		t.Fatalf("unexpected stored history: %#v", history[0])
	}
	if len(history[0].Attempts) != 1 || history[0].Attempts[0].ConnectionID != "codex-1" {
		t.Fatalf("unexpected stored attempts: %#v", history[0].Attempts)
	}
	if history[0].AttemptCount != 1 || history[0].LastConnectionID != "codex-1" {
		t.Fatalf("unexpected request history metadata: %#v", history[0])
	}
}

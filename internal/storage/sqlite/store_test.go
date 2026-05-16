package sqlite

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
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

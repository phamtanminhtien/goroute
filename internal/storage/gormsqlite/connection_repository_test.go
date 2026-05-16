package gormsqlite

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

func TestOpenMigratesEmptyDatabase(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "goroute.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()

	connections, err := repo.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections returned error: %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("expected empty connection list, got %#v", connections)
	}
}

func TestRepositoryConnectionCRUD(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "goroute.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()

	created := connection.Record{
		ID:                   "openai-1",
		ProviderID:           "openai",
		Name:                 "openai-user",
		APIKey:               "token-1",
		AccessToken:          "access-1",
		RefreshToken:         "refresh-1",
		TokenType:            "Bearer",
		ExpiresIn:            3600,
		AccessTokenExpiresAt: 1700000000,
	}
	if err := repo.CreateConnection(created); err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}

	got, ok, err := repo.GetConnection("openai-1")
	if err != nil {
		t.Fatalf("GetConnection returned error: %v", err)
	}
	if !ok || got.Name != "openai-user" || got.APIKey != "token-1" || got.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected stored connection: ok=%v value=%#v", ok, got)
	}

	updated := created
	updated.ID = "openai-renamed"
	updated.Name = "openai-admin"
	updated.APIKey = ""
	if err := repo.UpdateConnection("openai-1", updated); err != nil {
		t.Fatalf("UpdateConnection returned error: %v", err)
	}

	got, ok, err = repo.GetConnection("openai-renamed")
	if err != nil {
		t.Fatalf("GetConnection after update returned error: %v", err)
	}
	if !ok || got.Name != "openai-admin" || got.AccessToken != "access-1" || got.APIKey != "" {
		t.Fatalf("unexpected updated connection: ok=%v value=%#v", ok, got)
	}

	listed, err := repo.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections after update returned error: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "openai-renamed" {
		t.Fatalf("unexpected list output: %#v", listed)
	}

	if err := repo.DeleteConnection("openai-renamed"); err != nil {
		t.Fatalf("DeleteConnection returned error: %v", err)
	}

	_, ok, err = repo.GetConnection("openai-renamed")
	if err != nil {
		t.Fatalf("GetConnection after delete returned error: %v", err)
	}
	if ok {
		t.Fatal("expected deleted connection to be missing")
	}
}

func TestRepositoryEnforcesUniqueConnectionIDs(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "goroute.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repo.Close()

	record := connection.Record{ID: "cx-1", ProviderID: "cx", Name: "codex-user", AccessToken: "token"}
	if err := repo.CreateConnection(record); err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}

	err = repo.CreateConnection(record)
	var conflict connectionsusecase.ErrConflict
	if err == nil || !errors.As(err, &conflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

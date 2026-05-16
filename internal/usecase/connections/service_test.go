package connections

import (
	"context"
	"errors"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
)

type stubRepository struct {
	items []connection.Record
}

func (r *stubRepository) CreateConnection(item connection.Record) error {
	r.items = append(r.items, item)
	return nil
}

func (r *stubRepository) UpdateConnection(previousID string, item connection.Record) error {
	for i, current := range r.items {
		if current.ID == previousID {
			r.items[i] = item
			return nil
		}
	}
	return errors.New("not found")
}

func (r *stubRepository) DeleteConnection(id string) error {
	for i, current := range r.items {
		if current.ID == id {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func (r *stubRepository) ReplaceConnections(items []connection.Record) error {
	r.items = append([]connection.Record(nil), items...)
	return nil
}

type stubRuntime struct {
	err       error
	snapshots [][]connection.Record
}

func (r *stubRuntime) ReplaceConnections(items []connection.Record) error {
	snapshot := append([]connection.Record(nil), items...)
	r.snapshots = append(r.snapshots, snapshot)
	return r.err
}

type stubProviders struct{}

func (stubProviders) ValidateConnection(connection.Record) []string {
	return nil
}

func (stubProviders) GetUsage(context.Context, connection.Record) (providerregistry.UsageInfo, error) {
	return providerregistry.UsageInfo{}, nil
}

func (stubProviders) GenerateOAuthURL(connection.Record) (string, error) {
	return "", nil
}

func (stubProviders) StartOAuth(connection.Record) (providerregistry.OAuthSession, error) {
	return providerregistry.OAuthSession{}, nil
}

func (stubProviders) CompleteOAuth(connection.Record, map[string]string, string) (providerregistry.OAuthResult, error) {
	return providerregistry.OAuthResult{}, nil
}

func TestServiceRestoresRepositoryStateWhenRuntimeReloadFails(t *testing.T) {
	initial := []connection.Record{{
		ID:         "openai-1",
		ProviderID: "openai",
		Name:       "openai-user",
		APIKey:     "token",
	}}
	repo := &stubRepository{items: append([]connection.Record(nil), initial...)}
	runtime := &stubRuntime{err: errors.New("runtime rebuild failed")}
	service := NewService(initial, repo, runtime, stubProviders{}, nil)

	_, err := service.Create(connection.Record{
		ID:          "cx-1",
		ProviderID:  "cx",
		Name:        "codex-user",
		AccessToken: "oauth-token",
	})
	if err == nil || err.Error() != "runtime rebuild failed" {
		t.Fatalf("expected runtime error, got %v", err)
	}

	if len(repo.items) != 1 || repo.items[0].ID != "openai-1" {
		t.Fatalf("expected repository rollback to original snapshot, got %#v", repo.items)
	}
	if items := service.List(); len(items) != 1 || items[0].ID != "openai-1" {
		t.Fatalf("expected service cache to remain unchanged, got %#v", items)
	}
	if len(runtime.snapshots) != 2 {
		t.Fatalf("expected failed reload and restore reload attempts, got %#v", runtime.snapshots)
	}
}

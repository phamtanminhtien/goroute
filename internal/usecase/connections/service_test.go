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

func (r *stubRepository) ListConnections() ([]connection.Record, error) {
	return append([]connection.Record(nil), r.items...), nil
}

func (r *stubRepository) GetConnection(id string) (connection.Record, bool, error) {
	for _, current := range r.items {
		if current.ID == id {
			return current, true, nil
		}
	}

	return connection.Record{}, false, nil
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

func (r *stubRepository) Close() error {
	return nil
}

type stubRuntime struct {
	err     error
	reloads int
}

func (r *stubRuntime) ReloadConnections() error {
	r.reloads++
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

func TestServiceKeepsRepositoryStateWhenRuntimeReloadFails(t *testing.T) {
	initial := []connection.Record{{
		ID:         "openai-1",
		ProviderID: "openai",
		Name:       "openai-user",
		APIKey:     "token",
	}}
	repo := &stubRepository{items: append([]connection.Record(nil), initial...)}
	runtime := &stubRuntime{err: errors.New("runtime rebuild failed")}
	service := NewService(repo, runtime, stubProviders{}, nil)

	_, err := service.Create(connection.Record{
		ID:          "cx-1",
		ProviderID:  "cx",
		Name:        "codex-user",
		AccessToken: "oauth-token",
	})
	if err == nil || err.Error() != "runtime rebuild failed" {
		t.Fatalf("expected runtime error, got %v", err)
	}

	if len(repo.items) != 2 || repo.items[1].ID != "cx-1" {
		t.Fatalf("expected repository to remain source of truth, got %#v", repo.items)
	}
	if items := service.List(); len(items) != 2 || items[1].ID != "cx-1" {
		t.Fatalf("expected service reads to reflect repository state, got %#v", items)
	}
	if runtime.reloads != 1 {
		t.Fatalf("expected one runtime reload attempt, got %d", runtime.reloads)
	}
}

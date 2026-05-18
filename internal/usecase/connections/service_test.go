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

func (r *stubRepository) SetProviderConnectionsEnabled(providerID string, enabled bool) ([]connection.Record, error) {
	updated := make([]connection.Record, 0)
	for i, current := range r.items {
		if current.ProviderID != providerID {
			continue
		}
		r.items[i].Enabled = enabled
		updated = append(updated, r.items[i])
	}
	return updated, nil
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
		Enabled:    true,
	}}
	repo := &stubRepository{items: append([]connection.Record(nil), initial...)}
	runtime := &stubRuntime{err: errors.New("runtime rebuild failed")}
	service := NewService(repo, runtime, stubProviders{}, nil)

	_, err := service.Create(connection.Record{
		ID:          "cx-1",
		ProviderID:  "cx",
		Name:        "codex-user",
		AccessToken: "oauth-token",
		Enabled:     true,
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

func TestServiceBulkUpdatesProviderConnectionEnabledState(t *testing.T) {
	repo := &stubRepository{items: []connection.Record{
		{ID: "cx-1", ProviderID: "cx", Name: "primary", Enabled: true},
		{ID: "cx-2", ProviderID: "cx", Name: "secondary", Enabled: true},
		{ID: "openai-1", ProviderID: "openai", Name: "other", Enabled: true},
	}}
	runtime := &stubRuntime{}
	service := NewService(repo, runtime, stubProviders{}, nil)

	items, err := service.SetProviderConnectionsEnabled("cx", false)
	if err != nil {
		t.Fatalf("SetProviderConnectionsEnabled returned error: %v", err)
	}
	if len(items) != 2 || items[0].Enabled || items[1].Enabled {
		t.Fatalf("expected cx connections to be disabled, got %#v", items)
	}
	if !repo.items[2].Enabled {
		t.Fatalf("expected other provider connection to stay enabled: %#v", repo.items)
	}
	if runtime.reloads != 1 {
		t.Fatalf("expected one runtime reload, got %d", runtime.reloads)
	}
}

package providers

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

type stubProviderRepo struct {
	connections []connection.Record
	models      []provider.ModelRecord
	providers   []provider.Record
}

func (r *stubProviderRepo) ListProviders() ([]provider.Record, error) {
	return append([]provider.Record(nil), r.providers...), nil
}

func (r *stubProviderRepo) GetProvider(id string) (provider.Record, bool, error) {
	for _, current := range r.providers {
		if current.ID == id {
			return current, true, nil
		}
	}
	return provider.Record{}, false, nil
}

func (r *stubProviderRepo) GetConnection(id string) (connection.Record, bool, error) {
	for _, current := range r.connections {
		if current.ID == id {
			return current, true, nil
		}
	}
	return connection.Record{}, false, nil
}

func (r *stubProviderRepo) CreateProviderWithConnection(providerRecord provider.Record, connectionRecord connection.Record, modelRecord provider.ModelRecord) error {
	r.providers = append(r.providers, providerRecord)
	r.connections = append(r.connections, connectionRecord)
	r.models = append(r.models, modelRecord)
	return nil
}

func (r *stubProviderRepo) UpdateProviderWithConnection(id string, providerRecord provider.Record, connectionRecord connection.Record, modelRecord provider.ModelRecord) error {
	for index, current := range r.providers {
		if current.ID == id {
			r.providers[index] = providerRecord
			break
		}
	}
	for index, current := range r.connections {
		if current.ID == id {
			r.connections[index] = connectionRecord
			break
		}
	}
	for _, current := range r.models {
		if current.ProviderID == modelRecord.ProviderID && current.ID == modelRecord.ID {
			return nil
		}
	}
	r.models = append(r.models, modelRecord)
	return nil
}

func (r *stubProviderRepo) DeleteProviderWithManagedConnection(id string) error {
	for index, current := range r.providers {
		if current.ID == id {
			r.providers = append(r.providers[:index], r.providers[index+1:]...)
			break
		}
	}
	for index, current := range r.connections {
		if current.ID == id {
			r.connections = append(r.connections[:index], r.connections[index+1:]...)
			break
		}
	}
	return nil
}

type stubProviderRuntime struct {
	connectionReloads int
	providerReloads   int
	custom            map[string]bool
	system            map[string]bool
}

func (r *stubProviderRuntime) ReloadProviders() error {
	r.providerReloads++
	return nil
}

func (r *stubProviderRuntime) ReloadConnections() error {
	r.connectionReloads++
	return nil
}

func (r *stubProviderRuntime) IsSystemProvider(providerID string) bool {
	return r.system[providerID]
}

func (r *stubProviderRuntime) IsCustomProvider(providerID string) bool {
	return r.custom[providerID]
}

func TestServiceCreatesCustomProviderWithManagedConnection(t *testing.T) {
	repo := &stubProviderRepo{}
	runtime := &stubProviderRuntime{system: map[string]bool{"openai": true}}
	service := NewService(repo, runtime)

	created, err := service.Create(MutationInput{
		ID:           "openrouter",
		Name:         "OpenRouter",
		AdapterType:  provider.AdapterTypeOpenAICompatible,
		BaseURL:      "https://openrouter.ai/api/",
		DefaultModel: "openrouter/openai/gpt-4.1",
		APIKey:       "secret",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "openrouter" || created.Category != CustomCategory || created.BaseURL != "https://openrouter.ai/api" {
		t.Fatalf("unexpected provider %#v", created)
	}
	if len(repo.connections) != 1 {
		t.Fatalf("expected managed connection, got %#v", repo.connections)
	}
	if repo.connections[0].ID != "openrouter" || repo.connections[0].ProviderID != "openrouter" || repo.connections[0].Name != "OpenRouter" || repo.connections[0].APIKey != "secret" {
		t.Fatalf("unexpected managed connection %#v", repo.connections[0])
	}
	if len(repo.models) != 1 {
		t.Fatalf("expected default model record, got %#v", repo.models)
	}
	if repo.models[0].ID != "openrouter/openai/gpt-4.1" || repo.models[0].ProviderID != "openrouter" || repo.models[0].Name != "openai/gpt-4.1" {
		t.Fatalf("unexpected default model record %#v", repo.models[0])
	}
	if runtime.providerReloads != 1 || runtime.connectionReloads != 1 {
		t.Fatalf("expected runtime reloads, got providers=%d connections=%d", runtime.providerReloads, runtime.connectionReloads)
	}
}

func TestServiceUpdatesCustomProviderAndPreservesBlankAPIKey(t *testing.T) {
	repo := &stubProviderRepo{
		providers: []provider.Record{{
			ID:           "openrouter",
			Name:         "OpenRouter",
			AuthType:     provider.AuthTypeAPIKey,
			Category:     CustomCategory,
			AdapterType:  provider.AdapterTypeOpenAICompatible,
			BaseURL:      "https://openrouter.ai/api",
			DefaultModel: "openrouter/openai/gpt-4.1",
		}},
		connections: []connection.Record{{
			ID:         "openrouter",
			ProviderID: "openrouter",
			Name:       "OpenRouter",
			APIKey:     "existing-secret",
			Enabled:    true,
		}},
		models: []provider.ModelRecord{{
			ID:         "openrouter/openai/gpt-4.1",
			ProviderID: "openrouter",
			Name:       "openai/gpt-4.1",
		}},
	}
	runtime := &stubProviderRuntime{custom: map[string]bool{"openrouter": true}}
	service := NewService(repo, runtime)

	updated, err := service.Update("openrouter", MutationInput{
		Name:         "OpenRouter Prod",
		AdapterType:  provider.AdapterTypeOpenAICompatible,
		BaseURL:      "https://openrouter.ai/api",
		DefaultModel: "openrouter/openai/gpt-4.1",
		Enabled:      false,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Name != "OpenRouter Prod" || repo.connections[0].Name != "OpenRouter Prod" {
		t.Fatalf("expected provider and connection name sync, provider=%#v connection=%#v", updated, repo.connections[0])
	}
	if repo.connections[0].APIKey != "existing-secret" || repo.connections[0].Enabled {
		t.Fatalf("expected secret preservation and enabled update, got %#v", repo.connections[0])
	}
	if len(repo.models) != 1 {
		t.Fatalf("expected existing default model record to be reused, got %#v", repo.models)
	}
}

func TestServiceUpdateCreatesDefaultModelRecordWhenDefaultModelChanges(t *testing.T) {
	repo := &stubProviderRepo{
		providers: []provider.Record{{
			ID:           "openrouter",
			Name:         "OpenRouter",
			AuthType:     provider.AuthTypeAPIKey,
			Category:     CustomCategory,
			AdapterType:  provider.AdapterTypeOpenAICompatible,
			BaseURL:      "https://openrouter.ai/api",
			DefaultModel: "openrouter/openai/gpt-4.1",
		}},
		connections: []connection.Record{{
			ID:         "openrouter",
			ProviderID: "openrouter",
			Name:       "OpenRouter",
			APIKey:     "existing-secret",
			Enabled:    true,
		}},
		models: []provider.ModelRecord{{
			ID:         "openrouter/openai/gpt-4.1",
			ProviderID: "openrouter",
			Name:       "openai/gpt-4.1",
		}},
	}
	runtime := &stubProviderRuntime{custom: map[string]bool{"openrouter": true}}
	service := NewService(repo, runtime)

	_, err := service.Update("openrouter", MutationInput{
		Name:         "OpenRouter",
		AdapterType:  provider.AdapterTypeOpenAICompatible,
		BaseURL:      "https://openrouter.ai/api",
		DefaultModel: "openrouter/anthropic/claude-3.7-sonnet",
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if len(repo.models) != 2 {
		t.Fatalf("expected new default model record, got %#v", repo.models)
	}
	created := repo.models[1]
	if created.ID != "openrouter/anthropic/claude-3.7-sonnet" || created.ProviderID != "openrouter" || created.Name != "anthropic/claude-3.7-sonnet" {
		t.Fatalf("unexpected new default model record %#v", created)
	}
}

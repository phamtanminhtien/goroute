package providers

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

const CustomCategory = "custom"

type Repository interface {
	ListProviders() ([]provider.Record, error)
	GetProvider(string) (provider.Record, bool, error)
	GetConnection(string) (connection.Record, bool, error)
	CreateProviderWithConnection(provider.Record, connection.Record, provider.ModelRecord) error
	UpdateProviderWithConnection(string, provider.Record, connection.Record, provider.ModelRecord) error
	DeleteProviderWithManagedConnection(string) error
}

type Runtime interface {
	ReloadProviders() error
	ReloadConnections() error
	IsSystemProvider(string) bool
	IsCustomProvider(string) bool
}

type Service struct {
	mu      sync.Mutex
	repo    Repository
	runtime Runtime
}

type MutationInput struct {
	ID           string
	Name         string
	AdapterType  provider.AdapterType
	BaseURL      string
	DefaultModel string
	APIKey       string
	Enabled      bool
}

func NewService(repo Repository, runtime Runtime) *Service {
	return &Service{repo: repo, runtime: runtime}
}

func (s *Service) Create(input MutationInput) (provider.Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerRecord, connectionRecord, modelRecord, err := s.buildRecords(input, false, connection.Record{})
	if err != nil {
		return provider.Provider{}, err
	}
	if s.runtime.IsSystemProvider(providerRecord.ID) {
		return provider.Provider{}, fmt.Errorf("provider %q already exists", providerRecord.ID)
	}
	if _, ok, err := s.repo.GetProvider(providerRecord.ID); err != nil {
		return provider.Provider{}, err
	} else if ok {
		return provider.Provider{}, fmt.Errorf("provider %q already exists", providerRecord.ID)
	}
	if _, ok, err := s.repo.GetConnection(providerRecord.ID); err != nil {
		return provider.Provider{}, err
	} else if ok {
		return provider.Provider{}, fmt.Errorf("connection %q already exists", providerRecord.ID)
	}

	if err := s.repo.CreateProviderWithConnection(providerRecord, connectionRecord, modelRecord); err != nil {
		return provider.Provider{}, err
	}
	if err := s.runtime.ReloadProviders(); err != nil {
		return provider.Provider{}, err
	}
	if err := s.runtime.ReloadConnections(); err != nil {
		return provider.Provider{}, err
	}

	return providerRecord.Provider(), nil
}

func (s *Service) Update(id string, input MutationInput) (provider.Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if !s.runtime.IsCustomProvider(id) {
		return provider.Provider{}, fmt.Errorf("custom provider %q not found", id)
	}
	existing, ok, err := s.repo.GetProvider(id)
	if err != nil {
		return provider.Provider{}, err
	}
	if !ok {
		return provider.Provider{}, fmt.Errorf("provider %q not found", id)
	}
	existingConnection, ok, err := s.repo.GetConnection(id)
	if err != nil {
		return provider.Provider{}, err
	}
	if !ok {
		return provider.Provider{}, fmt.Errorf("managed connection %q not found", id)
	}

	input.ID = id
	providerRecord, connectionRecord, modelRecord, err := s.buildRecords(input, true, existingConnection)
	if err != nil {
		return provider.Provider{}, err
	}
	providerRecord.CreatedAt = existing.CreatedAt

	if err := s.repo.UpdateProviderWithConnection(id, providerRecord, connectionRecord, modelRecord); err != nil {
		return provider.Provider{}, err
	}
	if err := s.runtime.ReloadProviders(); err != nil {
		return provider.Provider{}, err
	}
	if err := s.runtime.ReloadConnections(); err != nil {
		return provider.Provider{}, err
	}

	return providerRecord.Provider(), nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if !s.runtime.IsCustomProvider(id) {
		return fmt.Errorf("custom provider %q not found", id)
	}
	if err := s.repo.DeleteProviderWithManagedConnection(id); err != nil {
		return err
	}
	if err := s.runtime.ReloadProviders(); err != nil {
		return err
	}
	return s.runtime.ReloadConnections()
}

func (s *Service) buildRecords(input MutationInput, preserveSecrets bool, existing connection.Record) (provider.Record, connection.Record, provider.ModelRecord, error) {
	id := strings.TrimSpace(input.ID)
	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	defaultModel := strings.TrimSpace(input.DefaultModel)
	adapterType := input.AdapterType
	if adapterType == "" {
		adapterType = provider.AdapterTypeOpenAICompatible
	}

	if id == "" || strings.Contains(id, "/") {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("provider id is required and must not contain /")
	}
	if name == "" {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("name is required")
	}
	if adapterType != provider.AdapterTypeOpenAICompatible && adapterType != provider.AdapterTypeAnthropicCompatible {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("unsupported adapter_type %q", adapterType)
	}
	if baseURL == "" {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("base_url is required")
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("base_url must be an absolute URL")
	}
	if defaultModel == "" {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("default_model is required")
	}
	if !strings.HasPrefix(defaultModel, id+"/") || strings.TrimPrefix(defaultModel, id+"/") == "" {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("default_model must start with provider prefix %q", id+"/")
	}

	apiKey := strings.TrimSpace(input.APIKey)
	if preserveSecrets && apiKey == "" {
		apiKey = existing.APIKey
	}
	if apiKey == "" {
		return provider.Record{}, connection.Record{}, provider.ModelRecord{}, fmt.Errorf("api_key is required")
	}

	providerRecord := provider.Record{
		ID:           id,
		Name:         name,
		AuthType:     provider.AuthTypeAPIKey,
		Category:     CustomCategory,
		AdapterType:  adapterType,
		BaseURL:      baseURL,
		DefaultModel: defaultModel,
	}
	connectionRecord := connection.Record{
		ID:                id,
		ProviderID:        id,
		Name:              name,
		APIKey:            apiKey,
		Enabled:           input.Enabled,
		LastErrorMessage:  existing.LastErrorMessage,
		LastErrorCategory: existing.LastErrorCategory,
		LastErrorAt:       existing.LastErrorAt,
		RetryAfter:        existing.RetryAfter,
	}
	modelRecord := provider.ModelRecord{
		ID:         defaultModel,
		ProviderID: id,
		Name:       strings.TrimPrefix(defaultModel, id+"/"),
	}

	return providerRecord, connectionRecord, modelRecord, nil
}

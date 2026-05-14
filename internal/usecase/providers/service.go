package providers

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/phamtanminhtien/goroute/internal/config"
)

type Runtime interface {
	ReplaceProviders([]config.ProviderConfig) error
}

type Service struct {
	mu         sync.RWMutex
	configPath string
	config     config.Config
	runtime    Runtime
	logger     *log.Logger
}

func NewService(configPath string, cfg config.Config, runtime Runtime, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}

	return &Service{
		configPath: configPath,
		config:     cfg,
		runtime:    runtime,
		logger:     logger,
	}
}

type Item struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	APIKey          string   `json:"api_key,omitempty"`
	AccessToken     string   `json:"access_token,omitempty"`
	RefreshToken    string   `json:"refresh_token,omitempty"`
	HasAPIKey       bool     `json:"has_api_key"`
	HasAccessToken  bool     `json:"has_access_token"`
	HasRefreshToken bool     `json:"has_refresh_token"`
	Status          string   `json:"status"`
	Problems        []string `json:"problems"`
}

func (s *Service) List() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Item, 0, len(s.config.Providers))
	for _, provider := range s.config.Providers {
		items = append(items, redactProvider(provider))
	}

	return items
}

func (s *Service) Get(id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, provider := range s.config.Providers {
		if provider.ID == id {
			return redactProvider(provider), true
		}
	}

	return Item{}, false
}

func (s *Service) Create(input config.ProviderConfig) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizeProvider(input)
	for _, provider := range s.config.Providers {
		if provider.ID == input.ID {
			return Item{}, fmt.Errorf("provider %q already exists", input.ID)
		}
	}

	nextConfig := s.config
	nextConfig.Providers = append(append([]config.ProviderConfig(nil), s.config.Providers...), input)
	if err := s.persist(nextConfig); err != nil {
		return Item{}, err
	}

	return redactProvider(input), nil
}

func (s *Service) Update(id string, input config.ProviderConfig) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizeProvider(input)

	index := -1
	for i, provider := range s.config.Providers {
		if provider.ID == id {
			index = i
			continue
		}
		if provider.ID == input.ID {
			return Item{}, fmt.Errorf("provider %q already exists", input.ID)
		}
	}
	if index < 0 {
		return Item{}, ErrNotFound{ProviderID: id}
	}

	nextProviders := append([]config.ProviderConfig(nil), s.config.Providers...)
	nextProviders[index] = input
	nextConfig := s.config
	nextConfig.Providers = nextProviders
	if err := s.persist(nextConfig); err != nil {
		return Item{}, err
	}

	return redactProvider(input), nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := -1
	for i, provider := range s.config.Providers {
		if provider.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return ErrNotFound{ProviderID: id}
	}

	nextProviders := append([]config.ProviderConfig(nil), s.config.Providers[:index]...)
	nextProviders = append(nextProviders, s.config.Providers[index+1:]...)
	nextConfig := s.config
	nextConfig.Providers = nextProviders
	return s.persist(nextConfig)
}

func (s *Service) persist(nextConfig config.Config) error {
	if err := config.Validate(nextConfig); err != nil {
		return err
	}
	if err := s.runtime.ReplaceProviders(nextConfig.Providers); err != nil {
		return err
	}
	if err := config.SavePath(s.configPath, nextConfig); err != nil {
		_ = s.runtime.ReplaceProviders(s.config.Providers)
		return err
	}

	s.config = nextConfig
	return nil
}

type ErrNotFound struct {
	ProviderID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("provider %q not found", e.ProviderID)
}

func normalizeProvider(provider config.ProviderConfig) config.ProviderConfig {
	provider.ID = strings.TrimSpace(provider.ID)
	provider.Type = strings.TrimSpace(provider.Type)
	provider.Name = strings.TrimSpace(provider.Name)
	provider.APIKey = strings.TrimSpace(provider.APIKey)
	provider.AccessToken = strings.TrimSpace(provider.AccessToken)
	provider.RefreshToken = strings.TrimSpace(provider.RefreshToken)
	return provider
}

func redactProvider(provider config.ProviderConfig) Item {
	problems := providerProblems(provider)
	status := "ready"
	if len(problems) > 0 {
		status = "misconfigured"
	}

	return Item{
		ID:              provider.ID,
		Type:            provider.Type,
		Name:            provider.Name,
		HasAPIKey:       strings.TrimSpace(provider.APIKey) != "",
		HasAccessToken:  strings.TrimSpace(provider.AccessToken) != "",
		HasRefreshToken: strings.TrimSpace(provider.RefreshToken) != "",
		Status:          status,
		Problems:        problems,
	}
}

func providerProblems(provider config.ProviderConfig) []string {
	problems := make([]string, 0, 2)

	switch provider.Type {
	case "codex":
		if strings.TrimSpace(provider.AccessToken) == "" && strings.TrimSpace(provider.APIKey) == "" {
			problems = append(problems, "missing access_token or api_key")
		}
	case "openai":
		if strings.TrimSpace(provider.APIKey) == "" && strings.TrimSpace(provider.AccessToken) == "" {
			problems = append(problems, "missing api_key or access_token")
		}
	default:
		problems = append(problems, "unsupported provider type")
	}

	return problems
}

package connections

import (
	"fmt"
	"strings"
	"sync"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/rs/zerolog"
)

type Runtime interface {
	ReplaceConnections([]config.ConnectionConfig) error
}

type Service struct {
	mu         sync.RWMutex
	configPath string
	config     config.Config
	runtime    Runtime
	logger     *zerolog.Logger
}

func NewService(configPath string, cfg config.Config, runtime Runtime, logger *zerolog.Logger) *Service {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
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
	ProviderID      string   `json:"provider_id"`
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

	items := make([]Item, 0, len(s.config.Connections))
	for _, connection := range s.config.Connections {
		items = append(items, redactConnection(connection))
	}

	return items
}

func (s *Service) Get(id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, connection := range s.config.Connections {
		if connection.ID == id {
			return redactConnection(connection), true
		}
	}

	return Item{}, false
}

func (s *Service) Create(input config.ConnectionConfig) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizeConnection(input)
	for _, connection := range s.config.Connections {
		if connection.ID == input.ID {
			return Item{}, fmt.Errorf("connection %q already exists", input.ID)
		}
	}

	nextConfig := s.config
	nextConfig.Connections = append(append([]config.ConnectionConfig(nil), s.config.Connections...), input)
	if err := s.persist(nextConfig); err != nil {
		return Item{}, err
	}

	s.logger.Info().
		Str("connection_id", input.ID).
		Str("provider_id", input.ProviderID).
		Str("connection_name", input.Name).
		Msg("connection_create")

	return redactConnection(input), nil
}

func (s *Service) Update(id string, input config.ConnectionConfig) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizeConnection(input)

	index := -1
	var existing config.ConnectionConfig
	for i, connection := range s.config.Connections {
		if connection.ID == id {
			index = i
			existing = connection
			continue
		}
		if connection.ID == input.ID {
			return Item{}, fmt.Errorf("connection %q already exists", input.ID)
		}
	}
	if index < 0 {
		return Item{}, ErrNotFound{ConnectionID: id}
	}

	input = preserveExistingSecrets(existing, input)

	nextConnections := append([]config.ConnectionConfig(nil), s.config.Connections...)
	nextConnections[index] = input
	nextConfig := s.config
	nextConfig.Connections = nextConnections
	if err := s.persist(nextConfig); err != nil {
		return Item{}, err
	}

	s.logger.Info().
		Str("connection_id", input.ID).
		Str("provider_id", input.ProviderID).
		Str("connection_name", input.Name).
		Msg("connection_update")

	return redactConnection(input), nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.config.Connections) == 1 {
		return fmt.Errorf("at least one connection must remain configured")
	}

	index := -1
	var deleted config.ConnectionConfig
	for i, connection := range s.config.Connections {
		if connection.ID == id {
			index = i
			deleted = connection
			break
		}
	}
	if index < 0 {
		return ErrNotFound{ConnectionID: id}
	}

	nextConnections := append([]config.ConnectionConfig(nil), s.config.Connections[:index]...)
	nextConnections = append(nextConnections, s.config.Connections[index+1:]...)
	nextConfig := s.config
	nextConfig.Connections = nextConnections
	if err := s.persist(nextConfig); err != nil {
		return err
	}

	s.logger.Info().
		Str("connection_id", deleted.ID).
		Str("provider_id", deleted.ProviderID).
		Str("connection_name", deleted.Name).
		Msg("connection_delete")

	return nil
}

func (s *Service) persist(nextConfig config.Config) error {
	if err := config.Validate(nextConfig); err != nil {
		s.logger.Error().Err(err).Msg("persist_failed")
		return err
	}
	if err := s.runtime.ReplaceConnections(nextConfig.Connections); err != nil {
		s.logger.Error().Err(err).Msg("runtime_reload_failed")
		return err
	}
	if err := config.SavePath(s.configPath, nextConfig); err != nil {
		s.logger.Error().Err(err).Msg("persist_failed")
		if rollbackErr := s.runtime.ReplaceConnections(s.config.Connections); rollbackErr != nil {
			s.logger.Error().Err(rollbackErr).Msg("runtime_reload_rollback_failed")
		}
		return err
	}

	s.config = nextConfig
	return nil
}

type ErrNotFound struct {
	ConnectionID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("connection %q not found", e.ConnectionID)
}

func normalizeConnection(connection config.ConnectionConfig) config.ConnectionConfig {
	connection.ID = strings.TrimSpace(connection.ID)
	connection.ProviderID = strings.TrimSpace(connection.ProviderID)
	connection.Name = strings.TrimSpace(connection.Name)
	connection.APIKey = strings.TrimSpace(connection.APIKey)
	connection.AccessToken = strings.TrimSpace(connection.AccessToken)
	connection.RefreshToken = strings.TrimSpace(connection.RefreshToken)
	return connection
}

func redactConnection(connection config.ConnectionConfig) Item {
	problems := connectionProblems(connection)
	status := "ready"
	if len(problems) > 0 {
		status = "misconfigured"
	}

	return Item{
		ID:              connection.ID,
		ProviderID:      connection.ProviderID,
		Name:            connection.Name,
		HasAPIKey:       strings.TrimSpace(connection.APIKey) != "",
		HasAccessToken:  strings.TrimSpace(connection.AccessToken) != "",
		HasRefreshToken: strings.TrimSpace(connection.RefreshToken) != "",
		Status:          status,
		Problems:        problems,
	}
}

func preserveExistingSecrets(
	existing config.ConnectionConfig,
	next config.ConnectionConfig,
) config.ConnectionConfig {
	if next.APIKey == "" {
		next.APIKey = existing.APIKey
	}
	if next.AccessToken == "" {
		next.AccessToken = existing.AccessToken
	}
	if next.RefreshToken == "" {
		next.RefreshToken = existing.RefreshToken
	}

	return next
}

func connectionProblems(connection config.ConnectionConfig) []string {
	problems := make([]string, 0, 2)

	switch connection.ProviderID {
	case "cx":
		if strings.TrimSpace(connection.AccessToken) == "" && strings.TrimSpace(connection.APIKey) == "" {
			problems = append(problems, "missing access_token or api_key")
		}
	case "openai":
		if strings.TrimSpace(connection.APIKey) == "" && strings.TrimSpace(connection.AccessToken) == "" {
			problems = append(problems, "missing api_key or access_token")
		}
	default:
		problems = append(problems, "unsupported provider")
	}

	return problems
}

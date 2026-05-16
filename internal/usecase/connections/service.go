package connections

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/rs/zerolog"
)

type Runtime interface {
	ReplaceConnections([]config.ConnectionConfig) error
}

type ProviderRegistry interface {
	ValidateConnection(config.ConnectionConfig) []string
	GetUsage(context.Context, config.ConnectionConfig) (providerregistry.UsageInfo, error)
	GenerateOAuthURL(config.ConnectionConfig) (string, error)
	StartOAuth(config.ConnectionConfig) (providerregistry.OAuthSession, error)
	CompleteOAuth(config.ConnectionConfig, map[string]string, string) (providerregistry.OAuthResult, error)
}

type Service struct {
	mu           sync.RWMutex
	configPath   string
	config       config.Config
	pendingOAuth map[string]pendingOAuthConnection
	runtime      Runtime
	providers    ProviderRegistry
	logger       *zerolog.Logger
}

type pendingOAuthConnection struct {
	ProviderID string
	Pending    map[string]string
}

type OAuthStartResult struct {
	SessionID        string `json:"session_id"`
	AuthorizationURL string `json:"url"`
}

func NewService(configPath string, cfg config.Config, runtime Runtime, providers ProviderRegistry, logger *zerolog.Logger) *Service {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return &Service{
		configPath:   configPath,
		config:       cfg,
		pendingOAuth: make(map[string]pendingOAuthConnection),
		runtime:      runtime,
		providers:    providers,
		logger:       logger,
	}
}

type Item struct {
	ID              string   `json:"id"`
	ProviderID      string   `json:"provider_id"`
	Name            string   `json:"name"`
	APIKey          string   `json:"api_key,omitempty"`
	AccessToken     string   `json:"access_token,omitempty"`
	RefreshToken    string   `json:"refresh_token,omitempty"`
	TokenType       string   `json:"token_type,omitempty"`
	ExpiresIn       int      `json:"expires_in,omitempty"`
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
		items = append(items, s.redactConnection(connection))
	}

	return items
}

func (s *Service) Get(id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, connection := range s.config.Connections {
		if connection.ID == id {
			return s.redactConnection(connection), true
		}
	}

	return Item{}, false
}

func (s *Service) GetUsage(ctx context.Context, id string) (providerregistry.UsageInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, connection := range s.config.Connections {
		if connection.ID == id {
			return s.providers.GetUsage(ctx, connection)
		}
	}

	return providerregistry.UsageInfo{}, ErrNotFound{ConnectionID: id}
}

func (s *Service) Providers() ProviderRegistry {
	return s.providers
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

	return s.redactConnection(input), nil
}

func (s *Service) StartOAuth(providerID string) (OAuthStartResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return OAuthStartResult{}, fmt.Errorf("provider id is required")
	}

	session, err := s.providers.StartOAuth(config.ConnectionConfig{
		ProviderID: providerID,
	})
	if err != nil {
		return OAuthStartResult{}, err
	}

	sessionID, err := randomBase64URL(24)
	if err != nil {
		return OAuthStartResult{}, err
	}

	s.pendingOAuth[sessionID] = pendingOAuthConnection{
		ProviderID: providerID,
		Pending:    clonePending(session.Pending),
	}

	return OAuthStartResult{
		SessionID:        sessionID,
		AuthorizationURL: session.AuthorizationURL,
	}, nil
}

func (s *Service) CompleteOAuth(sessionID, callbackURL string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Item{}, fmt.Errorf("oauth session id is required")
	}

	pending, ok := s.pendingOAuth[sessionID]
	if !ok {
		return Item{}, fmt.Errorf("oauth session %q not found", sessionID)
	}
	connectionID := s.nextOAuthConnectionID(pending.ProviderID)

	result, err := s.providers.CompleteOAuth(
		config.ConnectionConfig{ID: connectionID, ProviderID: pending.ProviderID},
		clonePending(pending.Pending),
		strings.TrimSpace(callbackURL),
	)
	if err != nil {
		return Item{}, err
	}

	input := normalizeConnection(config.ConnectionConfig{
		ID:                   connectionID,
		ProviderID:           pending.ProviderID,
		Name:                 defaultString(strings.TrimSpace(result.Name), connectionID),
		AccessToken:          result.AccessToken,
		RefreshToken:         result.RefreshToken,
		TokenType:            result.TokenType,
		ExpiresIn:            result.ExpiresIn,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})

	nextConfig := s.config
	nextConfig.Connections = append(append([]config.ConnectionConfig(nil), s.config.Connections...), input)
	if err := s.persist(nextConfig); err != nil {
		return Item{}, err
	}

	delete(s.pendingOAuth, sessionID)

	s.logger.Info().
		Str("connection_id", input.ID).
		Str("provider_id", input.ProviderID).
		Str("connection_name", input.Name).
		Msg("connection_create_oauth")

	return s.redactConnection(input), nil
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

	return s.redactConnection(input), nil
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
	connection.TokenType = strings.TrimSpace(connection.TokenType)
	return connection
}

func (s *Service) redactConnection(connection config.ConnectionConfig) Item {
	problems := connectionProblems(s.providers, connection)
	status := "ready"
	if len(problems) > 0 {
		status = "misconfigured"
	}

	return Item{
		ID:              connection.ID,
		ProviderID:      connection.ProviderID,
		Name:            connection.Name,
		TokenType:       connection.TokenType,
		ExpiresIn:       connection.ExpiresIn,
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
	if next.TokenType == "" {
		next.TokenType = existing.TokenType
	}
	if next.ExpiresIn == 0 {
		next.ExpiresIn = existing.ExpiresIn
	}
	if next.AccessTokenExpiresAt == 0 {
		next.AccessTokenExpiresAt = existing.AccessTokenExpiresAt
	}

	return next
}

func connectionProblems(providers ProviderRegistry, connection config.ConnectionConfig) []string {
	if providers == nil {
		return nil
	}

	return providers.ValidateConnection(connection)
}

func clonePending(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (s *Service) nextOAuthConnectionID(providerID string) string {
	prefix := providerID
	maxSuffix := 0
	for _, connection := range s.config.Connections {
		if connection.ProviderID != providerID {
			continue
		}

		nextPrefix, suffix, ok := splitNumericConnectionID(connection.ID)
		if !ok {
			continue
		}

		if prefix == providerID {
			prefix = nextPrefix
		}
		if nextPrefix == prefix && suffix > maxSuffix {
			maxSuffix = suffix
		}
	}

	return fmt.Sprintf("%s-%d", prefix, maxSuffix+1)
}

func randomBase64URL(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func splitNumericConnectionID(connectionID string) (string, int, bool) {
	index := strings.LastIndex(connectionID, "-")
	if index <= 0 || index == len(connectionID)-1 {
		return "", 0, false
	}

	suffix, err := strconv.Atoi(connectionID[index+1:])
	if err != nil {
		return "", 0, false
	}

	return connectionID[:index], suffix, true
}

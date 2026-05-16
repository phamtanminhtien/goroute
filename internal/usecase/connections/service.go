package connections

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/rs/zerolog"
)

type Runtime interface {
	ReloadConnections() error
}

type ProviderRegistry interface {
	ValidateConnection(connection.Record) []string
	GetUsage(context.Context, connection.Record) (providerregistry.UsageInfo, error)
	GenerateOAuthURL(connection.Record) (string, error)
	StartOAuth(connection.Record) (providerregistry.OAuthSession, error)
	CompleteOAuth(connection.Record, map[string]string, string) (providerregistry.OAuthResult, error)
}

type Service struct {
	mu           sync.Mutex
	pendingOAuth map[string]pendingOAuthConnection
	repo         Repository
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

func NewService(repo Repository, runtime Runtime, providers ProviderRegistry, logger *zerolog.Logger) *Service {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return &Service{
		pendingOAuth: make(map[string]pendingOAuthConnection),
		repo:         repo,
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
	records, err := s.repo.ListConnections()
	if err != nil {
		s.logger.Error().Err(err).Msg("connection_list_failed")
		return nil
	}

	items := make([]Item, 0, len(records))
	for _, record := range records {
		items = append(items, s.redactConnection(record))
	}

	return items
}

func (s *Service) Get(id string) (Item, bool) {
	record, ok, err := s.repo.GetConnection(id)
	if err != nil {
		s.logger.Error().Err(err).Str("connection_id", id).Msg("connection_get_failed")
		return Item{}, false
	}
	if !ok {
		return Item{}, false
	}

	return s.redactConnection(record), true
}

func (s *Service) GetUsage(ctx context.Context, id string) (providerregistry.UsageInfo, error) {
	record, ok, err := s.repo.GetConnection(id)
	if err != nil {
		return providerregistry.UsageInfo{}, err
	}
	if !ok {
		return providerregistry.UsageInfo{}, ErrNotFound{ConnectionID: id}
	}

	return s.providers.GetUsage(ctx, record)
}

func (s *Service) Providers() ProviderRegistry {
	return s.providers
}

func (s *Service) Create(input connection.Record) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizeConnection(input)
	existingConnections, err := s.repo.ListConnections()
	if err != nil {
		return Item{}, err
	}
	for _, existing := range existingConnections {
		if existing.ID == input.ID {
			return Item{}, ErrConflict{ConnectionID: input.ID}
		}
	}

	if err := s.persistMutation(func() error {
		return s.repo.CreateConnection(input)
	}); err != nil {
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

	session, err := s.providers.StartOAuth(connection.Record{
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
	connectionID, err := s.nextOAuthConnectionID(pending.ProviderID)
	if err != nil {
		return Item{}, err
	}

	result, err := s.providers.CompleteOAuth(
		connection.Record{ID: connectionID, ProviderID: pending.ProviderID},
		clonePending(pending.Pending),
		strings.TrimSpace(callbackURL),
	)
	if err != nil {
		return Item{}, err
	}

	input := normalizeConnection(connection.Record{
		ID:                   connectionID,
		ProviderID:           pending.ProviderID,
		Name:                 defaultString(strings.TrimSpace(result.Name), connectionID),
		AccessToken:          result.AccessToken,
		RefreshToken:         result.RefreshToken,
		TokenType:            result.TokenType,
		ExpiresIn:            result.ExpiresIn,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})

	if err := s.persistMutation(func() error {
		return s.repo.CreateConnection(input)
	}); err != nil {
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

func (s *Service) Update(id string, input connection.Record) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizeConnection(input)

	existing, ok, err := s.repo.GetConnection(id)
	if err != nil {
		return Item{}, err
	}
	if !ok {
		return Item{}, ErrNotFound{ConnectionID: id}
	}
	existingConnections, err := s.repo.ListConnections()
	if err != nil {
		return Item{}, err
	}
	for _, current := range existingConnections {
		if current.ID == input.ID && current.ID != id {
			return Item{}, ErrConflict{ConnectionID: input.ID}
		}
	}

	input = preserveExistingSecrets(existing, input)

	if err := s.persistMutation(func() error {
		return s.repo.UpdateConnection(id, input)
	}); err != nil {
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

	deleted, ok, err := s.repo.GetConnection(id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound{ConnectionID: id}
	}

	if err := s.persistMutation(func() error {
		return s.repo.DeleteConnection(id)
	}); err != nil {
		return err
	}

	s.logger.Info().
		Str("connection_id", deleted.ID).
		Str("provider_id", deleted.ProviderID).
		Str("connection_name", deleted.Name).
		Msg("connection_delete")

	return nil
}

func (s *Service) persistMutation(write func() error) error {
	if err := write(); err != nil {
		s.logger.Error().Err(err).Msg("persist_failed")
		return err
	}
	if err := s.runtime.ReloadConnections(); err != nil {
		s.logger.Error().Err(err).Msg("runtime_reload_failed")
		return err
	}

	return nil
}

func normalizeConnection(connection connection.Record) connection.Record {
	connection.ID = strings.TrimSpace(connection.ID)
	connection.ProviderID = strings.TrimSpace(connection.ProviderID)
	connection.Name = strings.TrimSpace(connection.Name)
	connection.APIKey = strings.TrimSpace(connection.APIKey)
	connection.AccessToken = strings.TrimSpace(connection.AccessToken)
	connection.RefreshToken = strings.TrimSpace(connection.RefreshToken)
	connection.TokenType = strings.TrimSpace(connection.TokenType)
	return connection
}

func (s *Service) redactConnection(connection connection.Record) Item {
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

func preserveExistingSecrets(existing connection.Record, next connection.Record) connection.Record {
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

func connectionProblems(providers ProviderRegistry, connection connection.Record) []string {
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

func (s *Service) nextOAuthConnectionID(providerID string) (string, error) {
	connections, err := s.repo.ListConnections()
	if err != nil {
		return "", err
	}

	prefix := providerID
	maxSuffix := 0
	for _, connection := range connections {
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

	return fmt.Sprintf("%s-%d", prefix, maxSuffix+1), nil
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

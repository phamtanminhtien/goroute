package config

import "sync"

type LLMLoggingState struct {
	FlowEnabled       bool
	ThirdPartyEnabled bool
}

type RTKState struct {
	Enabled bool
}

type OpenAICompatibleAuthState struct {
	Enabled bool
}

type ProviderRuntimeState struct {
	DialTimeoutMs           int
	TLSHandshakeTimeoutMs   int
	ResponseHeaderTimeoutMs int
	TimeoutRetryCount       int
	RetryableCooldownMs     int
}

type SettingsManager struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

func NewSettingsManager(path string, cfg Config) *SettingsManager {
	return &SettingsManager{
		path: path,
		cfg:  ApplyDefaults(cfg),
	}
}

func (m *SettingsManager) Snapshot() Config {
	if m == nil {
		return ApplyDefaults(Config{})
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cfg
}

func (m *SettingsManager) LLMLogging() LLMLoggingState {
	cfg := m.Snapshot()
	return LLMLoggingState{
		FlowEnabled:       cfg.LLMLogging.Flow,
		ThirdPartyEnabled: cfg.LLMLogging.ThirdParty,
	}
}

func (m *SettingsManager) RTK() RTKState {
	cfg := m.Snapshot()
	return RTKState{
		Enabled: cfg.RTK.Enabled,
	}
}

func (m *SettingsManager) OpenAICompatibleAuth() OpenAICompatibleAuthState {
	cfg := m.Snapshot()
	return OpenAICompatibleAuthState{
		Enabled: cfg.OpenAICompatibleAuth.Enabled,
	}
}

func (m *SettingsManager) ProviderRuntime(providerID string) ProviderRuntimeState {
	settings := EffectiveProviderRuntimeSettings(m.Snapshot(), providerID)
	return ProviderRuntimeState{
		DialTimeoutMs:           settings.DialTimeoutMs,
		TLSHandshakeTimeoutMs:   settings.TLSHandshakeTimeoutMs,
		ResponseHeaderTimeoutMs: settings.ResponseHeaderTimeoutMs,
		TimeoutRetryCount:       settings.TimeoutRetryCount,
		RetryableCooldownMs:     settings.RetryableCooldownMs,
	}
}

func (m *SettingsManager) UpdateLLMLogging(state LLMLoggingState) (Config, error) {
	if m == nil {
		return Config{}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.cfg
	next.LLMLogging = LLMLoggingConfig{
		Flow:       state.FlowEnabled,
		ThirdParty: state.ThirdPartyEnabled,
		present:    true,
	}
	if err := SavePath(m.path, next); err != nil {
		return Config{}, err
	}

	m.cfg = ApplyDefaults(next)
	return m.cfg, nil
}

func (m *SettingsManager) UpdateRTK(state RTKState) (Config, error) {
	if m == nil {
		return Config{}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.cfg
	next.RTK = RTKConfig{
		Enabled: state.Enabled,
		present: true,
	}
	if err := SavePath(m.path, next); err != nil {
		return Config{}, err
	}

	m.cfg = ApplyDefaults(next)
	return m.cfg, nil
}

func (m *SettingsManager) UpdateSettings(llmLogging LLMLoggingState, rtk RTKState, openAICompatibleAuth OpenAICompatibleAuthState) (Config, error) {
	if m == nil {
		return Config{}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.cfg
	next.LLMLogging = LLMLoggingConfig{
		Flow:       llmLogging.FlowEnabled,
		ThirdParty: llmLogging.ThirdPartyEnabled,
		present:    true,
	}
	next.RTK = RTKConfig{
		Enabled: rtk.Enabled,
		present: true,
	}
	next.OpenAICompatibleAuth = OpenAICompatibleAuthConfig{
		Enabled: openAICompatibleAuth.Enabled,
	}
	if err := SavePath(m.path, next); err != nil {
		return Config{}, err
	}

	m.cfg = ApplyDefaults(next)
	return m.cfg, nil
}

func (m *SettingsManager) UpdateProviderRuntime(providerID string, state ProviderRuntimeState) (Config, error) {
	if m == nil {
		return Config{}, nil
	}

	settings := ProviderRuntimeSettings{
		DialTimeoutMs:           state.DialTimeoutMs,
		TLSHandshakeTimeoutMs:   state.TLSHandshakeTimeoutMs,
		ResponseHeaderTimeoutMs: state.ResponseHeaderTimeoutMs,
		TimeoutRetryCount:       state.TimeoutRetryCount,
		RetryableCooldownMs:     state.RetryableCooldownMs,
	}
	if err := ValidateProviderRuntimeSettings(settings); err != nil {
		return Config{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.cfg
	next.ProviderRuntime = cloneProviderRuntimeConfig(next.ProviderRuntime)
	if next.ProviderRuntime == nil {
		next.ProviderRuntime = ProviderRuntimeConfig{}
	}
	next.ProviderRuntime[providerID] = settings
	if err := SavePath(m.path, next); err != nil {
		return Config{}, err
	}

	m.cfg = ApplyDefaults(next)
	return m.cfg, nil
}

func (m *SettingsManager) ResetProviderRuntime(providerID string) (Config, error) {
	if m == nil {
		return Config{}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.cfg
	next.ProviderRuntime = cloneProviderRuntimeConfig(next.ProviderRuntime)
	if next.ProviderRuntime != nil {
		delete(next.ProviderRuntime, providerID)
		if len(next.ProviderRuntime) == 0 {
			next.ProviderRuntime = nil
		}
	}
	if err := SavePath(m.path, next); err != nil {
		return Config{}, err
	}

	m.cfg = ApplyDefaults(next)
	return m.cfg, nil
}

func cloneProviderRuntimeConfig(source ProviderRuntimeConfig) ProviderRuntimeConfig {
	if len(source) == 0 {
		return nil
	}

	cloned := make(ProviderRuntimeConfig, len(source))
	for providerID, settings := range source {
		cloned[providerID] = settings
	}
	return cloned
}

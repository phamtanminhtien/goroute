package config

import "sync"

type LLMLoggingState struct {
	FlowEnabled       bool
	ThirdPartyEnabled bool
}

type RTKState struct {
	Enabled bool
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

func (m *SettingsManager) UpdateSettings(llmLogging LLMLoggingState, rtk RTKState) (Config, error) {
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
	if err := SavePath(m.path, next); err != nil {
		return Config{}, err
	}

	m.cfg = ApplyDefaults(next)
	return m.cfg, nil
}

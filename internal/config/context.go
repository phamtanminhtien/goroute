package config

import "context"

type contextKey string

const settingsManagerContextKey contextKey = "settings_manager"

func WithSettingsManager(ctx context.Context, manager *SettingsManager) context.Context {
	if manager == nil {
		return ctx
	}

	return context.WithValue(ctx, settingsManagerContextKey, manager)
}

func RTKEnabledFromContext(ctx context.Context) bool {
	manager, _ := ctx.Value(settingsManagerContextKey).(*SettingsManager)
	if manager == nil {
		return false
	}

	return manager.RTK().Enabled
}

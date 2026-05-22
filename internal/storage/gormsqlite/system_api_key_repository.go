package gormsqlite

import (
	"errors"
	"fmt"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/systemapikey"
	"gorm.io/gorm"
)

func (r *Repository) ListSystemAPIKeys() ([]systemapikey.Record, error) {
	var records []systemapikey.Record
	if err := r.db.Order("created_at DESC, id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list system api keys: %w", err)
	}

	now := time.Now().UTC()
	for i := range records {
		usage, err := r.SystemAPIKeyUsage(records[i], now)
		if err != nil {
			return nil, err
		}
		records[i].Usage = &usage
	}

	return records, nil
}

func (r *Repository) CreateSystemAPIKey(record systemapikey.Record) error {
	if err := r.db.Select("*").Create(&record).Error; err != nil {
		return normalizeWriteError(err, record.ID, "create")
	}

	return nil
}

func (r *Repository) GetSystemAPIKey(id string) (systemapikey.Record, bool, error) {
	var record systemapikey.Record
	if err := r.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return systemapikey.Record{}, false, nil
		}
		return systemapikey.Record{}, false, fmt.Errorf("get system api key %q: %w", id, err)
	}
	usage, err := r.SystemAPIKeyUsage(record, time.Now().UTC())
	if err != nil {
		return systemapikey.Record{}, false, err
	}
	record.Usage = &usage

	return record, true, nil
}

func (r *Repository) UpdateSystemAPIKey(record systemapikey.Record) error {
	updates := map[string]any{
		"name":                      record.Name,
		"enabled":                   record.Enabled,
		"requests_per_minute_limit": record.RequestsPerMinuteLimit,
		"daily_token_limit":         record.DailyTokenLimit,
		"monthly_token_limit":       record.MonthlyTokenLimit,
	}
	result := r.db.Model(&systemapikey.Record{}).Where("id = ?", record.ID).Updates(updates)
	if result.Error != nil {
		return normalizeWriteError(result.Error, record.ID, "update")
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *Repository) DeleteSystemAPIKey(id string) error {
	result := r.db.Delete(&systemapikey.Record{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete system api key %q: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *Repository) HasSystemAPIKeys() (bool, error) {
	var count int64
	if err := r.db.Model(&systemapikey.Record{}).Count(&count).Error; err != nil {
		return false, fmt.Errorf("count system api keys: %w", err)
	}

	return count > 0, nil
}

func (r *Repository) AuthenticateSystemAPIKey(key string) (systemapikey.Record, bool, error) {
	var record systemapikey.Record
	if err := r.db.First(&record, "key = ? AND enabled = ?", key, true).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return systemapikey.Record{}, false, nil
		}
		return systemapikey.Record{}, false, fmt.Errorf("authenticate system api key: %w", err)
	}

	now := time.Now().Unix()
	if err := r.db.Model(&systemapikey.Record{}).Where("id = ?", record.ID).Update("last_used_at", now).Error; err != nil {
		return systemapikey.Record{}, false, fmt.Errorf("update system api key last used: %w", err)
	}
	record.LastUsedAt = now

	return record, true, nil
}

func (r *Repository) CreateSystemAPIKeyRequestEvent(record systemapikey.RequestEvent) error {
	if err := r.db.Create(&record).Error; err != nil {
		return fmt.Errorf("create system api key request event for key %q: %w", record.SystemAPIKeyID, err)
	}

	return nil
}

func (r *Repository) CountSystemAPIKeyRequestsSince(id string, since time.Time) (int, error) {
	var count int64
	if err := r.db.Model(&systemapikey.RequestEvent{}).
		Where("system_api_key_id = ? AND created_at >= ?", id, since.UTC().UnixMilli()).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count system api key request events for key %q: %w", id, err)
	}

	return int(count), nil
}

func (r *Repository) SystemAPIKeyTokenUsage(id string, from time.Time, to time.Time) (int, error) {
	var total int64
	if err := r.db.Model(&airequestlog.RunRecord{}).
		Where("system_api_key_id = ? AND created_at >= ? AND created_at < ?", id, from.UTC().UnixMilli(), to.UTC().UnixMilli()).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("sum system api key token usage for key %q: %w", id, err)
	}

	return int(total), nil
}

func (r *Repository) SystemAPIKeyUsage(record systemapikey.Record, now time.Time) (systemapikey.UsageSummary, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	minuteStart := now.Add(-time.Minute)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextDay := dayStart.AddDate(0, 0, 1)
	nextMonth := monthStart.AddDate(0, 1, 0)

	currentMinute, err := r.CountSystemAPIKeyRequestsSince(record.ID, minuteStart)
	if err != nil {
		return systemapikey.UsageSummary{}, err
	}
	dailyTokens, err := r.SystemAPIKeyTokenUsage(record.ID, dayStart, nextDay)
	if err != nil {
		return systemapikey.UsageSummary{}, err
	}
	monthlyTokens, err := r.SystemAPIKeyTokenUsage(record.ID, monthStart, nextMonth)
	if err != nil {
		return systemapikey.UsageSummary{}, err
	}

	return systemapikey.UsageSummary{
		CurrentMinuteRequests: currentMinute,
		DailyTokens:           tokenUsage(dailyTokens, record.DailyTokenLimit),
		MonthlyTokens:         tokenUsage(monthlyTokens, record.MonthlyTokenLimit),
		RateLimitReached:      record.RequestsPerMinuteLimit != nil && currentMinute >= *record.RequestsPerMinuteLimit,
		DailyLimitReached:     record.DailyTokenLimit != nil && dailyTokens >= *record.DailyTokenLimit,
		MonthlyLimitReached:   record.MonthlyTokenLimit != nil && monthlyTokens >= *record.MonthlyTokenLimit,
	}, nil
}

func tokenUsage(used int, limit *int) systemapikey.TokenUsage {
	var remaining *int
	if limit != nil {
		value := *limit - used
		if value < 0 {
			value = 0
		}
		remaining = &value
	}

	return systemapikey.TokenUsage{
		Used:      used,
		Limit:     limit,
		Remaining: remaining,
	}
}

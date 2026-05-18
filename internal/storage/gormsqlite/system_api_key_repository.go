package gormsqlite

import (
	"errors"
	"fmt"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/systemapikey"
	"gorm.io/gorm"
)

func (r *Repository) ListSystemAPIKeys() ([]systemapikey.Record, error) {
	var records []systemapikey.Record
	if err := r.db.Order("created_at DESC, id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list system api keys: %w", err)
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

	return record, true, nil
}

func (r *Repository) UpdateSystemAPIKey(record systemapikey.Record) error {
	updates := map[string]any{
		"name":    record.Name,
		"enabled": record.Enabled,
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

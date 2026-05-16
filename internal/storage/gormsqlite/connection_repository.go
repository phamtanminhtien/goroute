package gormsqlite

import (
	"errors"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
	"gorm.io/gorm"
)

func (r *Repository) ListConnections() ([]connection.Record, error) {
	var records []connection.Record
	if err := r.db.Order("provider_id ASC, id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	return records, nil
}

func (r *Repository) GetConnection(id string) (connection.Record, bool, error) {
	var record connection.Record
	if err := r.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return connection.Record{}, false, nil
		}
		return connection.Record{}, false, fmt.Errorf("get connection %q: %w", id, err)
	}

	return record, true, nil
}

func (r *Repository) CreateConnection(record connection.Record) error {
	if err := r.db.Create(&record).Error; err != nil {
		return normalizeWriteError(err, record.ID, "create")
	}

	return nil
}

func (r *Repository) UpdateConnection(previousID string, record connection.Record) error {
	updates := map[string]any{
		"id":                      record.ID,
		"provider_id":             record.ProviderID,
		"api_key":                 record.APIKey,
		"access_token":            record.AccessToken,
		"refresh_token":           record.RefreshToken,
		"token_type":              record.TokenType,
		"expires_in":              record.ExpiresIn,
		"access_token_expires_at": record.AccessTokenExpiresAt,
		"name":                    record.Name,
	}

	result := r.db.Model(&connection.Record{}).Where("id = ?", previousID).Updates(updates)
	if result.Error != nil {
		return normalizeWriteError(result.Error, previousID, "update")
	}
	if result.RowsAffected == 0 {
		return connectionsusecase.ErrNotFound{ConnectionID: previousID}
	}

	return nil
}

func (r *Repository) DeleteConnection(id string) error {
	result := r.db.Delete(&connection.Record{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete connection %q: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return connectionsusecase.ErrNotFound{ConnectionID: id}
	}

	return nil
}

func (r *Repository) ReplaceConnections(records []connection.Record) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&connection.Record{}).Error; err != nil {
			return fmt.Errorf("clear connections: %w", err)
		}

		if len(records) == 0 {
			return nil
		}

		if err := tx.Create(&records).Error; err != nil {
			return normalizeWriteError(err, "", "replace")
		}

		return nil
	})
}

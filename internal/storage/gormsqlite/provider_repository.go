package gormsqlite

import (
	"errors"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"gorm.io/gorm"
)

func (r *Repository) ListProviders() ([]provider.Record, error) {
	var records []provider.Record
	if err := r.db.Order("id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	return records, nil
}

func (r *Repository) GetProvider(id string) (provider.Record, bool, error) {
	var record provider.Record
	if err := r.db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return provider.Record{}, false, nil
		}
		return provider.Record{}, false, fmt.Errorf("get provider %q: %w", id, err)
	}

	return record, true, nil
}

func (r *Repository) CreateProviderWithConnection(providerRecord provider.Record, connectionRecord connection.Record, modelRecord provider.ModelRecord) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&providerRecord).Error; err != nil {
			return normalizeProviderWriteError(err, providerRecord.ID, "create")
		}
		if err := tx.Select("*").Create(&connectionRecord).Error; err != nil {
			return normalizeWriteError(err, connectionRecord.ID, "create")
		}
		if err := tx.Create(&modelRecord).Error; err != nil {
			return normalizeWriteError(err, modelRecord.ID, "create")
		}

		return nil
	})
}

func (r *Repository) UpdateProviderWithConnection(id string, providerRecord provider.Record, connectionRecord connection.Record, modelRecord provider.ModelRecord) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&provider.Record{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"name":          providerRecord.Name,
				"auth_type":     providerRecord.AuthType,
				"category":      providerRecord.Category,
				"adapter_type":  providerRecord.AdapterType,
				"base_url":      providerRecord.BaseURL,
				"default_model": providerRecord.DefaultModel,
			})
		if result.Error != nil {
			return normalizeProviderWriteError(result.Error, id, "update")
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("provider %q not found", id)
		}

		result = tx.Model(&connection.Record{}).
			Where("id = ? AND provider_id = ?", id, id).
			Updates(map[string]any{
				"api_key":       connectionRecord.APIKey,
				"access_token":  connectionRecord.AccessToken,
				"refresh_token": connectionRecord.RefreshToken,
				"enabled":       connectionRecord.Enabled,
				"name":          connectionRecord.Name,
			})
		if result.Error != nil {
			return normalizeWriteError(result.Error, id, "update")
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("managed connection %q not found", id)
		}

		if err := tx.Where("provider_id = ? AND id = ?", modelRecord.ProviderID, modelRecord.ID).
			FirstOrCreate(&modelRecord).Error; err != nil {
			return normalizeWriteError(err, modelRecord.ID, "create")
		}

		return nil
	})
}

func (r *Repository) DeleteProviderWithManagedConnection(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var targetCount int64
		if err := tx.Model(&modelcombo.Target{}).Where("provider_id = ?", id).Count(&targetCount).Error; err != nil {
			return fmt.Errorf("check provider %q combo references: %w", id, err)
		}
		if targetCount > 0 {
			return fmt.Errorf("provider %q is used by model combos", id)
		}

		if err := tx.Where("provider_id = ?", id).Delete(&provider.ModelRecord{}).Error; err != nil {
			return fmt.Errorf("delete provider %q models: %w", id, err)
		}
		if err := tx.Delete(&connection.Record{}, "id = ? AND provider_id = ?", id, id).Error; err != nil {
			return fmt.Errorf("delete provider %q managed connection: %w", id, err)
		}
		result := tx.Delete(&provider.Record{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("delete provider %q: %w", id, result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("provider %q not found", id)
		}

		return nil
	})
}

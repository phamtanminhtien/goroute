package gormsqlite

import (
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

func (r *Repository) ListProviderModels() ([]provider.ModelRecord, error) {
	var records []provider.ModelRecord
	if err := r.db.Order("provider_id ASC, id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list provider models: %w", err)
	}

	return records, nil
}

func (r *Repository) CreateProviderModel(record provider.ModelRecord) error {
	if err := r.db.Create(&record).Error; err != nil {
		return normalizeWriteError(err, record.ID, "create")
	}

	return nil
}

func (r *Repository) UpdateProviderModel(providerID string, modelID string, record provider.ModelRecord) error {
	result := r.db.Model(&provider.ModelRecord{}).
		Where("provider_id = ? AND id = ?", providerID, modelID).
		Updates(map[string]any{
			"id":                           record.ID,
			"name":                         record.Name,
			"description":                  record.Description,
			"input_price_per_million_usd":  record.InputPricePerMillionUSD,
			"output_price_per_million_usd": record.OutputPricePerMillionUSD,
		})
	if result.Error != nil {
		return normalizeWriteError(result.Error, record.ID, "update")
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("model %q not found", modelID)
	}

	return nil
}

func (r *Repository) DeleteProviderModel(providerID string, modelID string) error {
	result := r.db.Where("provider_id = ? AND id = ?", providerID, modelID).
		Delete(&provider.ModelRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete provider model %q: %w", modelID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("model %q not found", modelID)
	}

	return nil
}

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

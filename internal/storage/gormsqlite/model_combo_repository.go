package gormsqlite

import (
	"errors"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"gorm.io/gorm"
)

func (r *Repository) ListModelCombos() ([]modelcombo.Combo, error) {
	var combos []modelcombo.Combo
	if err := r.db.Preload("Targets", func(db *gorm.DB) *gorm.DB {
		return db.Order("priority ASC, id ASC")
	}).Order("alias ASC").Find(&combos).Error; err != nil {
		return nil, fmt.Errorf("list model combos: %w", err)
	}

	return combos, nil
}

func (r *Repository) GetModelCombo(alias string) (modelcombo.Combo, bool, error) {
	var combo modelcombo.Combo
	err := r.db.Preload("Targets", func(db *gorm.DB) *gorm.DB {
		return db.Order("priority ASC, id ASC")
	}).First(&combo, "alias = ?", alias).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return modelcombo.Combo{}, false, nil
	}
	if err != nil {
		return modelcombo.Combo{}, false, fmt.Errorf("get model combo %q: %w", alias, err)
	}

	return combo, true, nil
}

func (r *Repository) CreateModelCombo(combo modelcombo.Combo) error {
	if err := r.db.Select("*").Create(&combo).Error; err != nil {
		return normalizeModelComboWriteError(err, combo.Alias, "create")
	}

	return nil
}

func (r *Repository) UpdateModelCombo(alias string, combo modelcombo.Combo) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&modelcombo.Combo{}).Where("alias = ?", alias).Updates(map[string]any{
			"alias":       combo.Alias,
			"name":        combo.Name,
			"description": combo.Description,
			"enabled":     combo.Enabled,
		})
		if result.Error != nil {
			return normalizeModelComboWriteError(result.Error, alias, "update")
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		targetAlias := alias
		if combo.Alias != alias {
			targetAlias = combo.Alias
		}
		if err := tx.Where("combo_alias = ?", targetAlias).Delete(&modelcombo.Target{}).Error; err != nil {
			return fmt.Errorf("replace model combo targets %q: %w", alias, err)
		}
		for index := range combo.Targets {
			combo.Targets[index].ID = 0
			combo.Targets[index].ComboAlias = combo.Alias
		}
		if len(combo.Targets) > 0 {
			if err := tx.Select("*").Create(&combo.Targets).Error; err != nil {
				return normalizeModelComboWriteError(err, combo.Alias, "update")
			}
		}

		return nil
	})
}

func (r *Repository) DeleteModelCombo(alias string) error {
	result := r.db.Delete(&modelcombo.Combo{}, "alias = ?", alias)
	if result.Error != nil {
		return fmt.Errorf("delete model combo %q: %w", alias, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func normalizeModelComboWriteError(err error, alias, action string) error {
	if alias == "" {
		return fmt.Errorf("%s model combo: %w", action, err)
	}

	return fmt.Errorf("%s model combo %q: %w", action, alias, err)
}

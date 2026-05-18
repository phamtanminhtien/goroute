package gormsqlite

import (
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

func (r *Repository) Migrate() error {
	hadConnectionEnabled := r.db.Migrator().HasColumn(&connection.Record{}, "enabled")
	hadModelComboEnabled := r.db.Migrator().HasColumn(&modelcombo.Combo{}, "enabled")

	if err := r.db.AutoMigrate(
		&connection.Record{},
		&provider.ModelRecord{},
		&modelcombo.Combo{},
		&modelcombo.Target{},
		&airequestlog.RunRecord{},
		&airequestlog.FlowRecord{},
		&airequestlog.ThirdPartyRequestLogRecord{},
		&airequestlog.RTKRecord{},
	); err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}
	if !hadConnectionEnabled {
		if err := r.db.Model(&connection.Record{}).Where("enabled = ?", false).Update("enabled", true).Error; err != nil {
			return fmt.Errorf("backfill connection enabled defaults: %w", err)
		}
	}
	if !hadModelComboEnabled {
		if err := r.db.Model(&modelcombo.Combo{}).Where("enabled = ?", false).Update("enabled", true).Error; err != nil {
			return fmt.Errorf("backfill model combo enabled defaults: %w", err)
		}
	}

	return nil
}

package gormsqlite

import (
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

func (r *Repository) Migrate() error {
	if err := r.db.AutoMigrate(
		&connection.Record{},
		&provider.ModelRecord{},
		&airequestlog.RunRecord{},
		&airequestlog.FlowRecord{},
		&airequestlog.ThirdPartyRequestLogRecord{},
		&airequestlog.RTKRecord{},
	); err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}

	return nil
}

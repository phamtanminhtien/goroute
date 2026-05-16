package gormsqlite

import (
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
)

func (r *Repository) Migrate() error {
	if err := r.db.AutoMigrate(
		&connection.Record{},
		&airequestlog.RunRecord{},
		&airequestlog.FlowRecord{},
		&airequestlog.ThirdPartyRequestLogRecord{},
	); err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}

	return nil
}

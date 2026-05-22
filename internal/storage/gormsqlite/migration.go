package gormsqlite

import (
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/domain/systemapikey"
)

func (r *Repository) Migrate() error {
	if err := r.db.AutoMigrate(
		&provider.Record{},
		&connection.Record{},
		&provider.ModelRecord{},
		&modelcombo.Combo{},
		&modelcombo.Target{},
		&systemapikey.Record{},
		&systemapikey.RequestEvent{},
		&airequestlog.RunRecord{},
		&airequestlog.FlowRecord{},
		&airequestlog.ThirdPartyRequestLogRecord{},
		&airequestlog.RTKRecord{},
	); err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}

	for _, column := range []string{"ResponseBody", "TranslatedResponseBody"} {
		if err := r.db.Migrator().AlterColumn(&airequestlog.FlowRecord{}, column); err != nil {
			return fmt.Errorf("relax ai request flow column %s nullability: %w", column, err)
		}
	}
	if err := r.db.Migrator().AlterColumn(&airequestlog.ThirdPartyRequestLogRecord{}, "ResponseBody"); err != nil {
		return fmt.Errorf("relax third party request log response body nullability: %w", err)
	}

	return nil
}

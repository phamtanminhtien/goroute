package gormsqlite

import (
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
)

func (r *Repository) CreateAIRequestRun(record airequestlog.RunRecord) error {
	if err := r.db.Create(&record).Error; err != nil {
		return fmt.Errorf("create ai request run %q: %w", record.RequestID, err)
	}

	return nil
}

func (r *Repository) CreateAIRequestFlow(record airequestlog.FlowRecord) error {
	if err := r.db.Create(&record).Error; err != nil {
		return fmt.Errorf("create ai request flow %q: %w", record.RequestID, err)
	}

	return nil
}

func (r *Repository) CreateThirdPartyRequestLog(record airequestlog.ThirdPartyRequestLogRecord) error {
	if err := r.db.Create(&record).Error; err != nil {
		return fmt.Errorf("create third party request log for %q: %w", record.FlowRequestID, err)
	}

	return nil
}

func (r *Repository) ListAIRequestRuns() ([]airequestlog.RunRecord, error) {
	var records []airequestlog.RunRecord
	if err := r.db.Order("created_at ASC, request_id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list ai request runs: %w", err)
	}

	return records, nil
}

func (r *Repository) ListAIRequestFlows() ([]airequestlog.FlowRecord, error) {
	var records []airequestlog.FlowRecord
	if err := r.db.Order("created_at ASC, request_id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list ai request flows: %w", err)
	}

	return records, nil
}

func (r *Repository) ListThirdPartyRequestLogs() ([]airequestlog.ThirdPartyRequestLogRecord, error) {
	var records []airequestlog.ThirdPartyRequestLogRecord
	if err := r.db.Order("created_at ASC, id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list third party request logs: %w", err)
	}

	return records, nil
}

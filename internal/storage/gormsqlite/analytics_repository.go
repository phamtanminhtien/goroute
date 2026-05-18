package gormsqlite

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/usecase/analytics"
	"gorm.io/gorm"
)

type summaryRow struct {
	EffectiveModel string `gorm:"column:effective_model"`
	Requests       int64  `gorm:"column:requests"`
	InputTokens    int64  `gorm:"column:input_tokens"`
	OutputTokens   int64  `gorm:"column:output_tokens"`
}

type timeseriesRow struct {
	BucketStartMs  int64  `gorm:"column:bucket_start_ms"`
	EffectiveModel string `gorm:"column:effective_model"`
	Requests       int64  `gorm:"column:requests"`
	InputTokens    int64  `gorm:"column:input_tokens"`
	OutputTokens   int64  `gorm:"column:output_tokens"`
}

type providerBreakdownRow struct {
	ProviderID     string `gorm:"column:provider_id"`
	ProviderName   string `gorm:"column:provider_name"`
	EffectiveModel string `gorm:"column:effective_model"`
	Requests       int64  `gorm:"column:requests"`
	InputTokens    int64  `gorm:"column:input_tokens"`
	OutputTokens   int64  `gorm:"column:output_tokens"`
}

func (r *Repository) AnalyticsSummary(filters analytics.Filters) ([]analytics.SummaryAggregate, error) {
	var rows []summaryRow
	query := applyAnalyticsFilters(r.db.Model(&airequestlog.RunRecord{}), filters)
	if err := query.Select(
		effectiveModelExpression() + " AS effective_model, COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) AS input_tokens, COALESCE(SUM(completion_tokens), 0) AS output_tokens",
	).Group("effective_model").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query analytics summary: %w", err)
	}

	items := make([]analytics.SummaryAggregate, 0, len(rows))
	for _, row := range rows {
		items = append(items, analytics.SummaryAggregate{
			EffectiveModel: row.EffectiveModel,
			Requests:       row.Requests,
			InputTokens:    row.InputTokens,
			OutputTokens:   row.OutputTokens,
		})
	}

	return items, nil
}

func (r *Repository) AnalyticsTimeseries(filters analytics.Filters, bucket analytics.Bucket) ([]analytics.TimeseriesAggregate, error) {
	expr := bucketStartExpression(bucket)
	var rows []timeseriesRow
	query := applyAnalyticsFilters(r.db.Model(&airequestlog.RunRecord{}), filters)
	if err := query.Select(
		expr + " AS bucket_start_ms, " + effectiveModelExpression() + " AS effective_model, COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) AS input_tokens, COALESCE(SUM(completion_tokens), 0) AS output_tokens",
	).Group("bucket_start_ms, effective_model").Order("bucket_start_ms ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query analytics timeseries: %w", err)
	}

	items := make([]analytics.TimeseriesAggregate, 0, len(rows))
	for _, row := range rows {
		items = append(items, analytics.TimeseriesAggregate{
			BucketStart:    time.UnixMilli(row.BucketStartMs).UTC(),
			EffectiveModel: row.EffectiveModel,
			Requests:       row.Requests,
			InputTokens:    row.InputTokens,
			OutputTokens:   row.OutputTokens,
		})
	}

	return items, nil
}

func (r *Repository) AnalyticsProviderBreakdown(filters analytics.Filters) ([]analytics.ProviderBreakdownAggregate, error) {
	var rows []providerBreakdownRow
	query := applyAnalyticsFilters(r.db.Model(&airequestlog.RunRecord{}), filters)
	if err := query.Select(
		"provider_id, provider_name, " + effectiveModelExpression() + " AS effective_model, COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) AS input_tokens, COALESCE(SUM(completion_tokens), 0) AS output_tokens",
	).Group("provider_id, provider_name, effective_model").Order("requests DESC, provider_id ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query analytics provider breakdown: %w", err)
	}

	items := make([]analytics.ProviderBreakdownAggregate, 0, len(rows))
	for _, row := range rows {
		items = append(items, analytics.ProviderBreakdownAggregate{
			ProviderID:     row.ProviderID,
			ProviderName:   row.ProviderName,
			EffectiveModel: row.EffectiveModel,
			Requests:       row.Requests,
			InputTokens:    row.InputTokens,
			OutputTokens:   row.OutputTokens,
		})
	}

	return items, nil
}

func (r *Repository) AnalyticsRecentRequests(filters analytics.Filters, limit int) ([]analytics.RecentRequestAggregate, error) {
	return r.analyticsRequestRuns(filters, limit, 0)
}

func (r *Repository) AnalyticsRequestLogs(filters analytics.Filters, limit int, offset int) ([]analytics.RecentRequestAggregate, error) {
	return r.analyticsRequestRuns(filters, limit, offset)
}

func (r *Repository) analyticsRequestRuns(filters analytics.Filters, limit int, offset int) ([]analytics.RecentRequestAggregate, error) {
	var rows []airequestlog.RunRecord
	query := applyAnalyticsFilters(r.db.Model(&airequestlog.RunRecord{}), filters)
	if err := query.Order("created_at DESC, request_id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query analytics recent requests: %w", err)
	}

	items := make([]analytics.RecentRequestAggregate, 0, len(rows))
	for _, row := range rows {
		model := row.ResolvedModel
		if strings.TrimSpace(model) == "" {
			model = row.RequestedModel
		}

		items = append(items, analytics.RecentRequestAggregate{
			RequestID:      row.RequestID,
			Timestamp:      time.UnixMilli(row.CreatedAt).UTC(),
			Model:          model,
			Path:           row.Path,
			StatusCode:     row.StatusCode,
			LatencyMs:      row.DurationMs,
			InputTokens:    int64(row.PromptTokens),
			OutputTokens:   int64(row.CompletionTokens),
			ProviderID:     row.ProviderID,
			ProviderName:   row.ProviderName,
			ConnectionID:   row.FinalConnectionID,
			ConnectionName: row.FinalConnectionName,
			Status:         row.FinalErrorCategory,
		})
	}

	return items, nil
}

func (r *Repository) AnalyticsRequestLogDetail(requestID string) (analytics.RequestLogRecords, error) {
	var run airequestlog.RunRecord
	if err := r.db.Where("request_id = ?", requestID).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return analytics.RequestLogRecords{}, analytics.ErrRequestLogNotFound
		}
		return analytics.RequestLogRecords{}, fmt.Errorf("query ai request run %q: %w", requestID, err)
	}

	var flow airequestlog.FlowRecord
	var flowPointer *airequestlog.FlowRecord
	if err := r.db.Where("run_id = ?", run.ID).Order("created_at DESC, id DESC").First(&flow).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return analytics.RequestLogRecords{}, fmt.Errorf("query ai request flow for run %d: %w", run.ID, err)
		}
	} else {
		flowPointer = &flow
	}

	thirdPartyLogs := make([]airequestlog.ThirdPartyRequestLogRecord, 0)
	if err := r.db.Where("run_id = ?", run.ID).Order("attempt_index ASC, created_at ASC, id ASC").Find(&thirdPartyLogs).Error; err != nil {
		return analytics.RequestLogRecords{}, fmt.Errorf("query third party request logs for run %d: %w", run.ID, err)
	}

	var rtk airequestlog.RTKRecord
	var rtkPointer *airequestlog.RTKRecord
	if err := r.db.Where("run_id = ?", run.ID).First(&rtk).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return analytics.RequestLogRecords{}, fmt.Errorf("query rtk record for run %d: %w", run.ID, err)
		}
	} else {
		rtkPointer = &rtk
	}

	return analytics.RequestLogRecords{
		Run:            run,
		Flow:           flowPointer,
		ThirdPartyLogs: thirdPartyLogs,
		RTK:            rtkPointer,
	}, nil
}

func applyAnalyticsFilters(query *gorm.DB, filters analytics.Filters) *gorm.DB {
	if !filters.From.IsZero() && !filters.To.IsZero() {
		query = query.Where("created_at >= ? AND created_at < ?", filters.From.UnixMilli(), filters.To.UnixMilli())
	}

	if filters.ProviderID != "" {
		query = query.Where("provider_id = ?", filters.ProviderID)
	}
	if filters.ConnectionID != "" {
		query = query.Where("final_connection_id = ?", filters.ConnectionID)
	}
	if filters.Path != "" {
		query = query.Where("path = ?", filters.Path)
	}
	if filters.Model != "" {
		query = query.Where("CASE WHEN resolved_model <> '' THEN resolved_model ELSE requested_model END = ?", filters.Model)
	}

	return query
}

func bucketStartExpression(bucket analytics.Bucket) string {
	switch bucket {
	case analytics.BucketHour:
		return "(created_at / 3600000) * 3600000"
	case analytics.BucketDay:
		return "(created_at / 86400000) * 86400000"
	case analytics.BucketWeek:
		return "CAST(strftime('%s', date(created_at / 1000, 'unixepoch', 'utc', printf('-%d days', (CAST(strftime('%w', created_at / 1000, 'unixepoch', 'utc') AS integer) + 6) % 7))) AS integer) * 1000"
	default:
		return "(created_at / 3600000) * 3600000"
	}
}

func effectiveModelExpression() string {
	return "CASE WHEN resolved_model <> '' THEN resolved_model ELSE requested_model END"
}

package analytics

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

const (
	maxRangeDuration = 365 * 24 * time.Hour
	defaultLimit     = 20
	maxLimit         = 100
)

var ErrRequestLogNotFound = errors.New("request log not found")

type Repository interface {
	AnalyticsSummary(filters Filters) ([]SummaryAggregate, error)
	AnalyticsTimeseries(filters Filters, bucket Bucket) ([]TimeseriesAggregate, error)
	AnalyticsProviderBreakdown(filters Filters) ([]ProviderBreakdownAggregate, error)
	AnalyticsRecentRequests(filters Filters, limit int) ([]RecentRequestAggregate, error)
	AnalyticsRequestLogs(filters Filters, limit int, offset int) ([]RecentRequestAggregate, error)
	AnalyticsRequestLogDetail(requestID string) (RequestLogRecords, error)
}

type CatalogSource interface {
	Catalog() provider.Catalog
}

type ProviderModelRepository interface {
	ListProviderModels() ([]provider.ModelRecord, error)
}

type Bucket string

const (
	BucketHour Bucket = "hour"
	BucketDay  Bucket = "day"
	BucketWeek Bucket = "week"
)

type Filters struct {
	From         time.Time
	To           time.Time
	ProviderID   string
	ConnectionID string
	Model        string
	Path         string
}

type SummaryAggregate struct {
	Requests       int64
	InputTokens    int64
	OutputTokens   int64
	EffectiveModel string
}

type TimeseriesAggregate struct {
	BucketStart    time.Time
	EffectiveModel string
	Requests       int64
	InputTokens    int64
	OutputTokens   int64
}

type ProviderBreakdownAggregate struct {
	ProviderID     string
	ProviderName   string
	EffectiveModel string
	Requests       int64
	InputTokens    int64
	OutputTokens   int64
}

type RecentRequestAggregate struct {
	RequestID      string
	Timestamp      time.Time
	Model          string
	Path           string
	StatusCode     int
	LatencyMs      int64
	InputTokens    int64
	OutputTokens   int64
	EstimatedCost  float64
	ProviderID     string
	ProviderName   string
	ConnectionID   string
	ConnectionName string
	Status         string
}

type SummaryResult struct {
	Requests       SummaryMetricInt
	InputTokens    SummaryMetricInt
	OutputTokens   SummaryMetricInt
	EstimatedCost  SummaryMetricFloat
	AverageCostUSD float64
}

type SummaryMetricInt struct {
	Value      int64
	DeltaLabel string
	DeltaTone  string
}

type SummaryMetricFloat struct {
	Value      float64
	DeltaLabel string
	DeltaTone  string
}

type TimeseriesPoint struct {
	BucketStart   time.Time
	BucketEnd     time.Time
	Requests      int64
	InputTokens   int64
	OutputTokens  int64
	EstimatedCost float64
}

type ProviderBreakdownItem struct {
	ProviderID    string
	ProviderName  string
	Requests      int64
	RequestsPct   float64
	InputTokens   int64
	OutputTokens  int64
	EstimatedCost float64
}

type RecentRequestsPage struct {
	Items    []RecentRequestAggregate
	Returned int
	HasMore  bool
	Limit    int
	Page     int
}

type RequestLogRecords struct {
	Run            airequestlog.RunRecord
	Flow           *airequestlog.FlowRecord
	ThirdPartyLogs []airequestlog.ThirdPartyRequestLogRecord
	RTK            *airequestlog.RTKRecord
}

type RequestLogDetail struct {
	RequestLogRecords
	Model         string
	Status        string
	EstimatedCost float64
}

type Service struct {
	repo          Repository
	pricingMu     sync.RWMutex
	inputPricing  map[string]float64
	outputPricing map[string]float64
	catalogSource CatalogSource
	modelRepo     ProviderModelRepository
}

func NewService(repo Repository, catalog provider.Catalog) *Service {
	service := &Service{
		repo:          repo,
		inputPricing:  map[string]float64{},
		outputPricing: map[string]float64{},
	}

	for _, item := range catalog.Providers {
		for _, model := range item.Models {
			service.inputPricing[model.ID] = model.InputPricePerMillionUSD
			service.outputPricing[model.ID] = model.OutputPricePerMillionUSD
		}
	}

	return service
}

func NewServiceWithPricingSource(repo Repository, catalogSource CatalogSource, modelRepo ProviderModelRepository) *Service {
	service := NewService(repo, catalogSource.Catalog())
	service.catalogSource = catalogSource
	service.modelRepo = modelRepo
	return service
}

func (s *Service) refreshPricing() error {
	if s.catalogSource == nil {
		return nil
	}

	catalog := s.catalogSource.Catalog()
	if s.modelRepo != nil {
		records, err := s.modelRepo.ListProviderModels()
		if err != nil {
			return err
		}
		catalog = catalog.WithModelRecords(records)
	}

	inputPricing := make(map[string]float64)
	outputPricing := make(map[string]float64)
	for _, item := range catalog.Providers {
		for _, model := range item.Models {
			inputPricing[model.ID] = model.InputPricePerMillionUSD
			outputPricing[model.ID] = model.OutputPricePerMillionUSD
		}
	}

	s.pricingMu.Lock()
	defer s.pricingMu.Unlock()
	s.inputPricing = inputPricing
	s.outputPricing = outputPricing
	return nil
}

func ParseFilters(values url.Values) (Filters, error) {
	from, err := parseUTCTimestamp(values.Get("from"), "from")
	if err != nil {
		return Filters{}, err
	}

	to, err := parseUTCTimestamp(values.Get("to"), "to")
	if err != nil {
		return Filters{}, err
	}

	if !to.After(from) {
		return Filters{}, fmt.Errorf("from must be before to")
	}
	if to.Sub(from) > maxRangeDuration {
		return Filters{}, fmt.Errorf("range too large")
	}

	return Filters{
		From:         from,
		To:           to,
		ProviderID:   strings.TrimSpace(values.Get("provider_id")),
		ConnectionID: strings.TrimSpace(values.Get("connection_id")),
		Model:        strings.TrimSpace(values.Get("model")),
		Path:         strings.TrimSpace(values.Get("path")),
	}, nil
}

func ParseOptionalFilters(values url.Values) (Filters, error) {
	var from time.Time
	var to time.Time
	var err error

	if strings.TrimSpace(values.Get("from")) != "" {
		from, err = parseUTCTimestamp(values.Get("from"), "from")
		if err != nil {
			return Filters{}, err
		}
	}
	if strings.TrimSpace(values.Get("to")) != "" {
		to, err = parseUTCTimestamp(values.Get("to"), "to")
		if err != nil {
			return Filters{}, err
		}
	}

	if !from.IsZero() || !to.IsZero() {
		if from.IsZero() {
			return Filters{}, fmt.Errorf("missing from")
		}
		if to.IsZero() {
			return Filters{}, fmt.Errorf("missing to")
		}
		if !to.After(from) {
			return Filters{}, fmt.Errorf("from must be before to")
		}
		if to.Sub(from) > maxRangeDuration {
			return Filters{}, fmt.Errorf("range too large")
		}
	}

	return Filters{
		From:         from,
		To:           to,
		ProviderID:   strings.TrimSpace(values.Get("provider_id")),
		ConnectionID: strings.TrimSpace(values.Get("connection_id")),
		Model:        strings.TrimSpace(values.Get("model")),
		Path:         strings.TrimSpace(values.Get("path")),
	}, nil
}

func ParseBucket(value string) (Bucket, error) {
	switch Bucket(strings.TrimSpace(value)) {
	case BucketHour, BucketDay, BucketWeek:
		return Bucket(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("unsupported bucket")
	}
}

func ParsePage(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 1, nil
	}

	page, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid page")
	}
	if page < 1 {
		return 0, fmt.Errorf("invalid page")
	}

	return page, nil
}

func ParseLimit(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid limit")
	}
	if limit < 1 || limit > maxLimit {
		return 0, fmt.Errorf("invalid limit")
	}

	return limit, nil
}

func (s *Service) Summary(filters Filters) (SummaryResult, error) {
	if err := s.refreshPricing(); err != nil {
		return SummaryResult{}, err
	}

	summary, err := s.repo.AnalyticsSummary(filters)
	if err != nil {
		return SummaryResult{}, err
	}

	current := summarizeRows(summary, s)
	previous, err := s.previousSummary(filters)
	if err != nil {
		return SummaryResult{}, err
	}

	result := SummaryResult{
		Requests: SummaryMetricInt{
			Value:      current.Requests.Value,
			DeltaLabel: percentDeltaLabel(current.Requests.Value, previous.Requests.Value, "prev period"),
			DeltaTone:  growthDeltaTone(current.Requests.Value, previous.Requests.Value),
		},
		InputTokens: SummaryMetricInt{
			Value:      current.InputTokens.Value,
			DeltaLabel: percentDeltaLabel(current.InputTokens.Value, previous.InputTokens.Value, "previous period"),
			DeltaTone:  warningDeltaTone(current.InputTokens.Value, previous.InputTokens.Value),
		},
		OutputTokens: SummaryMetricInt{
			Value:      current.OutputTokens.Value,
			DeltaLabel: percentDeltaLabel(current.OutputTokens.Value, previous.OutputTokens.Value, "previous period"),
			DeltaTone:  growthDeltaTone(current.OutputTokens.Value, previous.OutputTokens.Value),
		},
		EstimatedCost: SummaryMetricFloat{
			Value:      current.EstimatedCost.Value,
			DeltaLabel: fmt.Sprintf("$%.3f avg / req", current.AverageCostUSD),
			DeltaTone:  "critical",
		},
		AverageCostUSD: current.AverageCostUSD,
	}

	return result, nil
}

func summarizeRows(rows []SummaryAggregate, service *Service) SummaryResult {
	result := SummaryResult{}
	for _, item := range rows {
		result.Requests.Value += item.Requests
		result.InputTokens.Value += item.InputTokens
		result.OutputTokens.Value += item.OutputTokens
		result.EstimatedCost.Value += service.calculateCost(item.EffectiveModel, item.InputTokens, item.OutputTokens)
	}
	if result.Requests.Value == 0 {
		result.AverageCostUSD = 0
	} else {
		result.AverageCostUSD = result.EstimatedCost.Value / float64(result.Requests.Value)
	}

	return result
}

func (s *Service) previousSummary(filters Filters) (SummaryResult, error) {
	duration := filters.To.Sub(filters.From)
	previousFilters := filters
	previousFilters.To = filters.From
	previousFilters.From = filters.From.Add(-duration)

	rows, err := s.repo.AnalyticsSummary(previousFilters)
	if err != nil {
		return SummaryResult{}, err
	}

	return summarizeRows(rows, s), nil
}

func (s *Service) Timeseries(filters Filters, bucket Bucket) ([]TimeseriesPoint, error) {
	if err := s.refreshPricing(); err != nil {
		return nil, err
	}

	rows, err := s.repo.AnalyticsTimeseries(filters, bucket)
	if err != nil {
		return nil, err
	}

	byStart := make(map[int64][]TimeseriesAggregate, len(rows))
	for _, row := range rows {
		key := row.BucketStart.UnixMilli()
		byStart[key] = append(byStart[key], row)
	}

	points := make([]TimeseriesPoint, 0)
	for current := truncateToBucket(filters.From, bucket); current.Before(filters.To); current = addBucket(current, bucket) {
		current = current.UTC()
		row, ok := byStart[current.UnixMilli()]
		if !ok {
			points = append(points, TimeseriesPoint{
				BucketStart: current,
				BucketEnd:   addBucket(current, bucket).Add(-time.Second),
			})
			continue
		}

		estimatedCost := 0.0
		for _, item := range row {
			estimatedCost += s.calculateCost(item.EffectiveModel, item.InputTokens, item.OutputTokens)
		}

		points = append(points, TimeseriesPoint{
			BucketStart:   current,
			BucketEnd:     addBucket(current, bucket).Add(-time.Second),
			Requests:      sumTimeseriesRequests(row),
			InputTokens:   sumTimeseriesInputTokens(row),
			OutputTokens:  sumTimeseriesOutputTokens(row),
			EstimatedCost: estimatedCost,
		})
	}

	return points, nil
}

func (s *Service) ProviderBreakdown(filters Filters) ([]ProviderBreakdownItem, error) {
	if err := s.refreshPricing(); err != nil {
		return nil, err
	}

	items, err := s.repo.AnalyticsProviderBreakdown(filters)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string]*ProviderBreakdownItem)
	var total int64
	for _, item := range items {
		key := item.ProviderID + "\x00" + item.ProviderName
		group := grouped[key]
		if group == nil {
			group = &ProviderBreakdownItem{
				ProviderID:   item.ProviderID,
				ProviderName: item.ProviderName,
			}
			grouped[key] = group
		}
		group.Requests += item.Requests
		group.InputTokens += item.InputTokens
		group.OutputTokens += item.OutputTokens
		group.EstimatedCost += s.calculateCost(item.EffectiveModel, item.InputTokens, item.OutputTokens)
		total += item.Requests
	}

	result := make([]ProviderBreakdownItem, 0, len(grouped))
	for _, item := range grouped {
		if total > 0 {
			item.RequestsPct = (float64(item.Requests) / float64(total)) * 100
		}
		result = append(result, *item)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Requests == result[j].Requests {
			return result[i].ProviderID < result[j].ProviderID
		}
		return result[i].Requests > result[j].Requests
	})

	return result, nil
}

func (s *Service) RecentRequests(filters Filters, limit int) (RecentRequestsPage, error) {
	if err := s.refreshPricing(); err != nil {
		return RecentRequestsPage{}, err
	}

	rows, err := s.repo.AnalyticsRecentRequests(filters, limit+1)
	if err != nil {
		return RecentRequestsPage{}, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	for i := range rows {
		rows[i].Status = normalizeStatus(rows[i].StatusCode, rows[i].Status)
		rows[i].EstimatedCost = s.calculateCost(rows[i].Model, rows[i].InputTokens, rows[i].OutputTokens)
	}

	return RecentRequestsPage{
		Items:    rows,
		Returned: len(rows),
		HasMore:  hasMore,
		Limit:    limit,
		Page:     1,
	}, nil
}

func (s *Service) RequestLogs(filters Filters, limit int, page int) (RecentRequestsPage, error) {
	if err := s.refreshPricing(); err != nil {
		return RecentRequestsPage{}, err
	}

	offset := (page - 1) * limit
	rows, err := s.repo.AnalyticsRequestLogs(filters, limit+1, offset)
	if err != nil {
		return RecentRequestsPage{}, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	for i := range rows {
		rows[i].Status = normalizeStatus(rows[i].StatusCode, rows[i].Status)
		rows[i].EstimatedCost = s.calculateCost(rows[i].Model, rows[i].InputTokens, rows[i].OutputTokens)
	}

	return RecentRequestsPage{
		Items:    rows,
		Returned: len(rows),
		HasMore:  hasMore,
		Limit:    limit,
		Page:     page,
	}, nil
}

func (s *Service) RequestLogDetail(requestID string) (RequestLogDetail, error) {
	if err := s.refreshPricing(); err != nil {
		return RequestLogDetail{}, err
	}

	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return RequestLogDetail{}, ErrRequestLogNotFound
	}

	records, err := s.repo.AnalyticsRequestLogDetail(requestID)
	if err != nil {
		return RequestLogDetail{}, err
	}

	model := records.Run.ResolvedModel
	if strings.TrimSpace(model) == "" {
		model = records.Run.RequestedModel
	}

	return RequestLogDetail{
		RequestLogRecords: records,
		Model:             model,
		Status:            normalizeStatus(records.Run.StatusCode, records.Run.FinalErrorCategory),
		EstimatedCost:     s.calculateCost(model, int64(records.Run.PromptTokens), int64(records.Run.CompletionTokens)),
	}, nil
}

func (s *Service) calculateCost(model string, inputTokens int64, outputTokens int64) float64 {
	return s.calculateCostByModel(inputTokens, outputTokens, model)
}

func (s *Service) calculateCostByModel(inputTokens int64, outputTokens int64, model string) float64 {
	s.pricingMu.RLock()
	defer s.pricingMu.RUnlock()

	inputPrice := s.inputPricing[model]
	outputPrice := s.outputPricing[model]

	return (float64(inputTokens)/1_000_000)*inputPrice + (float64(outputTokens)/1_000_000)*outputPrice
}

func sumTimeseriesRequests(items []TimeseriesAggregate) int64 {
	var total int64
	for _, item := range items {
		total += item.Requests
	}
	return total
}

func sumTimeseriesInputTokens(items []TimeseriesAggregate) int64 {
	var total int64
	for _, item := range items {
		total += item.InputTokens
	}
	return total
}

func sumTimeseriesOutputTokens(items []TimeseriesAggregate) int64 {
	var total int64
	for _, item := range items {
		total += item.OutputTokens
	}
	return total
}

func parseUTCTimestamp(value string, field string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("missing %s", field)
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s", field)
	}

	return parsed.UTC(), nil
}

func truncateToBucket(value time.Time, bucket Bucket) time.Time {
	value = value.UTC()
	switch bucket {
	case BucketHour:
		return value.Truncate(time.Hour)
	case BucketDay:
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	case BucketWeek:
		offset := (int(value.Weekday()) + 6) % 7
		day := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
		return day.AddDate(0, 0, -offset)
	default:
		return value
	}
}

func addBucket(value time.Time, bucket Bucket) time.Time {
	switch bucket {
	case BucketHour:
		return value.Add(time.Hour)
	case BucketDay:
		return value.AddDate(0, 0, 1)
	case BucketWeek:
		return value.AddDate(0, 0, 7)
	default:
		return value
	}
}

func normalizeStatus(statusCode int, finalErrorCategory string) string {
	if statusCode >= 200 && statusCode < 300 && strings.TrimSpace(finalErrorCategory) == "" {
		return "completed"
	}

	return "failed"
}

func percentDeltaLabel(current int64, previous int64, suffix string) string {
	if previous <= 0 {
		if current <= 0 {
			return "0.0% vs " + suffix
		}

		return "+100.0% vs " + suffix
	}

	delta := ((float64(current) - float64(previous)) / float64(previous)) * 100
	return fmt.Sprintf("%+.1f%% vs %s", delta, suffix)
}

func growthDeltaTone(current int64, previous int64) string {
	if current >= previous {
		return "positive"
	}

	return "critical"
}

func warningDeltaTone(current int64, previous int64) string {
	if current > previous {
		return "warning"
	}
	if current < previous {
		return "critical"
	}

	return "warning"
}

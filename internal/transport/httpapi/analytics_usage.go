package httpapi

import (
	"net/http"
	"time"

	"github.com/phamtanminhtien/goroute/internal/usecase/analytics"
)

type analyticsFiltersResponse struct {
	ProviderID   *string `json:"provider_id"`
	ConnectionID *string `json:"connection_id"`
	Model        *string `json:"model"`
	Path         *string `json:"path"`
}

type analyticsSummaryResponse struct {
	From             string                   `json:"from"`
	To               string                   `json:"to"`
	GeneratedAt      string                   `json:"generated_at"`
	Filters          analyticsFiltersResponse `json:"filters"`
	Requests         analyticsValueInt        `json:"requests"`
	InputTokens      analyticsValueInt        `json:"input_tokens"`
	OutputTokens     analyticsValueInt        `json:"output_tokens"`
	EstimatedCostUSD analyticsCostSummary     `json:"estimated_cost_usd"`
}

type analyticsValueInt struct {
	Value      int64  `json:"value"`
	DeltaLabel string `json:"delta_label,omitempty"`
	DeltaTone  string `json:"delta_tone,omitempty"`
}

type analyticsCostSummary struct {
	Value         float64 `json:"value"`
	AvgPerRequest float64 `json:"avg_per_request"`
	DeltaLabel    string  `json:"delta_label,omitempty"`
	DeltaTone     string  `json:"delta_tone,omitempty"`
}

type analyticsTimeseriesResponse struct {
	From        string                   `json:"from"`
	To          string                   `json:"to"`
	Bucket      analytics.Bucket         `json:"bucket"`
	GeneratedAt string                   `json:"generated_at"`
	Filters     analyticsFiltersResponse `json:"filters"`
	Points      []analyticsTimeseriesRow `json:"points"`
}

type analyticsTimeseriesRow struct {
	BucketStart      string  `json:"bucket_start"`
	BucketEnd        string  `json:"bucket_end"`
	Requests         int64   `json:"requests"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

type analyticsProviderBreakdownResponse struct {
	From        string                          `json:"from"`
	To          string                          `json:"to"`
	GeneratedAt string                          `json:"generated_at"`
	Filters     analyticsFiltersResponse        `json:"filters"`
	Items       []analyticsProviderBreakdownRow `json:"items"`
}

type analyticsProviderBreakdownRow struct {
	ProviderID       string  `json:"provider_id"`
	ProviderName     string  `json:"provider_name"`
	Requests         int64   `json:"requests"`
	RequestsPct      float64 `json:"requests_pct"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

type analyticsRecentRequestsResponse struct {
	From        string                      `json:"from"`
	To          string                      `json:"to"`
	GeneratedAt string                      `json:"generated_at"`
	Filters     analyticsFiltersResponse    `json:"filters"`
	Items       []analyticsRecentRequestRow `json:"items"`
	Page        analyticsRecentRequestsPage `json:"page"`
}

type analyticsRecentRequestRow struct {
	RequestID        string  `json:"request_id"`
	Timestamp        string  `json:"timestamp"`
	Model            string  `json:"model"`
	Path             string  `json:"path"`
	Status           string  `json:"status"`
	StatusCode       int     `json:"status_code"`
	LatencyMs        int64   `json:"latency_ms"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	ProviderID       string  `json:"provider_id"`
	ProviderName     string  `json:"provider_name"`
	ConnectionID     string  `json:"connection_id"`
	ConnectionName   string  `json:"connection_name"`
}

type analyticsRecentRequestsPage struct {
	Limit    int  `json:"limit"`
	Returned int  `json:"returned"`
	HasMore  bool `json:"has_more"`
}

func analyticsUsageSummaryHandler(service *analytics.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		filters, ok := parseAnalyticsFiltersOrWriteError(r, w)
		if !ok {
			return
		}

		result, err := service.Summary(filters)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, analyticsSummaryResponse{
			From:        formatAnalyticsTime(filters.From),
			To:          formatAnalyticsTime(filters.To),
			GeneratedAt: formatAnalyticsTime(time.Now().UTC()),
			Filters:     buildAnalyticsFiltersResponse(filters),
			Requests: analyticsValueInt{
				Value:      result.Requests.Value,
				DeltaLabel: result.Requests.DeltaLabel,
				DeltaTone:  result.Requests.DeltaTone,
			},
			InputTokens: analyticsValueInt{
				Value:      result.InputTokens.Value,
				DeltaLabel: result.InputTokens.DeltaLabel,
				DeltaTone:  result.InputTokens.DeltaTone,
			},
			OutputTokens: analyticsValueInt{
				Value:      result.OutputTokens.Value,
				DeltaLabel: result.OutputTokens.DeltaLabel,
				DeltaTone:  result.OutputTokens.DeltaTone,
			},
			EstimatedCostUSD: analyticsCostSummary{
				Value:         result.EstimatedCost.Value,
				AvgPerRequest: result.AverageCostUSD,
				DeltaLabel:    result.EstimatedCost.DeltaLabel,
				DeltaTone:     result.EstimatedCost.DeltaTone,
			},
		})
	})
}

func analyticsUsageTimeseriesHandler(service *analytics.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		filters, ok := parseAnalyticsFiltersOrWriteError(r, w)
		if !ok {
			return
		}

		bucket, err := analytics.ParseBucket(r.URL.Query().Get("bucket"))
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		points, err := service.Timeseries(filters, bucket)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		items := make([]analyticsTimeseriesRow, 0, len(points))
		for _, point := range points {
			items = append(items, analyticsTimeseriesRow{
				BucketStart:      formatAnalyticsTime(point.BucketStart),
				BucketEnd:        formatAnalyticsTime(point.BucketEnd),
				Requests:         point.Requests,
				InputTokens:      point.InputTokens,
				OutputTokens:     point.OutputTokens,
				EstimatedCostUSD: point.EstimatedCost,
			})
		}

		writeJSON(w, http.StatusOK, analyticsTimeseriesResponse{
			From:        formatAnalyticsTime(filters.From),
			To:          formatAnalyticsTime(filters.To),
			Bucket:      bucket,
			GeneratedAt: formatAnalyticsTime(time.Now().UTC()),
			Filters:     buildAnalyticsFiltersResponse(filters),
			Points:      items,
		})
	})
}

func analyticsUsageProviderBreakdownHandler(service *analytics.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		filters, ok := parseAnalyticsFiltersOrWriteError(r, w)
		if !ok {
			return
		}

		items, err := service.ProviderBreakdown(filters)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		responseItems := make([]analyticsProviderBreakdownRow, 0, len(items))
		for _, item := range items {
			responseItems = append(responseItems, analyticsProviderBreakdownRow{
				ProviderID:       item.ProviderID,
				ProviderName:     item.ProviderName,
				Requests:         item.Requests,
				RequestsPct:      item.RequestsPct,
				InputTokens:      item.InputTokens,
				OutputTokens:     item.OutputTokens,
				EstimatedCostUSD: item.EstimatedCost,
			})
		}

		writeJSON(w, http.StatusOK, analyticsProviderBreakdownResponse{
			From:        formatAnalyticsTime(filters.From),
			To:          formatAnalyticsTime(filters.To),
			GeneratedAt: formatAnalyticsTime(time.Now().UTC()),
			Filters:     buildAnalyticsFiltersResponse(filters),
			Items:       responseItems,
		})
	})
}

func analyticsUsageRecentRequestsHandler(service *analytics.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		filters, ok := parseAnalyticsFiltersOrWriteError(r, w)
		if !ok {
			return
		}

		limit, err := analytics.ParseLimit(r.URL.Query().Get("limit"))
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		page, err := service.RecentRequests(filters, limit)
		if err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		items := make([]analyticsRecentRequestRow, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, analyticsRecentRequestRow{
				RequestID:        item.RequestID,
				Timestamp:        formatAnalyticsTime(item.Timestamp),
				Model:            item.Model,
				Path:             item.Path,
				Status:           item.Status,
				StatusCode:       item.StatusCode,
				LatencyMs:        item.LatencyMs,
				InputTokens:      item.InputTokens,
				OutputTokens:     item.OutputTokens,
				EstimatedCostUSD: item.EstimatedCost,
				ProviderID:       item.ProviderID,
				ProviderName:     item.ProviderName,
				ConnectionID:     item.ConnectionID,
				ConnectionName:   item.ConnectionName,
			})
		}

		writeJSON(w, http.StatusOK, analyticsRecentRequestsResponse{
			From:        formatAnalyticsTime(filters.From),
			To:          formatAnalyticsTime(filters.To),
			GeneratedAt: formatAnalyticsTime(time.Now().UTC()),
			Filters:     buildAnalyticsFiltersResponse(filters),
			Items:       items,
			Page: analyticsRecentRequestsPage{
				Limit:    page.Limit,
				Returned: page.Returned,
				HasMore:  page.HasMore,
			},
		})
	})
}

func parseAnalyticsFiltersOrWriteError(r *http.Request, w http.ResponseWriter) (analytics.Filters, bool) {
	filters, err := analytics.ParseFilters(r.URL.Query())
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
		return analytics.Filters{}, false
	}

	return filters, true
}

func buildAnalyticsFiltersResponse(filters analytics.Filters) analyticsFiltersResponse {
	return analyticsFiltersResponse{
		ProviderID:   stringPointerOrNil(filters.ProviderID),
		ConnectionID: stringPointerOrNil(filters.ConnectionID),
		Model:        stringPointerOrNil(filters.Model),
		Path:         stringPointerOrNil(filters.Path),
	}
}

func stringPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func formatAnalyticsTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

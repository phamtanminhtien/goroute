package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/airequestlog"
	"github.com/phamtanminhtien/goroute/internal/storage/gormsqlite"
)

func TestUsageAnalyticsSummaryReturnsAggregatesAndCost(t *testing.T) {
	handler, repo := analyticsTestServer(t)
	seedAnalyticsRuns(t, repo,
		airequestlog.RunRecord{
			RequestID:           "prev-1",
			Path:                "/v1/chat/completions",
			RequestedModel:      "ignored",
			ResolvedModel:       "cx/gpt-5.4",
			ProviderID:          "cx",
			ProviderName:        "Codex",
			FinalConnectionID:   "codex-1",
			FinalConnectionName: "codex-user",
			StatusCode:          200,
			PromptTokens:        80000,
			CompletionTokens:    25000,
			DurationMs:          1400,
			CreatedAt:           mustParseRFC3339(t, "2026-05-15T02:15:00Z").UnixMilli(),
		},
		airequestlog.RunRecord{
			RequestID:           "prev-2",
			Path:                "/v1/chat/completions",
			RequestedModel:      "opena/gpt-4.1",
			ProviderID:          "opena",
			ProviderName:        "OpenAI",
			FinalConnectionID:   "openai-1",
			FinalConnectionName: "openai-user",
			StatusCode:          200,
			PromptTokens:        120000,
			CompletionTokens:    75000,
			DurationMs:          950,
			CreatedAt:           mustParseRFC3339(t, "2026-05-15T03:45:00Z").UnixMilli(),
		},
		airequestlog.RunRecord{
			RequestID:           "req-1",
			Path:                "/v1/chat/completions",
			RequestedModel:      "ignored",
			ResolvedModel:       "cx/gpt-5.4",
			ProviderID:          "cx",
			ProviderName:        "Codex",
			FinalConnectionID:   "codex-1",
			FinalConnectionName: "codex-user",
			StatusCode:          200,
			PromptTokens:        100000,
			CompletionTokens:    50000,
			DurationMs:          1200,
			CreatedAt:           mustParseRFC3339(t, "2026-05-16T01:15:00Z").UnixMilli(),
		},
		airequestlog.RunRecord{
			RequestID:           "req-2",
			Path:                "/v1/chat/completions",
			RequestedModel:      "opena/gpt-4.1",
			ProviderID:          "opena",
			ProviderName:        "OpenAI",
			FinalConnectionID:   "openai-1",
			FinalConnectionName: "openai-user",
			StatusCode:          200,
			PromptTokens:        200000,
			CompletionTokens:    100000,
			DurationMs:          900,
			CreatedAt:           mustParseRFC3339(t, "2026-05-16T03:45:00Z").UnixMilli(),
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/summary?from=2026-05-16T00:00:00Z&to=2026-05-17T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Requests struct {
			Value      int64  `json:"value"`
			DeltaLabel string `json:"delta_label"`
			DeltaTone  string `json:"delta_tone"`
		} `json:"requests"`
		InputTokens struct {
			Value      int64  `json:"value"`
			DeltaLabel string `json:"delta_label"`
			DeltaTone  string `json:"delta_tone"`
		} `json:"input_tokens"`
		OutputTokens struct {
			Value      int64  `json:"value"`
			DeltaLabel string `json:"delta_label"`
			DeltaTone  string `json:"delta_tone"`
		} `json:"output_tokens"`
		EstimatedCostUSD struct {
			Value         float64 `json:"value"`
			AvgPerRequest float64 `json:"avg_per_request"`
			DeltaLabel    string  `json:"delta_label"`
			DeltaTone     string  `json:"delta_tone"`
		} `json:"estimated_cost_usd"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Requests.Value != 2 || response.InputTokens.Value != 300000 || response.OutputTokens.Value != 150000 {
		t.Fatalf("unexpected summary payload %#v", response)
	}

	expectedCost := 0.625 + 1.2
	if !almostEqual(response.EstimatedCostUSD.Value, expectedCost) {
		t.Fatalf("expected total cost %.6f, got %#v", expectedCost, response.EstimatedCostUSD)
	}
	if !almostEqual(response.EstimatedCostUSD.AvgPerRequest, expectedCost/2) {
		t.Fatalf("expected average cost %.6f, got %#v", expectedCost/2, response.EstimatedCostUSD)
	}
	if response.Requests.DeltaLabel != "+0.0% vs prev period" || response.Requests.DeltaTone != "positive" {
		t.Fatalf("unexpected request delta %#v", response.Requests)
	}
	if response.InputTokens.DeltaLabel != "+50.0% vs previous period" || response.InputTokens.DeltaTone != "warning" {
		t.Fatalf("unexpected input delta %#v", response.InputTokens)
	}
	if response.OutputTokens.DeltaLabel != "+50.0% vs previous period" || response.OutputTokens.DeltaTone != "positive" {
		t.Fatalf("unexpected output delta %#v", response.OutputTokens)
	}
	if response.EstimatedCostUSD.DeltaLabel != "$0.913 avg / req" || response.EstimatedCostUSD.DeltaTone != "critical" {
		t.Fatalf("unexpected cost delta %#v", response.EstimatedCostUSD)
	}
}

func TestUsageAnalyticsTimeseriesFillsEmptyBucketsAndSupportsModelFilter(t *testing.T) {
	handler, repo := analyticsTestServer(t)
	seedAnalyticsRuns(t, repo,
		airequestlog.RunRecord{
			RequestID:           "req-1",
			Path:                "/v1/chat/completions",
			RequestedModel:      "fallback",
			ResolvedModel:       "cx/gpt-5.4",
			ProviderID:          "cx",
			ProviderName:        "Codex",
			FinalConnectionID:   "codex-1",
			FinalConnectionName: "codex-user",
			StatusCode:          200,
			PromptTokens:        100000,
			CompletionTokens:    50000,
			DurationMs:          1200,
			CreatedAt:           mustParseRFC3339(t, "2026-05-16T00:15:00Z").UnixMilli(),
		},
		airequestlog.RunRecord{
			RequestID:           "req-2",
			Path:                "/v1/chat/completions",
			RequestedModel:      "opena/gpt-4.1",
			ProviderID:          "opena",
			ProviderName:        "OpenAI",
			FinalConnectionID:   "openai-1",
			FinalConnectionName: "openai-user",
			StatusCode:          200,
			PromptTokens:        200000,
			CompletionTokens:    100000,
			DurationMs:          900,
			CreatedAt:           mustParseRFC3339(t, "2026-05-16T02:45:00Z").UnixMilli(),
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/timeseries?from=2026-05-16T00:00:00Z&to=2026-05-16T04:00:00Z&bucket=hour&model=cx/gpt-5.4", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Points []struct {
			BucketStart      string  `json:"bucket_start"`
			Requests         int64   `json:"requests"`
			InputTokens      int64   `json:"input_tokens"`
			OutputTokens     int64   `json:"output_tokens"`
			EstimatedCostUSD float64 `json:"estimated_cost_usd"`
		} `json:"points"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Points) != 4 {
		t.Fatalf("expected 4 hourly points, got %#v", response.Points)
	}
	if response.Points[0].BucketStart != "2026-05-16T00:00:00Z" || response.Points[0].Requests != 1 {
		t.Fatalf("unexpected first point %#v", response.Points[0])
	}
	if response.Points[1].Requests != 0 || response.Points[2].Requests != 0 || response.Points[3].Requests != 0 {
		t.Fatalf("expected empty buckets after the first one, got %#v", response.Points)
	}
	if !almostEqual(response.Points[0].EstimatedCostUSD, 0.625) {
		t.Fatalf("expected filtered model cost 0.625, got %#v", response.Points[0])
	}
}

func TestUsageAnalyticsProviderBreakdownAndRecentRequests(t *testing.T) {
	handler, repo := analyticsTestServer(t)
	seedAnalyticsRuns(t, repo,
		airequestlog.RunRecord{
			RequestID:           "req-3",
			Path:                "/v1/chat/completions",
			RequestedModel:      "unpriced-model",
			ProviderID:          "cx",
			ProviderName:        "Codex",
			FinalConnectionID:   "codex-1",
			FinalConnectionName: "codex-user",
			StatusCode:          500,
			PromptTokens:        9000,
			CompletionTokens:    1000,
			DurationMs:          441,
			CreatedAt:           mustParseRFC3339(t, "2026-05-16T03:00:00Z").UnixMilli(),
			FinalErrorCategory:  "upstream_error",
		},
		airequestlog.RunRecord{
			RequestID:           "req-2",
			Path:                "/v1/chat/completions",
			RequestedModel:      "opena/gpt-4.1",
			ProviderID:          "opena",
			ProviderName:        "OpenAI",
			FinalConnectionID:   "openai-1",
			FinalConnectionName: "openai-user",
			StatusCode:          200,
			PromptTokens:        200000,
			CompletionTokens:    100000,
			DurationMs:          900,
			CreatedAt:           mustParseRFC3339(t, "2026-05-16T02:00:00Z").UnixMilli(),
		},
		airequestlog.RunRecord{
			RequestID:           "req-1",
			Path:                "/v1/chat/completions",
			ResolvedModel:       "cx/gpt-5.4",
			ProviderID:          "cx",
			ProviderName:        "Codex",
			FinalConnectionID:   "codex-1",
			FinalConnectionName: "codex-user",
			StatusCode:          200,
			PromptTokens:        100000,
			CompletionTokens:    50000,
			DurationMs:          1200,
			CreatedAt:           mustParseRFC3339(t, "2026-05-16T01:00:00Z").UnixMilli(),
		},
	)

	breakdownReq := httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/provider-breakdown?from=2026-05-16T00:00:00Z&to=2026-05-17T00:00:00Z", nil)
	breakdownReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	breakdownRec := httptest.NewRecorder()
	handler.ServeHTTP(breakdownRec, breakdownReq)

	if breakdownRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, breakdownRec.Code, breakdownRec.Body.String())
	}

	var breakdownResponse struct {
		Items []struct {
			ProviderID       string  `json:"provider_id"`
			Requests         int64   `json:"requests"`
			RequestsPct      float64 `json:"requests_pct"`
			EstimatedCostUSD float64 `json:"estimated_cost_usd"`
		} `json:"items"`
	}
	if err := json.Unmarshal(breakdownRec.Body.Bytes(), &breakdownResponse); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(breakdownResponse.Items) != 2 {
		t.Fatalf("expected 2 provider groups, got %#v", breakdownResponse.Items)
	}
	if breakdownResponse.Items[0].ProviderID != "cx" || breakdownResponse.Items[0].Requests != 2 {
		t.Fatalf("expected cx to be first provider group, got %#v", breakdownResponse.Items)
	}
	if !almostEqual(breakdownResponse.Items[0].RequestsPct, 66.6666666667) {
		t.Fatalf("expected cx request share near 66.67, got %#v", breakdownResponse.Items[0])
	}
	if !almostEqual(breakdownResponse.Items[0].EstimatedCostUSD, 0.625) {
		t.Fatalf("expected unmatched model to contribute zero cost, got %#v", breakdownResponse.Items[0])
	}

	recentReq := httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/recent-requests?from=2026-05-16T00:00:00Z&to=2026-05-17T00:00:00Z&limit=2", nil)
	recentReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	recentRec := httptest.NewRecorder()
	handler.ServeHTTP(recentRec, recentReq)

	if recentRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, recentRec.Code, recentRec.Body.String())
	}

	var recentResponse struct {
		Items []struct {
			RequestID        string  `json:"request_id"`
			Model            string  `json:"model"`
			Status           string  `json:"status"`
			EstimatedCostUSD float64 `json:"estimated_cost_usd"`
		} `json:"items"`
		Page struct {
			Limit    int  `json:"limit"`
			Returned int  `json:"returned"`
			HasMore  bool `json:"has_more"`
		} `json:"page"`
	}
	if err := json.Unmarshal(recentRec.Body.Bytes(), &recentResponse); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if recentResponse.Page.Limit != 2 || recentResponse.Page.Returned != 2 || !recentResponse.Page.HasMore {
		t.Fatalf("unexpected page payload %#v", recentResponse.Page)
	}
	if recentResponse.Items[0].RequestID != "req-3" || recentResponse.Items[0].Status != "failed" {
		t.Fatalf("expected newest failed request first, got %#v", recentResponse.Items)
	}
	if recentResponse.Items[0].Model != "unpriced-model" || recentResponse.Items[0].EstimatedCostUSD != 0 {
		t.Fatalf("expected unmatched model to return zero cost, got %#v", recentResponse.Items[0])
	}
}

func TestAIRequestLogsListAndDetail(t *testing.T) {
	handler, repo := analyticsTestServer(t)
	run := airequestlog.RunRecord{
		RequestID:           "req-detail",
		Type:                "completions",
		RequestMode:         "stream",
		ProviderRequestMode: "stream",
		Method:              http.MethodPost,
		Path:                "/v1/chat/completions",
		RequestedModel:      "ignored",
		ResolvedModel:       "cx/gpt-5.4",
		ProviderID:          "cx",
		ProviderName:        "Codex",
		FinalConnectionID:   "codex-1",
		FinalConnectionName: "codex-user",
		AttemptCount:        2,
		StatusCode:          200,
		PromptTokens:        100000,
		CompletionTokens:    50000,
		TotalTokens:         150000,
		StartedAt:           mustParseRFC3339(t, "2026-05-16T01:00:00Z").UnixMilli(),
		CompletedAt:         mustParseRFC3339(t, "2026-05-16T01:00:02Z").UnixMilli(),
		DurationMs:          2000,
		CreatedAt:           mustParseRFC3339(t, "2026-05-16T01:00:00Z").UnixMilli(),
	}
	if err := repo.CreateAIRequestRun(&run); err != nil {
		t.Fatalf("seed detail run: %v", err)
	}
	if err := repo.CreateAIRequestFlow(airequestlog.FlowRecord{
		RunID:                 run.ID,
		RequestID:             run.RequestID,
		Type:                  run.Type,
		RequestMode:           run.RequestMode,
		ProviderRequestMode:   run.ProviderRequestMode,
		Method:                run.Method,
		Path:                  run.Path,
		RequestBody:           `{"model":"cx/gpt-5.4"}`,
		TranslatedRequestBody: `{"model":"gpt-5.4"}`,
		ResponseStatusCode:    200,
		ResponseBody:          `{"id":"chatcmpl-1"}`,
		AttemptTrace:          `[{"connection_id":"codex-1"}]`,
		StartedAt:             run.StartedAt,
		CompletedAt:           run.CompletedAt,
		DurationMs:            run.DurationMs,
		CreatedAt:             run.CreatedAt,
	}); err != nil {
		t.Fatalf("seed detail flow: %v", err)
	}
	for _, logRecord := range []airequestlog.ThirdPartyRequestLogRecord{
		{
			RunID:              run.ID,
			RequestID:          run.RequestID,
			Type:               run.Type,
			RequestMode:        run.RequestMode,
			ProviderID:         "cx",
			ProviderName:       "Codex",
			ConnectionID:       "codex-2",
			ConnectionName:     "fallback",
			AttemptIndex:       1,
			RequestMethod:      http.MethodPost,
			RequestURL:         "https://chatgpt.com/backend-api/codex",
			ResponseStatusCode: 200,
			ResponseBody:       `{"ok":true}`,
			CreatedAt:          run.CreatedAt + 10,
		},
		{
			RunID:              run.ID,
			RequestID:          run.RequestID,
			Type:               run.Type,
			RequestMode:        run.RequestMode,
			ProviderID:         "cx",
			ProviderName:       "Codex",
			ConnectionID:       "codex-1",
			ConnectionName:     "primary",
			AttemptIndex:       0,
			RequestMethod:      http.MethodPost,
			RequestURL:         "https://chatgpt.com/backend-api/codex",
			ResponseStatusCode: 429,
			ErrorType:          "upstream_retryable_error",
			ErrorMessage:       "rate limited",
			CreatedAt:          run.CreatedAt + 20,
		},
	} {
		if err := repo.CreateThirdPartyRequestLog(logRecord); err != nil {
			t.Fatalf("seed third party log: %v", err)
		}
	}
	if err := repo.CreateRTKRecord(&airequestlog.RTKRecord{
		RunID:        run.ID,
		RequestID:    run.RequestID,
		Applied:      true,
		BytesBefore:  1000,
		BytesAfter:   700,
		SavedBytes:   300,
		SavedPercent: 30,
		FilterChain:  "dedup-log",
		HitCount:     2,
		FieldCount:   1,
		CreatedAt:    run.CreatedAt,
	}); err != nil {
		t.Fatalf("seed rtk record: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/requests?limit=1&page=1", nil)
	listReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	var listResponse struct {
		Items []struct {
			RequestID        string  `json:"request_id"`
			Status           string  `json:"status"`
			EstimatedCostUSD float64 `json:"estimated_cost_usd"`
		} `json:"items"`
		Page struct {
			Page    int  `json:"page"`
			HasPrev bool `json:"has_prev"`
		} `json:"page"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResponse.Items) != 1 || listResponse.Items[0].RequestID != "req-detail" || listResponse.Items[0].Status != "completed" {
		t.Fatalf("unexpected list payload %#v", listResponse.Items)
	}
	if !almostEqual(listResponse.Items[0].EstimatedCostUSD, 0.625) {
		t.Fatalf("expected list cost 0.625, got %#v", listResponse.Items[0])
	}
	if listResponse.Page.Page != 1 || listResponse.Page.HasPrev {
		t.Fatalf("unexpected list page payload %#v", listResponse.Page)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/requests/req-detail", nil)
	detailReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	detailRec := httptest.NewRecorder()
	handler.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, detailRec.Code, detailRec.Body.String())
	}
	var detailResponse struct {
		Model            string  `json:"model"`
		Status           string  `json:"status"`
		EstimatedCostUSD float64 `json:"estimated_cost_usd"`
		Run              struct {
			RequestID string `json:"request_id"`
		} `json:"run"`
		Flow *struct {
			RequestBody string `json:"request_body"`
		} `json:"flow"`
		ThirdPartyLogs []struct {
			AttemptIndex int    `json:"attempt_index"`
			ConnectionID string `json:"connection_id"`
			ErrorMessage string `json:"error_message"`
			ResponseBody string `json:"response_body"`
		} `json:"third_party_logs"`
		RTK *struct {
			Applied bool `json:"applied"`
		} `json:"rtk"`
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detailResponse); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if detailResponse.Run.RequestID != "req-detail" || detailResponse.Flow == nil || detailResponse.Flow.RequestBody == "" || detailResponse.RTK == nil || !detailResponse.RTK.Applied {
		t.Fatalf("unexpected detail payload %#v", detailResponse)
	}
	if detailResponse.Model != "cx/gpt-5.4" || detailResponse.Status != "completed" || !almostEqual(detailResponse.EstimatedCostUSD, 0.625) {
		t.Fatalf("unexpected computed detail fields %#v", detailResponse)
	}
	if len(detailResponse.ThirdPartyLogs) != 2 || detailResponse.ThirdPartyLogs[0].AttemptIndex != 0 || detailResponse.ThirdPartyLogs[1].AttemptIndex != 1 {
		t.Fatalf("expected third party logs ordered by attempt, got %#v", detailResponse.ThirdPartyLogs)
	}
}

func TestUsageAnalyticsValidationAndAuth(t *testing.T) {
	handler, _ := analyticsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/summary?from=2026-05-16T00:00:00Z&to=2026-05-17T00:00:00Z", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/timeseries?from=2026-05-16T00:00:00Z&to=2026-05-16T00:00:00Z&bucket=month", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/recent-requests?from=2026-05-16T00:00:00Z&to=2027-06-17T00:00:00Z&limit=101", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/requests?from=2026-05-16T00:00:00Z&to=2026-05-17T00:00:00Z", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/api/analytics/usage/requests/missing-request", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestUsageAnalyticsAcceptsOffsetTimestamps(t *testing.T) {
	handler, repo := analyticsTestServer(t)
	seedAnalyticsRuns(t, repo, airequestlog.RunRecord{
		RequestID:           "req-offset",
		Path:                "/v1/chat/completions",
		ResolvedModel:       "cx/gpt-5.4",
		ProviderID:          "cx",
		ProviderName:        "Codex",
		FinalConnectionID:   "codex-1",
		FinalConnectionName: "codex-user",
		StatusCode:          200,
		PromptTokens:        1200,
		CompletionTokens:    300,
		DurationMs:          250,
		CreatedAt:           mustParseRFC3339(t, "2026-05-16T00:30:00Z").UnixMilli(),
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/api/analytics/usage/summary?from=2026-05-16T07:00:00%2B07:00&to=2026-05-16T08:00:00%2B07:00",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Requests struct {
			Value int64 `json:"value"`
		} `json:"requests"`
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Requests.Value != 1 {
		t.Fatalf("expected offset-based interval to match seeded run, got %#v", response)
	}
	if response.From != "2026-05-16T00:00:00Z" || response.To != "2026-05-16T01:00:00Z" {
		t.Fatalf("expected response interval to normalize to UTC, got %#v", response)
	}
}

func analyticsTestServer(t *testing.T) (http.Handler, *gormsqlite.Repository) {
	t.Helper()

	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	handler := testServerWithUsageAndConnectionAndWebUIAtPath(t, nil, &testProvider{}, nil, databasePath, testSettingsConfig())
	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	return handler, repo
}

func seedAnalyticsRuns(t *testing.T, repo *gormsqlite.Repository, runs ...airequestlog.RunRecord) {
	t.Helper()

	for _, run := range runs {
		run.Type = "completions"
		run.RequestMode = "sync"
		run.ProviderRequestMode = "sync"
		run.Method = http.MethodPost
		run.StartedAt = run.CreatedAt
		run.CompletedAt = run.CreatedAt + run.DurationMs
		if err := repo.CreateAIRequestRun(&run); err != nil {
			t.Fatalf("seed ai request run %q: %v", run.RequestID, err)
		}
	}
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}

	return parsed.UTC()
}

func almostEqual(a float64, b float64) bool {
	const tolerance = 0.000001
	if a > b {
		return a-b < tolerance
	}
	return b-a < tolerance
}

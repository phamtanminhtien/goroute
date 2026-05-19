package analytics

import (
	"math"
	"testing"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

type stubAnalyticsRepo struct {
	summary []SummaryAggregate
}

func (r stubAnalyticsRepo) AnalyticsSummary(Filters) ([]SummaryAggregate, error) {
	return r.summary, nil
}

func (r stubAnalyticsRepo) AnalyticsTimeseries(Filters, Bucket) ([]TimeseriesAggregate, error) {
	return nil, nil
}

func (r stubAnalyticsRepo) AnalyticsProviderBreakdown(Filters) ([]ProviderBreakdownAggregate, error) {
	return nil, nil
}

func (r stubAnalyticsRepo) AnalyticsRecentRequests(Filters, int) ([]RecentRequestAggregate, error) {
	return nil, nil
}

func (r stubAnalyticsRepo) AnalyticsRequestLogs(Filters, int, int) ([]RecentRequestAggregate, error) {
	return nil, nil
}

func (r stubAnalyticsRepo) AnalyticsRequestLogDetail(string) (RequestLogRecords, error) {
	return RequestLogRecords{}, nil
}

type stubCatalogSource struct {
	catalog provider.Catalog
}

func (s stubCatalogSource) Catalog() provider.Catalog {
	return s.catalog
}

type stubProviderModelRepo struct {
	records []provider.ModelRecord
}

func (r stubProviderModelRepo) ListProviderModels() ([]provider.ModelRecord, error) {
	return r.records, nil
}

func TestSummaryUsesCustomProviderModelPricing(t *testing.T) {
	service := NewServiceWithPricingSource(
		stubAnalyticsRepo{summary: []SummaryAggregate{{
			Requests:       1,
			InputTokens:    200000,
			OutputTokens:   100000,
			EffectiveModel: "custom/gpt-test",
		}}},
		stubCatalogSource{catalog: provider.Catalog{Providers: []provider.Provider{{
			ID:           "custom",
			Name:         "Custom",
			Category:     "custom",
			DefaultModel: "custom/gpt-test",
		}}}},
		stubProviderModelRepo{records: []provider.ModelRecord{{
			ID:                       "custom/gpt-test",
			ProviderID:               "custom",
			Name:                     "GPT Test",
			InputPricePerMillionUSD:  2,
			OutputPricePerMillionUSD: 8,
		}}},
	)

	result, err := service.Summary(Filters{
		From: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	expectedCost := 1.2
	if math.Abs(result.EstimatedCost.Value-expectedCost) > 0.000001 {
		t.Fatalf("expected custom model cost %.6f, got %.6f", expectedCost, result.EstimatedCost.Value)
	}
}

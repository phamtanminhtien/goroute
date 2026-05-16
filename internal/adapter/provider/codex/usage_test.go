package codex

import (
	"encoding/json"
	"testing"
)

func TestNormalizeUsageInfoMapsQuotaFamilies(t *testing.T) {
	var data usageResponse
	if err := json.Unmarshal([]byte(`{
		"plan_type": "plus",
		"rate_limit": {
			"limit_reached": false,
			"primary_window": {
				"used_percent": 42,
				"reset_at": 1747404000
			},
			"secondary_window": {
				"used_percent": 68,
				"reset_at": 1747922400000
			}
		},
		"code_review_rate_limit": {
			"limit_reached": false,
			"primary": {
				"used_percent": 15,
				"reset_at": "2026-05-16T10:00:00Z"
			},
			"secondary": {
				"used_percent": 27,
				"reset_at": "2026-05-22T10:00:00Z"
			}
		}
	}`), &data); err != nil {
		t.Fatalf("unmarshal usage response: %v", err)
	}

	usage := normalizeUsageInfo(data)

	if usage.Plan != "plus" {
		t.Fatalf("expected plus plan, got %#v", usage)
	}
	if usage.LimitReached || usage.ReviewLimitReached {
		t.Fatalf("expected limits not reached, got %#v", usage)
	}
	if usage.Quotas["session"].Used != 42 || usage.Quotas["session"].Remaining != 58 {
		t.Fatalf("expected normalized session quota, got %#v", usage.Quotas["session"])
	}
	if usage.Quotas["weekly"].ResetAt != "2025-05-22T14:00:00.000Z" {
		t.Fatalf("expected millisecond reset time, got %#v", usage.Quotas["weekly"])
	}
	if usage.Quotas["review_session"].ResetAt != "2026-05-16T10:00:00.000Z" {
		t.Fatalf("expected parsed review session reset, got %#v", usage.Quotas["review_session"])
	}
	if usage.Quotas["review_weekly"].Used != 27 || usage.Quotas["review_weekly"].Remaining != 73 {
		t.Fatalf("expected normalized review weekly quota, got %#v", usage.Quotas["review_weekly"])
	}
}

func TestNormalizeUsageInfoFindsReviewQuotaFromAdditionalRateLimits(t *testing.T) {
	var data usageResponse
	if err := json.Unmarshal([]byte(`{
		"summary": {
			"plan": "pro"
		},
		"rate_limits_by_limit_id": {
			"codex": {
				"limit_reached": true
			}
		},
		"additional_rate_limits": [
			{
				"id": "codex_review",
				"primary_window": {
					"used_percent": 80
				}
			}
		]
	}`), &data); err != nil {
		t.Fatalf("unmarshal usage response: %v", err)
	}

	usage := normalizeUsageInfo(data)

	if usage.Plan != "pro" {
		t.Fatalf("expected fallback plan, got %#v", usage)
	}
	if !usage.LimitReached {
		t.Fatalf("expected normal limit reached, got %#v", usage)
	}
	if usage.Quotas["review_session"].Used != 80 {
		t.Fatalf("expected review additional rate limit to be mapped, got %#v", usage.Quotas)
	}
}

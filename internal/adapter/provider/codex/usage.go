package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/providerregistry"
)

const usageBaseURL = "https://chatgpt.com/backend-api/wham/usage"

type usageResponse struct {
	PlanType             string                `json:"plan_type"`
	Summary              usageSummary          `json:"summary"`
	RateLimit            *usageRateLimitFamily `json:"rate_limit"`
	RateLimits           *usageRateLimitFamily `json:"rate_limits"`
	CodeReviewRateLimit  *usageRateLimitFamily `json:"code_review_rate_limit"`
	ReviewRateLimit      *usageRateLimitFamily `json:"review_rate_limit"`
	RateLimitsByLimitID  usageRateLimitByID    `json:"rate_limits_by_limit_id"`
	AdditionalRateLimits []identifiedRateLimit `json:"additional_rate_limits"`
}

type usageSummary struct {
	Plan string `json:"plan"`
}

type usageRateLimitByID struct {
	Codex       *usageRateLimitFamily `json:"codex"`
	CodeReview  *usageRateLimitFamily `json:"code_review"`
	CodexReview *usageRateLimitFamily `json:"codex_review"`
}

type identifiedRateLimit struct {
	ID string `json:"id"`
	usageRateLimitFamily
}

type usageRateLimitFamily struct {
	LimitReached    bool         `json:"limit_reached"`
	PrimaryWindow   *usageWindow `json:"primary_window"`
	SecondaryWindow *usageWindow `json:"secondary_window"`
	Primary         *usageWindow `json:"primary"`
	Secondary       *usageWindow `json:"secondary"`
}

type usageWindow struct {
	UsedPercent float64      `json:"used_percent"`
	ResetAt     usageResetAt `json:"reset_at"`
	Unlimited   bool         `json:"unlimited"`
}

type usageResetAt struct {
	ISO string
}

func (c *Client) Usage(ctx context.Context) (providerregistry.UsageInfo, error) {
	credential, err := c.resolveAccessToken(false)
	if err != nil {
		return providerregistry.UsageInfo{}, err
	}

	resp, err := c.doUsageRequest(ctx, credential)
	if err != nil {
		return providerregistry.UsageInfo{}, fmt.Errorf("execute codex usage request: %w", err)
	}

	if shouldRetryWithTokenRefresh(resp.StatusCode, c.connection) {
		resp.Body.Close()

		credential, err = c.resolveAccessToken(true)
		if err != nil {
			return providerregistry.UsageInfo{}, err
		}

		resp, err = c.doUsageRequest(ctx, credential)
		if err != nil {
			return providerregistry.UsageInfo{}, fmt.Errorf("execute codex usage retry request: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return providerregistry.UsageInfo{}, providerregistry.UsageUnavailableError{StatusCode: resp.StatusCode}
	}

	var payload usageResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return providerregistry.UsageInfo{}, fmt.Errorf("decode codex usage response: %w", err)
	}

	return normalizeUsageInfo(payload), nil
}

func (c *Client) doUsageRequest(ctx context.Context, credential string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build codex usage request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+credential)
	req.Header.Set("originator", "codex-cli")
	req.Header.Set("User-Agent", defaultUserAgent)

	return c.httpClient.Do(req)
}

func normalizeUsageInfo(data usageResponse) providerregistry.UsageInfo {
	info := providerregistry.UsageInfo{
		Plan:   defaultString(strings.TrimSpace(data.PlanType), strings.TrimSpace(data.Summary.Plan)),
		Quotas: map[string]providerregistry.UsageWindow{},
	}
	if info.Plan == "" {
		info.Plan = "unknown"
	}

	normalQuota := firstRateLimitFamily(
		data.RateLimit,
		data.RateLimits,
		data.RateLimitsByLimitID.Codex,
	)
	reviewQuota := firstRateLimitFamily(
		data.CodeReviewRateLimit,
		data.ReviewRateLimit,
		data.RateLimitsByLimitID.CodeReview,
		data.RateLimitsByLimitID.CodexReview,
		findReviewAdditionalRateLimit(data.AdditionalRateLimits),
	)

	if normalQuota != nil {
		info.LimitReached = normalQuota.LimitReached
		appendUsageWindow(info.Quotas, "session", firstUsageWindow(normalQuota.PrimaryWindow, normalQuota.Primary))
		appendUsageWindow(info.Quotas, "weekly", firstUsageWindow(normalQuota.SecondaryWindow, normalQuota.Secondary))
	}
	if reviewQuota != nil {
		info.ReviewLimitReached = reviewQuota.LimitReached
		appendUsageWindow(info.Quotas, "review_session", firstUsageWindow(reviewQuota.PrimaryWindow, reviewQuota.Primary))
		appendUsageWindow(info.Quotas, "review_weekly", firstUsageWindow(reviewQuota.SecondaryWindow, reviewQuota.Secondary))
	}
	if len(info.Quotas) == 0 {
		info.Quotas = nil
	}

	return info
}

func appendUsageWindow(quotas map[string]providerregistry.UsageWindow, key string, window *usageWindow) {
	normalized := normalizeUsageWindow(window)
	if normalized == nil {
		return
	}

	quotas[key] = *normalized
}

func normalizeUsageWindow(window *usageWindow) *providerregistry.UsageWindow {
	if window == nil {
		return nil
	}

	used := clampPercent(window.UsedPercent)
	total := 100
	remaining := 0
	if window.Unlimited {
		total = 0
	} else {
		remaining = total - used
	}

	return &providerregistry.UsageWindow{
		Used:      used,
		Total:     total,
		Remaining: remaining,
		ResetAt:   window.ResetAt.ISO,
		Unlimited: window.Unlimited,
	}
}

func (r *usageResetAt) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		return nil
	}

	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}

		r.ISO = parseResetAtString(raw)
		return nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}

	r.ISO = formatResetAtMillis(numberToMillis(parsed))
	return nil
}

func parseResetAtString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return formatResetAtMillis(numberToMillis(parsed))
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed.UTC().Format("2006-01-02T15:04:05.000Z")
		}
	}

	return ""
}

func formatResetAtMillis(ms int64) string {
	if ms <= 0 {
		return ""
	}

	return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
}

func numberToMillis(value float64) int64 {
	if value <= 0 {
		return 0
	}
	if value >= 1_000_000_000_000 {
		return int64(value)
	}

	return int64(value * 1000)
}

func clampPercent(value float64) int {
	switch {
	case value < 0:
		return 0
	case value > 100:
		return 100
	default:
		return int(value)
	}
}

func firstRateLimitFamily(families ...*usageRateLimitFamily) *usageRateLimitFamily {
	for _, family := range families {
		if family != nil {
			return family
		}
	}

	return nil
}

func firstUsageWindow(windows ...*usageWindow) *usageWindow {
	for _, window := range windows {
		if window != nil {
			return window
		}
	}

	return nil
}

func findReviewAdditionalRateLimit(items []identifiedRateLimit) *usageRateLimitFamily {
	for _, item := range items {
		if strings.Contains(strings.ToLower(strings.TrimSpace(item.ID)), "review") {
			family := item.usageRateLimitFamily
			return &family
		}
	}

	return nil
}

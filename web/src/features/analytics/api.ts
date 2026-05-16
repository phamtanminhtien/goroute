import { apiClient } from "@/shared/api/client";

export type UsageRange = "24h" | "7d" | "30d";
export type UsageBucket = "day" | "hour" | "week";

export const usageRangeOptions: Array<{ label: string; value: UsageRange }> = [
  { label: "24H", value: "24h" },
  { label: "7D", value: "7d" },
  { label: "30D", value: "30d" },
];

export const usageSummaryQueryKey = (range: UsageRange) =>
  ["analytics", "usage", "summary", range] as const;
export const usageTimeseriesQueryKey = (range: UsageRange) =>
  ["analytics", "usage", "timeseries", range] as const;
export const usageProviderBreakdownQueryKey = (range: UsageRange) =>
  ["analytics", "usage", "provider-breakdown", range] as const;
export const usageRecentRequestsQueryKey = (range: UsageRange) =>
  ["analytics", "usage", "recent-requests", range] as const;

export type UsageSummaryResponse = {
  estimated_cost_usd: {
    avg_per_request: number;
    delta_label?: string;
    delta_tone?: "critical" | "positive" | "warning";
    value: number;
  };
  filters: {
    connection_id: string | null;
    model: string | null;
    path: string | null;
    provider_id: string | null;
  };
  from: string;
  generated_at: string;
  input_tokens: {
    delta_label?: string;
    delta_tone?: "critical" | "positive" | "warning";
    value: number;
  };
  output_tokens: {
    delta_label?: string;
    delta_tone?: "critical" | "positive" | "warning";
    value: number;
  };
  requests: {
    delta_label?: string;
    delta_tone?: "critical" | "positive" | "warning";
    value: number;
  };
  to: string;
};

export type UsageTimeseriesResponse = {
  bucket: UsageBucket;
  filters: UsageSummaryResponse["filters"];
  from: string;
  generated_at: string;
  points: Array<{
    bucket_end: string;
    bucket_start: string;
    estimated_cost_usd: number;
    input_tokens: number;
    output_tokens: number;
    requests: number;
  }>;
  to: string;
};

export type UsageProviderBreakdownResponse = {
  filters: UsageSummaryResponse["filters"];
  from: string;
  generated_at: string;
  items: Array<{
    estimated_cost_usd: number;
    input_tokens: number;
    output_tokens: number;
    provider_id: string;
    provider_name: string;
    requests: number;
    requests_pct: number;
  }>;
  to: string;
};

export type UsageRecentRequestsResponse = {
  filters: UsageSummaryResponse["filters"];
  from: string;
  generated_at: string;
  items: Array<{
    connection_id: string;
    connection_name: string;
    estimated_cost_usd: number;
    input_tokens: number;
    latency_ms: number;
    model: string;
    output_tokens: number;
    path: string;
    provider_id: string;
    provider_name: string;
    request_id: string;
    status: "completed" | "failed";
    status_code: number;
    timestamp: string;
  }>;
  page: {
    has_more: boolean;
    limit: number;
    returned: number;
  };
  to: string;
};

export async function getUsageSummary(range: UsageRange) {
  const response = await apiClient.get<UsageSummaryResponse>(
    `/analytics/usage/summary${buildUsageQueryString(range)}`,
  );
  return response.data;
}

export async function getUsageTimeseries(range: UsageRange) {
  const preset = usageRangePreset(range);
  const response = await apiClient.get<UsageTimeseriesResponse>(
    `/analytics/usage/timeseries${buildUsageQueryString(range, {
      bucket: preset.bucket,
    })}`,
  );
  return response.data;
}

export async function getUsageProviderBreakdown(range: UsageRange) {
  const response = await apiClient.get<UsageProviderBreakdownResponse>(
    `/analytics/usage/provider-breakdown${buildUsageQueryString(range)}`,
  );
  return response.data;
}

export async function getUsageRecentRequests(range: UsageRange) {
  const response = await apiClient.get<UsageRecentRequestsResponse>(
    `/analytics/usage/recent-requests${buildUsageQueryString(range, {
      limit: "20",
    })}`,
  );
  return response.data;
}

export function usageRangePreset(range: UsageRange) {
  const now = new Date();

  switch (range) {
    case "24h":
      return {
        bucket: "hour" as const,
        from: new Date(now.getTime() - 24 * 60 * 60 * 1000),
        to: now,
      };
    case "7d":
      return {
        bucket: "day" as const,
        from: new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000),
        to: now,
      };
    case "30d":
      return {
        bucket: "week" as const,
        from: new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000),
        to: now,
      };
  }
}

function buildUsageQueryString(
  range: UsageRange,
  extras?: Record<string, string>,
) {
  const preset = usageRangePreset(range);
  const params = new URLSearchParams({
    from: formatUsageTimestamp(preset.from),
    to: formatUsageTimestamp(preset.to),
    ...extras,
  });

  return `?${params.toString()}`;
}

function formatUsageTimestamp(value: Date) {
  return new Date(Math.floor(value.getTime() / 1000) * 1000).toISOString();
}

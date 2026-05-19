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
export const aiRequestLogsQueryKey = (
  filters: AIRequestLogFilters,
  page: number,
) => ["analytics", "usage", "requests", filters, page] as const;
export const aiRequestLogDetailQueryKey = (requestID: string) =>
  ["analytics", "usage", "requests", requestID] as const;

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

export type AIRequestLogFilters = {
  connection_id?: string;
  model?: string;
  path?: string;
  provider_id?: string;
};

export type AIRequestLogsResponse = Omit<
  UsageRecentRequestsResponse,
  "from" | "to"
> & {
  page: UsageRecentRequestsResponse["page"] & {
    has_prev: boolean;
    page: number;
  };
};

export type AIRequestLogRun = {
  attempt_count: number;
  completed_at: number;
  completion_tokens: number;
  created_at: number;
  duration_ms: number;
  error_message: string;
  error_type: string;
  final_connection_id: string;
  final_connection_name: string;
  final_error_category: string;
  id: number;
  method: string;
  path: string;
  prompt_tokens: number;
  provider_id: string;
  provider_name: string;
  provider_request_mode: string;
  request_id: string;
  request_mode: string;
  requested_model: string;
  resolved_model: string;
  started_at: number;
  status_code: number;
  total_tokens: number;
  type: string;
  updated_at: number;
};

export type AIRequestLogFlow = {
  attempt_trace: string;
  completed_at: number;
  created_at: number;
  duration_ms: number;
  error_message: string;
  error_type: string;
  id: number;
  method: string;
  path: string;
  provider_id: string;
  provider_name: string;
  provider_request_mode: string;
  query: string;
  remote_addr: string;
  request_body: string;
  request_headers: string;
  request_id: string;
  request_mode: string;
  response_body: string | null;
  response_headers: string;
  response_status_code: number;
  run_id: number;
  started_at: number;
  translated_request_body: string;
  translated_response_body: string | null;
  type: string;
  updated_at: number;
  user_agent: string;
};

export type AIRequestLogThirdParty = {
  attempt_index: number;
  completed_at: number;
  connection_id: string;
  connection_name: string;
  created_at: number;
  duration_ms: number;
  error_message: string;
  error_type: string;
  id: number;
  provider_id: string;
  provider_name: string;
  provider_request_mode: string;
  request_body: string;
  request_headers: string;
  request_id: string;
  request_method: string;
  request_mode: string;
  request_url: string;
  response_body: string | null;
  response_headers: string;
  response_status_code: number;
  run_id: number;
  started_at: number;
  type: string;
  updated_at: number;
};

export type AIRequestLogRTK = {
  applied: boolean;
  bytes_after: number;
  bytes_before: number;
  created_at: number;
  field_count: number;
  filter_chain: string;
  hit_count: number;
  id: number;
  request_id: string;
  run_id: number;
  saved_bytes: number;
  saved_percent: number;
  updated_at: number;
};

export type AIRequestLogDetailResponse = {
  estimated_cost_usd: number;
  flow: AIRequestLogFlow | null;
  model: string;
  rtk: AIRequestLogRTK | null;
  run: AIRequestLogRun;
  status: "completed" | "failed";
  third_party_logs: AIRequestLogThirdParty[];
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

export async function getAIRequestLogs(filters: AIRequestLogFilters, page = 1) {
  const response = await apiClient.get<AIRequestLogsResponse>(
    `/analytics/usage/requests${buildAIRequestLogsQueryString({
      ...cleanAIRequestLogFilters(filters),
      limit: "50",
      page: String(page),
    })}`,
  );
  return response.data;
}

export async function getAIRequestLogDetail(requestID: string) {
  const response = await apiClient.get<AIRequestLogDetailResponse>(
    `/analytics/usage/requests/${encodeURIComponent(requestID)}`,
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

function cleanAIRequestLogFilters(filters: AIRequestLogFilters) {
  return Object.fromEntries(
    Object.entries(filters)
      .map(([key, value]) => [key, value?.trim() ?? ""])
      .filter(([, value]) => value !== ""),
  ) as Record<string, string>;
}

function buildAIRequestLogsQueryString(params: Record<string, string>) {
  const searchParams = new URLSearchParams(params);
  return `?${searchParams.toString()}`;
}

function formatUsageTimestamp(value: Date) {
  return new Date(Math.floor(value.getTime() / 1000) * 1000).toISOString();
}

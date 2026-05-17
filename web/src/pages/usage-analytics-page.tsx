import { useQuery } from "@tanstack/react-query";
import type { EChartsOption } from "echarts";
import {
  ArrowUpRight,
  ChartNoAxesColumn,
  Clock3,
  Coins,
  Database,
} from "lucide-react";
import { useState } from "react";

import { AnalyticsChart } from "@/features/analytics/analytics-chart";
import {
  getUsageProviderBreakdown,
  getUsageRecentRequests,
  getUsageSummary,
  getUsageTimeseries,
  usageProviderBreakdownQueryKey,
  type UsageRange,
  usageRangeOptions,
  usageRecentRequestsQueryKey,
  usageSummaryQueryKey,
  usageTimeseriesQueryKey,
} from "@/features/analytics/api";
import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { PageHeader } from "@/shared/ui/page-header";
import { Skeleton } from "@/shared/ui/skeleton";
import { SurfaceCard } from "@/shared/ui/surface-card";

const statusToneClasses: Record<"Completed" | "Failed", string> = {
  Completed:
    "border border-[color:color-mix(in_srgb,var(--success)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--success)_12%,transparent)] text-[var(--success)]",
  Failed:
    "border border-[color:color-mix(in_srgb,var(--error)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--error)_10%,transparent)] text-[var(--error)]",
};

export function UsageAnalyticsPage() {
  const [range, setRange] = useState<UsageRange>("24h");
  const summaryQuery = useQuery({
    queryFn: () => getUsageSummary(range),
    queryKey: usageSummaryQueryKey(range),
  });
  const timeseriesQuery = useQuery({
    queryFn: () => getUsageTimeseries(range),
    queryKey: usageTimeseriesQueryKey(range),
  });
  const breakdownQuery = useQuery({
    queryFn: () => getUsageProviderBreakdown(range),
    queryKey: usageProviderBreakdownQueryKey(range),
  });
  const recentRequestsQuery = useQuery({
    queryFn: () => getUsageRecentRequests(range),
    queryKey: usageRecentRequestsQueryKey(range),
  });

  const summary = summaryQuery.data;
  const timeseries = timeseriesQuery.data?.points ?? [];
  const breakdown = breakdownQuery.data?.items ?? [];
  const recentRequests = recentRequestsQuery.data?.items ?? [];

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="System-wide request volume, token flow, and estimated spend across the proxy from live admin analytics."
        eyebrow="Analytics"
        title="Usage & Analytics"
      >
        <div
          aria-label="Usage range"
          className="border-border/80 bg-bg-secondary/80 inline-flex rounded-[18px] border p-1 shadow-[var(--shadow-sm)]"
          role="tablist"
        >
          {usageRangeOptions.map((option) => {
            const active = option.value === range;

            return (
              <Button
                aria-selected={active}
                className={cn(
                  "min-h-10 rounded-[14px] border-0 px-4 py-2 text-xs tracking-[0.16em] uppercase shadow-none",
                  active
                    ? "bg-primary text-white hover:bg-[var(--primary-hover)]"
                    : "text-fg-secondary hover:text-fg-primary bg-transparent hover:bg-white/[0.04]",
                )}
                key={option.value}
                onClick={() => setRange(option.value)}
                ripple={false}
                role="tab"
                tone="ghost"
              >
                {option.label}
              </Button>
            );
          })}
        </div>
      </PageHeader>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          deltaLabel={summary?.requests.delta_label}
          deltaTone={summary?.requests.delta_tone}
          icon="requests"
          label="Total Requests"
          loading={summaryQuery.isPending}
          value={formatInteger(summary?.requests.value ?? 0)}
        />
        <MetricCard
          deltaLabel={summary?.input_tokens.delta_label}
          deltaTone={summary?.input_tokens.delta_tone}
          icon="input"
          label="Total Input Tokens"
          loading={summaryQuery.isPending}
          value={formatCompactNumber(summary?.input_tokens.value ?? 0)}
        />
        <MetricCard
          deltaLabel={summary?.output_tokens.delta_label}
          deltaTone={summary?.output_tokens.delta_tone}
          icon="output"
          label="Output Tokens"
          loading={summaryQuery.isPending}
          value={formatCompactNumber(summary?.output_tokens.value ?? 0)}
        />
        <MetricCard
          deltaLabel={summary?.estimated_cost_usd.delta_label}
          deltaTone={summary?.estimated_cost_usd.delta_tone}
          icon="cost"
          label="Est. Cost"
          loading={summaryQuery.isPending}
          value={formatCurrency(summary?.estimated_cost_usd.value ?? 0)}
        />
      </div>

      {summaryQuery.isError ? (
        <InlineAlert tone="error">
          {summaryQuery.error instanceof Error
            ? summaryQuery.error.message
            : "Summary analytics request failed."}
        </InlineAlert>
      ) : null}

      <div className="grid gap-3 xl:grid-cols-[1.5fr_1fr]">
        <ChartPanel
          description="Traffic volume across the selected window."
          error={timeseriesQuery.isError}
          loading={timeseriesQuery.isPending}
          option={buildRequestVolumeOption(timeseries, range)}
          title="Request Volume"
        />
        <ChartPanel
          description="Estimated provider share for the selected range by request volume."
          error={breakdownQuery.isError}
          loading={breakdownQuery.isPending}
          option={buildBreakdownOption(breakdown)}
          title="Provider Breakdown"
        />
      </div>

      <div className="grid gap-3 xl:grid-cols-[1.3fr_1fr]">
        <ChartPanel
          description="Input and output token movement over time."
          error={timeseriesQuery.isError}
          loading={timeseriesQuery.isPending}
          option={buildTokenFlowOption(timeseries, range)}
          title="Token Flow"
        />
        <ChartPanel
          description="Estimated spend trend from live admin analytics."
          error={timeseriesQuery.isError}
          loading={timeseriesQuery.isPending}
          option={buildCostOption(timeseries, range)}
          title="Cost Trend"
        />
      </div>

      <SurfaceCard className="overflow-hidden" tone="glass">
        <div className="border-border/70 flex flex-col gap-2 border-b px-5 py-4 sm:flex-row sm:items-end sm:justify-between">
          <div className="space-y-1">
            <p className="text-fg-primary text-lg font-semibold tracking-[-0.03em]">
              Recent Requests
            </p>
            <p className="text-fg-secondary text-sm leading-6">
              Latest matching proxy requests for the selected analytics range.
            </p>
          </div>
          <div className="border-border/70 text-fg-muted inline-flex items-center gap-2 rounded-full border bg-white/[0.02] px-3 py-1.5 text-[11px] font-semibold tracking-[0.18em] uppercase">
            <Clock3 className="size-3.5" />
            Latest activity
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full">
            <thead>
              <tr className="border-border/60 text-fg-muted border-b bg-white/[0.02] text-left text-[11px] font-semibold tracking-[0.18em] uppercase">
                <th className="px-5 py-3">Request</th>
                <th className="px-5 py-3">Route</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Latency</th>
                <th className="px-5 py-3">Input</th>
                <th className="px-5 py-3">Output</th>
                <th className="px-5 py-3">Est. Cost</th>
              </tr>
            </thead>
            <tbody>
              {recentRequestsQuery.isPending
                ? Array.from({ length: 5 }).map((_, index) => (
                    <RecentRequestRowSkeleton key={index} />
                  ))
                : recentRequests.map((request) => (
                    <RecentRequestRow
                      key={request.request_id}
                      request={request}
                    />
                  ))}
            </tbody>
          </table>
        </div>

        {recentRequestsQuery.isError ? (
          <div className="p-5">
            <InlineAlert tone="error">
              {recentRequestsQuery.error instanceof Error
                ? recentRequestsQuery.error.message
                : "Recent requests could not be loaded."}
            </InlineAlert>
          </div>
        ) : null}

        {!recentRequestsQuery.isPending &&
        !recentRequestsQuery.isError &&
        recentRequests.length === 0 ? (
          <div className="p-5">
            <InlineAlert>
              No matching requests were found for this range.
            </InlineAlert>
          </div>
        ) : null}
      </SurfaceCard>
    </section>
  );
}

function MetricCard({
  deltaLabel,
  deltaTone,
  icon,
  label,
  loading,
  value,
}: {
  deltaLabel?: string;
  deltaTone?: "critical" | "positive" | "warning";
  icon: "cost" | "input" | "output" | "requests";
  label: string;
  loading?: boolean;
  value: string;
}) {
  const metricIcon = getMetricIcon(icon);

  return (
    <SurfaceCard className="p-5" tone="glass">
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-3">
          <div className="space-y-1">
            <p className="text-fg-muted text-[11px] font-semibold tracking-[0.18em] uppercase">
              {label}
            </p>
            <div className="text-fg-primary text-3xl font-semibold tracking-[-0.04em]">
              {loading ? <Skeleton className="h-9 w-28" /> : value}
            </div>
          </div>
          {deltaLabel && deltaTone ? (
            <div
              className={cn(
                "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-semibold",
                metricDeltaClassName(deltaTone),
              )}
            >
              <ArrowUpRight className="size-3.5" />
              <span>{deltaLabel}</span>
            </div>
          ) : null}
        </div>

        <div className="dashboard-icon-surface text-fg-primary flex size-12 shrink-0 items-center justify-center rounded-[16px] border">
          {metricIcon}
        </div>
      </div>
    </SurfaceCard>
  );
}

function ChartPanel({
  description,
  error,
  loading,
  option,
  title,
}: {
  description: string;
  error?: boolean;
  loading?: boolean;
  option: EChartsOption;
  title: string;
}) {
  return (
    <SurfaceCard className="p-5" tone="glass">
      <div className="space-y-1">
        <p className="text-fg-primary text-lg font-semibold tracking-[-0.03em]">
          {title}
        </p>
        <p className="text-fg-secondary text-sm leading-6">{description}</p>
      </div>
      {loading ? <Skeleton className="mt-5 h-[300px] w-full" /> : null}
      {!loading && !error ? (
        <AnalyticsChart className="mt-5 h-[300px] w-full" option={option} />
      ) : null}
      {!loading && error ? (
        <InlineAlert className="mt-5" tone="error">
          Analytics chart data could not be loaded.
        </InlineAlert>
      ) : null}
    </SurfaceCard>
  );
}

function RecentRequestRow({
  request,
}: {
  request: {
    estimated_cost_usd: number;
    input_tokens: number;
    latency_ms: number;
    model: string;
    output_tokens: number;
    path: string;
    request_id: string;
    status: "completed" | "failed";
    timestamp: string;
  };
}) {
  const statusLabel = request.status === "completed" ? "Completed" : "Failed";

  return (
    <tr className="border-border/50 border-b text-sm last:border-b-0">
      <td className="px-5 py-4 align-top">
        <div className="space-y-1">
          <div className="text-fg-primary font-semibold">{request.model}</div>
          <div className="text-fg-secondary text-xs">
            {formatDateTime(request.timestamp)}
          </div>
        </div>
      </td>
      <td className="px-5 py-4 align-top">
        <div className="space-y-1">
          <div className="text-fg-primary font-medium">{request.path}</div>
          <div className="text-fg-secondary text-xs">{request.request_id}</div>
        </div>
      </td>
      <td className="px-5 py-4 align-top">
        <span
          className={cn(
            "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold",
            statusToneClasses[statusLabel],
          )}
        >
          {statusLabel}
        </span>
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {formatLatency(request.latency_ms)}
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {formatCompactNumber(request.input_tokens)}
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {formatCompactNumber(request.output_tokens)}
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {formatCurrency(request.estimated_cost_usd)}
      </td>
    </tr>
  );
}

function RecentRequestRowSkeleton() {
  return (
    <tr className="border-border/50 border-b text-sm last:border-b-0">
      <td className="px-5 py-4">
        <Skeleton className="h-10 w-32" />
      </td>
      <td className="px-5 py-4">
        <Skeleton className="h-10 w-40" />
      </td>
      <td className="px-5 py-4">
        <Skeleton className="h-6 w-20" />
      </td>
      <td className="px-5 py-4">
        <Skeleton className="h-6 w-16" />
      </td>
      <td className="px-5 py-4">
        <Skeleton className="h-6 w-16" />
      </td>
      <td className="px-5 py-4">
        <Skeleton className="h-6 w-16" />
      </td>
      <td className="px-5 py-4">
        <Skeleton className="h-6 w-16" />
      </td>
    </tr>
  );
}

function getMetricIcon(icon: "cost" | "input" | "output" | "requests") {
  switch (icon) {
    case "requests":
      return <ChartNoAxesColumn className="size-5" />;
    case "input":
      return <Database className="size-5" />;
    case "output":
      return <ArrowUpRight className="size-5" />;
    case "cost":
      return <Coins className="size-5" />;
  }
}

function metricDeltaClassName(tone: "critical" | "positive" | "warning") {
  switch (tone) {
    case "positive":
      return "border-[color:color-mix(in_srgb,var(--success)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--success)_12%,transparent)] text-[var(--success)]";
    case "warning":
      return "border-[color:color-mix(in_srgb,var(--warning)_26%,transparent)] bg-[color:color-mix(in_srgb,var(--warning)_12%,transparent)] text-[var(--warning)]";
    case "critical":
      return "border-[color:color-mix(in_srgb,var(--error)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--error)_10%,transparent)] text-[var(--error)]";
  }
}

function buildRequestVolumeOption(
  points: Array<{ bucket_start: string; requests: number }>,
  range: UsageRange,
): EChartsOption {
  return {
    animationDuration: 450,
    animationEasing: "cubicOut",
    grid: { bottom: 24, left: 12, right: 12, top: 18 },
    series: [
      {
        areaStyle: {
          color: "rgba(223, 122, 82, 0.18)",
        },
        data: points.map((point) => point.requests),
        itemStyle: { color: "#df7a52" },
        lineStyle: { color: "#df7a52", width: 3 },
        smooth: true,
        symbol: "circle",
        symbolSize: 8,
        type: "line",
      },
    ],
    textStyle: {
      fontFamily: "Inter, sans-serif",
    },
    tooltip: {
      backgroundColor: "rgba(23, 23, 23, 0.96)",
      borderColor: "rgba(255,255,255,0.08)",
      borderWidth: 1,
      textStyle: { color: "#f2f3f5" },
      trigger: "axis",
      valueFormatter: (value) => `${formatCompactNumber(Number(value))} reqs`,
    },
    xAxis: {
      axisLabel: { color: "#8e939d" },
      axisLine: { lineStyle: { color: "rgba(255,255,255,0.08)" } },
      axisTick: { show: false },
      boundaryGap: false,
      data: points.map((point) => formatBucketLabel(point.bucket_start, range)),
      type: "category",
    },
    yAxis: {
      axisLabel: {
        color: "#8e939d",
        formatter: (value: number) => formatCompactNumber(value),
      },
      splitLine: { lineStyle: { color: "rgba(255,255,255,0.06)" } },
      type: "value",
    },
  };
}

function buildTokenFlowOption(
  points: Array<{
    bucket_start: string;
    input_tokens: number;
    output_tokens: number;
  }>,
  range: UsageRange,
): EChartsOption {
  return {
    animationDuration: 450,
    grid: { bottom: 24, left: 12, right: 12, top: 18 },
    legend: {
      itemHeight: 8,
      itemWidth: 8,
      right: 0,
      textStyle: { color: "#b1b5bd" },
      top: 0,
    },
    series: [
      {
        barMaxWidth: 26,
        data: points.map((point) => point.input_tokens),
        itemStyle: {
          borderRadius: [8, 8, 0, 0],
          color: "#df7a52",
        },
        name: "Input",
        type: "bar",
      },
      {
        barMaxWidth: 26,
        data: points.map((point) => point.output_tokens),
        itemStyle: {
          borderRadius: [8, 8, 0, 0],
          color: "#9fd768",
        },
        name: "Output",
        type: "bar",
      },
    ],
    textStyle: {
      fontFamily: "Inter, sans-serif",
    },
    tooltip: {
      backgroundColor: "rgba(23, 23, 23, 0.96)",
      borderColor: "rgba(255,255,255,0.08)",
      borderWidth: 1,
      textStyle: { color: "#f2f3f5" },
      trigger: "axis",
      valueFormatter: (value) => formatCompactNumber(Number(value)),
    },
    xAxis: {
      axisLabel: { color: "#8e939d" },
      axisLine: { lineStyle: { color: "rgba(255,255,255,0.08)" } },
      axisTick: { show: false },
      data: points.map((point) => formatBucketLabel(point.bucket_start, range)),
      type: "category",
    },
    yAxis: {
      axisLabel: {
        color: "#8e939d",
        formatter: (value: number) => formatCompactNumber(value),
      },
      splitLine: { lineStyle: { color: "rgba(255,255,255,0.06)" } },
      type: "value",
    },
  };
}

function buildCostOption(
  points: Array<{ bucket_start: string; estimated_cost_usd: number }>,
  range: UsageRange,
): EChartsOption {
  return {
    animationDuration: 450,
    grid: { bottom: 24, left: 12, right: 12, top: 18 },
    series: [
      {
        data: points.map((point) => point.estimated_cost_usd),
        itemStyle: { color: "#9fd768" },
        lineStyle: { color: "#9fd768", width: 3 },
        smooth: true,
        symbol: "circle",
        symbolSize: 8,
        type: "line",
      },
    ],
    textStyle: {
      fontFamily: "Inter, sans-serif",
    },
    tooltip: {
      backgroundColor: "rgba(23, 23, 23, 0.96)",
      borderColor: "rgba(255,255,255,0.08)",
      borderWidth: 1,
      textStyle: { color: "#f2f3f5" },
      trigger: "axis",
      valueFormatter: (value) => `$${Number(value).toFixed(2)}`,
    },
    xAxis: {
      axisLabel: { color: "#8e939d" },
      axisLine: { lineStyle: { color: "rgba(255,255,255,0.08)" } },
      axisTick: { show: false },
      boundaryGap: false,
      data: points.map((point) => formatBucketLabel(point.bucket_start, range)),
      type: "category",
    },
    yAxis: {
      axisLabel: {
        color: "#8e939d",
        formatter: (value: number) => `$${value}`,
      },
      splitLine: { lineStyle: { color: "rgba(255,255,255,0.06)" } },
      type: "value",
    },
  };
}

function buildBreakdownOption(
  items: Array<{ provider_name: string; requests_pct: number }>,
): EChartsOption {
  return {
    animationDuration: 450,
    legend: {
      bottom: 0,
      itemHeight: 10,
      itemWidth: 10,
      textStyle: { color: "#b1b5bd" },
    },
    series: [
      {
        avoidLabelOverlap: true,
        center: ["50%", "44%"],
        data: items.map((item) => ({
          itemStyle: { color: breakdownColor(item.provider_name) },
          name: item.provider_name,
          value: item.requests_pct,
        })),
        label: {
          color: "#f2f3f5",
          formatter: "{d}%",
        },
        labelLine: { lineStyle: { color: "rgba(255,255,255,0.16)" } },
        radius: ["56%", "74%"],
        type: "pie",
      },
    ],
    textStyle: {
      fontFamily: "Inter, sans-serif",
    },
    tooltip: {
      backgroundColor: "rgba(23, 23, 23, 0.96)",
      borderColor: "rgba(255,255,255,0.08)",
      borderWidth: 1,
      textStyle: { color: "#f2f3f5" },
      trigger: "item",
    },
  };
}

function formatCompactNumber(value: number) {
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: value >= 1000000 ? 1 : 0,
    notation: "compact",
  }).format(value);
}

function formatInteger(value: number) {
  return new Intl.NumberFormat("en-US").format(value);
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat("en-US", {
    currency: "USD",
    maximumFractionDigits: value >= 1 ? 2 : 3,
    minimumFractionDigits: value >= 1 ? 2 : 0,
    style: "currency",
  }).format(value);
}

function formatLatency(value: number) {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}s`;
  }

  return `${value}ms`;
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    month: "short",
  }).format(new Date(value));
}

function formatBucketLabel(value: string, range: UsageRange) {
  const date = new Date(value);

  switch (range) {
    case "24h":
      return new Intl.DateTimeFormat("en-US", {
        hour: "2-digit",
        minute: "2-digit",
      }).format(date);
    case "7d":
      return new Intl.DateTimeFormat("en-US", {
        day: "numeric",
        month: "short",
      }).format(date);
    case "30d":
      return new Intl.DateTimeFormat("en-US", {
        day: "numeric",
        month: "short",
      }).format(date);
  }
}

function breakdownColor(value: string) {
  const palette = ["#df7a52", "#9fd768", "#56d364", "#f59e0b", "#63b3ed"];
  let hash = 0;
  for (const char of value) {
    hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  }

  return palette[hash % palette.length];
}

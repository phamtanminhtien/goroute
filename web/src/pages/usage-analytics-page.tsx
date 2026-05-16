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
  type RecentRequestItem,
  usageAnalyticsDatasets,
  type UsageBreakdownItem,
  type UsageMetricSnapshot,
  type UsageRange,
  usageRangeOptions,
  type UsageTimeseriesPoint,
} from "@/features/analytics/mock-usage-data";
import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import { PageHeader } from "@/shared/ui/page-header";
import { SurfaceCard } from "@/shared/ui/surface-card";

const statusToneClasses: Record<RecentRequestItem["status"], string> = {
  Cached:
    "border border-[color:color-mix(in_srgb,var(--primary)_22%,transparent)] bg-[color:color-mix(in_srgb,var(--primary)_12%,transparent)] text-primary",
  Completed:
    "border border-[color:color-mix(in_srgb,var(--success)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--success)_12%,transparent)] text-[var(--success)]",
  Failed:
    "border border-[color:color-mix(in_srgb,var(--error)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--error)_10%,transparent)] text-[var(--error)]",
  Streaming:
    "border border-[color:color-mix(in_srgb,var(--warning)_26%,transparent)] bg-[color:color-mix(in_srgb,var(--warning)_12%,transparent)] text-[var(--warning)]",
};

export function UsageAnalyticsPage() {
  const [range, setRange] = useState<UsageRange>("24h");
  const dataset = usageAnalyticsDatasets[range];

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="System-wide request volume, token flow, and estimated spend across the proxy. All numbers on this screen are mocked locally for design and UX work."
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
        {dataset.metrics.map((metric) => (
          <MetricCard key={metric.label} metric={metric} />
        ))}
      </div>

      <div className="grid gap-3 xl:grid-cols-[1.5fr_1fr]">
        <ChartPanel
          description="Traffic volume across the selected window."
          option={buildRequestVolumeOption(dataset.timeseries)}
          title="Request Volume"
        />
        <ChartPanel
          description="Estimated provider share for the selected range."
          option={buildBreakdownOption(dataset.breakdown)}
          title="Provider Breakdown"
        />
      </div>

      <div className="grid gap-3 xl:grid-cols-[1.3fr_1fr]">
        <ChartPanel
          description="Input and output token movement over time."
          option={buildTokenFlowOption(dataset.timeseries)}
          title="Token Flow"
        />
        <ChartPanel
          description="Estimated spend trend from mocked local analytics data."
          option={buildCostOption(dataset.timeseries)}
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
              Mock recent traffic samples for the selected analytics range.
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
              {dataset.recentRequests.map((request) => (
                <RecentRequestRow key={request.id} request={request} />
              ))}
            </tbody>
          </table>
        </div>
      </SurfaceCard>
    </section>
  );
}

function MetricCard({ metric }: { metric: UsageMetricSnapshot }) {
  const icon = getMetricIcon(metric.label);

  return (
    <SurfaceCard className="p-5" tone="glass">
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-3">
          <div className="space-y-1">
            <p className="text-fg-muted text-[11px] font-semibold tracking-[0.18em] uppercase">
              {metric.label}
            </p>
            <p className="text-fg-primary text-3xl font-semibold tracking-[-0.04em]">
              {metric.value}
            </p>
          </div>
          <div
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-semibold",
              metric.deltaTone === "positive" &&
                "border-[color:color-mix(in_srgb,var(--success)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--success)_12%,transparent)] text-[var(--success)]",
              metric.deltaTone === "warning" &&
                "border-[color:color-mix(in_srgb,var(--warning)_26%,transparent)] bg-[color:color-mix(in_srgb,var(--warning)_12%,transparent)] text-[var(--warning)]",
              metric.deltaTone === "critical" &&
                "border-[color:color-mix(in_srgb,var(--error)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--error)_10%,transparent)] text-[var(--error)]",
            )}
          >
            <ArrowUpRight className="size-3.5" />
            <span>{metric.deltaLabel}</span>
          </div>
        </div>

        <div className="dashboard-icon-surface text-fg-primary flex size-12 shrink-0 items-center justify-center rounded-[16px] border">
          {icon}
        </div>
      </div>
    </SurfaceCard>
  );
}

function ChartPanel({
  description,
  option,
  title,
}: {
  description: string;
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
      <AnalyticsChart className="mt-5 h-[300px] w-full" option={option} />
    </SurfaceCard>
  );
}

function RecentRequestRow({ request }: { request: RecentRequestItem }) {
  return (
    <tr className="border-border/50 border-b text-sm last:border-b-0">
      <td className="px-5 py-4 align-top">
        <div className="space-y-1">
          <div className="text-fg-primary font-semibold">{request.model}</div>
          <div className="text-fg-secondary text-xs">{request.timestamp}</div>
        </div>
      </td>
      <td className="px-5 py-4 align-top">
        <div className="space-y-1">
          <div className="text-fg-primary font-medium">{request.route}</div>
          <div className="text-fg-secondary text-xs">{request.id}</div>
        </div>
      </td>
      <td className="px-5 py-4 align-top">
        <span
          className={cn(
            "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold",
            statusToneClasses[request.status],
          )}
        >
          {request.status}
        </span>
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {request.latency}
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {request.inputTokens}
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {request.outputTokens}
      </td>
      <td className="text-fg-primary px-5 py-4 align-top font-medium">
        {request.estimatedCost}
      </td>
    </tr>
  );
}

function getMetricIcon(label: string) {
  switch (label) {
    case "Total Requests":
      return <ChartNoAxesColumn className="size-5" />;
    case "Total Input Tokens":
      return <Database className="size-5" />;
    case "Output Tokens":
      return <ArrowUpRight className="size-5" />;
    case "Est. Cost":
      return <Coins className="size-5" />;
    default:
      return <ChartNoAxesColumn className="size-5" />;
  }
}

function buildRequestVolumeOption(
  points: UsageTimeseriesPoint[],
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
      data: points.map((point) => point.label),
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

function buildTokenFlowOption(points: UsageTimeseriesPoint[]): EChartsOption {
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
        data: points.map((point) => point.inputTokens),
        itemStyle: {
          borderRadius: [8, 8, 0, 0],
          color: "#df7a52",
        },
        name: "Input",
        type: "bar",
      },
      {
        barMaxWidth: 26,
        data: points.map((point) => point.outputTokens),
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
      data: points.map((point) => point.label),
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

function buildCostOption(points: UsageTimeseriesPoint[]): EChartsOption {
  return {
    animationDuration: 450,
    grid: { bottom: 24, left: 12, right: 12, top: 18 },
    series: [
      {
        data: points.map((point) => point.cost),
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
      data: points.map((point) => point.label),
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

function buildBreakdownOption(items: UsageBreakdownItem[]): EChartsOption {
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
          itemStyle: { color: item.color },
          name: item.label,
          value: item.value,
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

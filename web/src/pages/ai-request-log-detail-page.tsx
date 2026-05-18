import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  Database,
  GitBranch,
  Timer,
  WalletCards,
} from "lucide-react";
import type { ReactNode } from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  aiRequestLogDetailQueryKey,
  type AIRequestLogDetailResponse,
  type AIRequestLogFlow,
  type AIRequestLogRTK,
  type AIRequestLogThirdParty,
  getAIRequestLogDetail,
} from "@/features/analytics/api";
import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import { DetailList, KeyValueRow } from "@/shared/ui/detail-list";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Skeleton } from "@/shared/ui/skeleton";
import { SurfaceCard } from "@/shared/ui/surface-card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";

const statusToneClasses: Record<"completed" | "failed", string> = {
  completed:
    "border border-[color:color-mix(in_srgb,var(--success)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--success)_12%,transparent)] text-[var(--success)]",
  failed:
    "border border-[color:color-mix(in_srgb,var(--error)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--error)_10%,transparent)] text-[var(--error)]",
};

export function AIRequestLogDetailPage() {
  const navigate = useNavigate();
  const { requestId = "" } = useParams<{ requestId: string }>();
  const detailQuery = useQuery({
    enabled: requestId.trim() !== "",
    queryFn: () => getAIRequestLogDetail(requestId),
    queryKey: aiRequestLogDetailQueryKey(requestId),
  });

  if (detailQuery.isPending) {
    return <AIRequestLogDetailSkeleton />;
  }

  if (detailQuery.isError) {
    return (
      <section className="space-y-4 pb-6">
        <Button
          leadingIcon={<ArrowLeft className="size-4" />}
          onClick={() => navigate("/ai-logs")}
          tone="secondary"
        >
          Back to AI Logs
        </Button>
        <InlineAlert tone="error">
          {detailQuery.error instanceof Error
            ? detailQuery.error.message
            : "AI request log could not be loaded."}
        </InlineAlert>
      </section>
    );
  }

  const detail = detailQuery.data;

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description={detail.run.path}
        eyebrow="AI Log Detail"
        title={detail.run.request_id}
      >
        <div className="flex flex-wrap items-center gap-2">
          <Button
            leadingIcon={<ArrowLeft className="size-4" />}
            onClick={() => navigate("/ai-logs")}
            tone="secondary"
          >
            Back
          </Button>
          <span
            className={cn(
              "inline-flex rounded-full px-2.5 py-1 text-xs font-semibold capitalize",
              statusToneClasses[detail.status],
            )}
          >
            {detail.status}
          </span>
        </div>
      </PageHeader>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          icon={<GitBranch className="size-5" />}
          label="Route"
          value={`${detail.run.provider_name || detail.run.provider_id} / ${
            detail.run.final_connection_name || detail.run.final_connection_id
          }`}
        />
        <MetricCard
          icon={<Timer className="size-5" />}
          label="Latency"
          value={formatLatency(detail.run.duration_ms)}
        />
        <MetricCard
          icon={<Database className="size-5" />}
          label="Tokens"
          value={formatInteger(detail.run.total_tokens)}
        />
        <MetricCard
          icon={<WalletCards className="size-5" />}
          label="Est. Cost"
          value={formatCurrency(detail.estimated_cost_usd)}
        />
      </div>

      <Tabs defaultValue="overview">
        <TabsList className="flex flex-wrap">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="flow">Flow</TabsTrigger>
          <TabsTrigger value="upstream">Upstream Attempts</TabsTrigger>
          <TabsTrigger value="rtk">RTK</TabsTrigger>
          <TabsTrigger value="raw">Raw</TabsTrigger>
        </TabsList>

        <TabsContent className="mt-4" value="overview">
          <OverviewTab detail={detail} />
        </TabsContent>
        <TabsContent className="mt-4" value="flow">
          <FlowTab flow={detail.flow} />
        </TabsContent>
        <TabsContent className="mt-4" value="upstream">
          <UpstreamTab logs={detail.third_party_logs} />
        </TabsContent>
        <TabsContent className="mt-4" value="rtk">
          <RTKTab rtk={detail.rtk} />
        </TabsContent>
        <TabsContent className="mt-4" value="raw">
          <DebugBlock label="Full detail payload" value={detail} />
        </TabsContent>
      </Tabs>
    </section>
  );
}

function OverviewTab({ detail }: { detail: AIRequestLogDetailResponse }) {
  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <SectionCard title="Request">
        <DetailList>
          <KeyValueRow label="Request ID" value={detail.run.request_id} />
          <KeyValueRow label="Type" value={detail.run.type || "-"} />
          <KeyValueRow label="Mode" value={detail.run.request_mode || "-"} />
          <KeyValueRow
            label="Provider mode"
            value={detail.run.provider_request_mode || "-"}
          />
          <KeyValueRow label="Method" value={detail.run.method || "-"} />
          <KeyValueRow label="Path" value={detail.run.path || "-"} />
          <KeyValueRow label="Model" value={detail.model || "-"} />
        </DetailList>
      </SectionCard>

      <SectionCard title="Outcome">
        <DetailList>
          <KeyValueRow label="Status code" value={detail.run.status_code} />
          <KeyValueRow
            label="Started"
            value={formatEpochMillis(detail.run.started_at)}
          />
          <KeyValueRow
            label="Completed"
            value={formatEpochMillis(detail.run.completed_at)}
          />
          <KeyValueRow
            label="Attempts"
            value={formatInteger(detail.run.attempt_count)}
          />
          <KeyValueRow
            label="Error type"
            value={detail.run.error_type || "-"}
          />
          <KeyValueRow
            label="Error message"
            value={detail.run.error_message || "-"}
          />
          <KeyValueRow
            label="Final category"
            value={detail.run.final_error_category || "-"}
          />
        </DetailList>
      </SectionCard>
    </div>
  );
}

function FlowTab({ flow }: { flow: AIRequestLogFlow | null }) {
  if (!flow) {
    return <InlineAlert>No persisted flow record is available.</InlineAlert>;
  }

  return (
    <div className="space-y-4">
      <SectionCard title="Flow Metadata">
        <DetailList>
          <KeyValueRow label="Remote address" value={flow.remote_addr || "-"} />
          <KeyValueRow label="User agent" value={flow.user_agent || "-"} />
          <KeyValueRow label="Query" value={flow.query || "-"} />
          <KeyValueRow
            label="Response status"
            value={flow.response_status_code}
          />
          <KeyValueRow
            label="Duration"
            value={formatLatency(flow.duration_ms)}
          />
        </DetailList>
      </SectionCard>

      <div className="grid gap-4 xl:grid-cols-2">
        <DebugBlock label="Request headers" value={flow.request_headers} />
        <DebugBlock label="Response headers" value={flow.response_headers} />
        <DebugBlock label="Request body" value={flow.request_body} />
        <DebugBlock
          label="Translated request body"
          value={flow.translated_request_body}
        />
        <DebugBlock label="Response body" value={flow.response_body} />
        <DebugBlock
          label="Translated response body"
          value={flow.translated_response_body}
        />
      </div>

      <DebugBlock label="Attempt trace" value={flow.attempt_trace} />
    </div>
  );
}

function UpstreamTab({ logs }: { logs: AIRequestLogThirdParty[] }) {
  if (logs.length === 0) {
    return (
      <InlineAlert>
        No persisted third-party upstream logs are available.
      </InlineAlert>
    );
  }

  return (
    <div className="space-y-4">
      {logs.map((log) => (
        <SectionCard
          description={`${log.request_method || "-"} ${log.request_url || "-"}`}
          key={log.id}
          title={`Attempt ${log.attempt_index}`}
        >
          <DetailList>
            <KeyValueRow
              label="Connection"
              value={log.connection_name || log.connection_id || "-"}
            />
            <KeyValueRow
              label="Status code"
              value={log.response_status_code || "-"}
            />
            <KeyValueRow
              label="Duration"
              value={formatLatency(log.duration_ms)}
            />
            <KeyValueRow label="Error type" value={log.error_type || "-"} />
            <KeyValueRow
              label="Error message"
              value={log.error_message || "-"}
            />
          </DetailList>
          <div className="mt-4 grid gap-4 xl:grid-cols-2">
            <DebugBlock label="Request headers" value={log.request_headers} />
            <DebugBlock label="Request body" value={log.request_body} />
            <DebugBlock label="Response headers" value={log.response_headers} />
            <DebugBlock label="Response body" value={log.response_body} />
          </div>
        </SectionCard>
      ))}
    </div>
  );
}

function RTKTab({ rtk }: { rtk: AIRequestLogRTK | null }) {
  if (!rtk) {
    return (
      <InlineAlert>No persisted RTK diagnostics are available.</InlineAlert>
    );
  }

  return (
    <SectionCard title="RTK Diagnostics">
      <DetailList>
        <KeyValueRow label="Applied" value={rtk.applied ? "Yes" : "No"} />
        <KeyValueRow
          label="Bytes before"
          value={formatInteger(rtk.bytes_before)}
        />
        <KeyValueRow
          label="Bytes after"
          value={formatInteger(rtk.bytes_after)}
        />
        <KeyValueRow
          label="Saved bytes"
          value={formatInteger(rtk.saved_bytes)}
        />
        <KeyValueRow label="Saved percent" value={`${rtk.saved_percent}%`} />
        <KeyValueRow label="Filter chain" value={rtk.filter_chain || "-"} />
        <KeyValueRow label="Hits" value={formatInteger(rtk.hit_count)} />
        <KeyValueRow label="Fields" value={formatInteger(rtk.field_count)} />
      </DetailList>
    </SectionCard>
  );
}

function MetricCard({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <SurfaceCard className="p-5" tone="glass">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <p className="text-fg-muted text-[11px] font-semibold tracking-[0.18em] uppercase">
            {label}
          </p>
          <p className="text-fg-primary truncate text-2xl font-semibold tracking-[-0.03em]">
            {value}
          </p>
        </div>
        <div className="dashboard-icon-surface text-fg-primary flex size-12 shrink-0 items-center justify-center rounded-[16px] border">
          {icon}
        </div>
      </div>
    </SurfaceCard>
  );
}

function DebugBlock({ label, value }: { label: string; value: unknown }) {
  return (
    <SectionCard title={label}>
      <pre className="border-border/70 max-h-[520px] overflow-auto rounded-[18px] border bg-[#111111] p-4 text-xs leading-5 whitespace-pre-wrap text-slate-200">
        {formatDebugValue(value)}
      </pre>
    </SectionCard>
  );
}

function AIRequestLogDetailSkeleton() {
  return (
    <section className="space-y-6 pb-6">
      <Skeleton className="h-20 w-full" />
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton className="h-32 w-full" key={index} />
        ))}
      </div>
      <Skeleton className="h-[420px] w-full" />
    </section>
  );
}

function formatDebugValue(value: unknown) {
  if (value === null || value === undefined || value === "") {
    return "No data";
  }

  if (typeof value !== "string") {
    return JSON.stringify(value, null, 2);
  }

  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

function formatEpochMillis(value: number) {
  if (!value) {
    return "-";
  }

  return new Date(value).toLocaleString();
}

function formatLatency(value: number) {
  if (!value) {
    return "-";
  }
  if (value < 1000) {
    return `${value}ms`;
  }

  return `${(value / 1000).toFixed(2)}s`;
}

function formatInteger(value: number) {
  return new Intl.NumberFormat().format(value);
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat(undefined, {
    currency: "USD",
    maximumFractionDigits: value >= 1 ? 2 : 4,
    style: "currency",
  }).format(value);
}

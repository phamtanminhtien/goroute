import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router-dom";

import {
  type AIRequestLogFilters,
  aiRequestLogsQueryKey,
  getAIRequestLogs,
} from "@/features/analytics/api";
import { listProviders, providersQueryKey } from "@/features/providers/api";
import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { PageHeader } from "@/shared/ui/page-header";
import { Select } from "@/shared/ui/select";
import { Skeleton } from "@/shared/ui/skeleton";
import { SurfaceCard } from "@/shared/ui/surface-card";

const statusToneClasses: Record<"completed" | "failed", string> = {
  completed:
    "border border-[color:color-mix(in_srgb,var(--success)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--success)_12%,transparent)] text-[var(--success)]",
  failed:
    "border border-[color:color-mix(in_srgb,var(--error)_24%,transparent)] bg-[color:color-mix(in_srgb,var(--error)_10%,transparent)] text-[var(--error)]",
};

const allFilterValue = "__all__";
const pathOptions = [
  { label: "All paths", value: allFilterValue },
  { label: "/v1/chat/completions", value: "/v1/chat/completions" },
  { label: "/v1/responses", value: "/v1/responses" },
];

type SelectFilters = {
  connection_id: string;
  model: string;
  path: string;
  provider_id: string;
};

export function AIRequestLogsPage() {
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<SelectFilters>({
    connection_id: allFilterValue,
    model: allFilterValue,
    path: allFilterValue,
    provider_id: allFilterValue,
  });
  const apiFilters = useMemo(
    () => buildAPIRequestLogFilters(filters),
    [filters],
  );
  const logsQuery = useQuery({
    queryFn: () => getAIRequestLogs(apiFilters, page),
    queryKey: aiRequestLogsQueryKey(apiFilters, page),
  });
  const providersQuery = useQuery({
    queryFn: listProviders,
    queryKey: providersQueryKey,
  });
  const providers = providersQuery.data ?? [];
  const selectedProvider =
    filters.provider_id === allFilterValue
      ? null
      : providers.find((provider) => provider.id === filters.provider_id);
  const providerOptions = [
    { label: "All providers", value: allFilterValue },
    ...providers.map((provider) => ({
      label: provider.name,
      value: provider.id,
    })),
  ];
  const connectionOptions = [
    { label: "All connections", value: allFilterValue },
    ...(selectedProvider
      ? selectedProvider.connections
      : providers.flatMap((provider) => provider.connections)
    ).map((connection) => ({
      label: connection.name,
      value: connection.id,
    })),
  ];
  const modelOptions = [
    { label: "All models", value: allFilterValue },
    ...(selectedProvider
      ? selectedProvider.models
      : providers.flatMap((provider) => provider.models)
    ).map((model) => ({
      label: model.id,
      value: model.id,
    })),
  ];

  function updateFilter(key: keyof SelectFilters, value: string) {
    setPage(1);
    setFilters((current) => {
      const next = { ...current, [key]: value };
      if (key === "provider_id") {
        next.connection_id = allFilterValue;
        next.model = allFilterValue;
      }
      return next;
    });
  }

  const items = logsQuery.data?.items ?? [];
  const pageInfo = logsQuery.data?.page;

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Persisted AI request diagnostics with request, routing, upstream, and runtime logging data."
        eyebrow="Observability"
        title="AI Logs"
      />

      <SurfaceCard className="p-5" tone="glass">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <Field label="Provider">
            <Select
              disabled={providersQuery.isPending}
              onValueChange={(value) => updateFilter("provider_id", value)}
              options={providerOptions}
              value={filters.provider_id}
            />
          </Field>
          <Field label="Connection">
            <Select
              disabled={providersQuery.isPending}
              onValueChange={(value) => updateFilter("connection_id", value)}
              options={connectionOptions}
              value={filters.connection_id}
            />
          </Field>
          <Field label="Model">
            <Select
              disabled={providersQuery.isPending}
              onValueChange={(value) => updateFilter("model", value)}
              options={modelOptions}
              value={filters.model}
            />
          </Field>
          <Field label="Path">
            <Select
              onValueChange={(value) => updateFilter("path", value)}
              options={pathOptions}
              value={filters.path}
            />
          </Field>
        </div>
      </SurfaceCard>

      <SurfaceCard className="overflow-hidden" tone="glass">
        <div className="border-border/70 flex flex-col gap-2 border-b px-5 py-4 sm:flex-row sm:items-end sm:justify-between">
          <div className="space-y-1">
            <p className="text-fg-primary text-lg font-semibold tracking-[-0.03em]">
              Requests
            </p>
            <p className="text-fg-secondary text-sm leading-6">
              Latest matching persisted AI request runs.
            </p>
          </div>
          <div className="border-border/70 text-fg-muted inline-flex items-center gap-2 rounded-full border bg-white/[0.02] px-3 py-1.5 text-[11px] font-semibold tracking-[0.18em] uppercase">
            <Search className="size-3.5" />
            Page {pageInfo?.page ?? page}
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
                <th className="px-5 py-3">Tokens</th>
                <th className="px-5 py-3">Est. Cost</th>
              </tr>
            </thead>
            <tbody>
              {logsQuery.isPending
                ? Array.from({ length: 6 }).map((_, index) => (
                    <AIRequestLogRowSkeleton key={index} />
                  ))
                : items.map((request) => (
                    <tr
                      className="border-border/50 border-b text-sm last:border-b-0"
                      key={request.request_id}
                    >
                      <td className="px-5 py-4 align-top">
                        <div className="space-y-1">
                          <Link
                            className="text-fg-primary hover:text-primary font-semibold"
                            to={`/ai-logs/${encodeURIComponent(
                              request.request_id,
                            )}`}
                          >
                            {request.model || request.request_id}
                          </Link>
                          <div className="text-fg-secondary text-xs">
                            {formatDateTime(request.timestamp)}
                          </div>
                          <div className="text-fg-muted font-mono text-[11px]">
                            {request.request_id}
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4 align-top">
                        <div className="space-y-1">
                          <div className="text-fg-primary font-medium">
                            {request.path || "-"}
                          </div>
                          <div className="text-fg-secondary text-xs">
                            {request.provider_name || request.provider_id} /{" "}
                            {request.connection_name || request.connection_id}
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4 align-top">
                        <span
                          className={cn(
                            "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold capitalize",
                            statusToneClasses[request.status],
                          )}
                        >
                          {request.status}
                        </span>
                      </td>
                      <td className="text-fg-primary px-5 py-4 align-top font-medium">
                        {formatLatency(request.latency_ms)}
                      </td>
                      <td className="text-fg-primary px-5 py-4 align-top font-medium">
                        {formatCompactNumber(
                          request.input_tokens + request.output_tokens,
                        )}
                      </td>
                      <td className="text-fg-primary px-5 py-4 align-top font-medium">
                        {formatCurrency(request.estimated_cost_usd)}
                      </td>
                    </tr>
                  ))}
            </tbody>
          </table>
        </div>

        {logsQuery.isError ? (
          <div className="p-5">
            <InlineAlert tone="error">
              {logsQuery.error instanceof Error
                ? logsQuery.error.message
                : "AI request logs could not be loaded."}
            </InlineAlert>
          </div>
        ) : null}

        {!logsQuery.isPending && !logsQuery.isError && items.length === 0 ? (
          <div className="p-5">
            <InlineAlert>No matching AI request logs were found.</InlineAlert>
          </div>
        ) : null}

        <div className="border-border/70 flex flex-col gap-3 border-t px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-fg-secondary text-sm">
            {pageInfo?.returned ?? 0} requests on page {pageInfo?.page ?? page}
          </p>
          <div className="flex items-center gap-2">
            <Button
              disabled={
                logsQuery.isFetching || !(pageInfo?.has_prev ?? page > 1)
              }
              leadingIcon={<ChevronLeft className="size-4" />}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              tone="secondary"
            >
              Previous
            </Button>
            <Button
              disabled={logsQuery.isFetching || !(pageInfo?.has_more ?? false)}
              leadingIcon={<ChevronRight className="size-4" />}
              onClick={() => setPage((current) => current + 1)}
              tone="secondary"
            >
              Next
            </Button>
          </div>
        </div>
      </SurfaceCard>
    </section>
  );
}

function buildAPIRequestLogFilters(
  filters: SelectFilters,
): AIRequestLogFilters {
  return {
    connection_id:
      filters.connection_id === allFilterValue
        ? undefined
        : filters.connection_id,
    model: filters.model === allFilterValue ? undefined : filters.model,
    path: filters.path === allFilterValue ? undefined : filters.path,
    provider_id:
      filters.provider_id === allFilterValue ? undefined : filters.provider_id,
  };
}

function AIRequestLogRowSkeleton() {
  return (
    <tr className="border-border/50 border-b text-sm last:border-b-0">
      {Array.from({ length: 6 }).map((_, index) => (
        <td className="px-5 py-4" key={index}>
          <Skeleton className="h-10 w-32" />
        </td>
      ))}
    </tr>
  );
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString([], {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "short",
  });
}

function formatLatency(value: number) {
  if (value < 1000) {
    return `${value}ms`;
  }

  return `${(value / 1000).toFixed(2)}s`;
}

function formatCompactNumber(value: number) {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1,
    notation: "compact",
  }).format(value);
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat(undefined, {
    currency: "USD",
    maximumFractionDigits: value >= 1 ? 2 : 4,
    style: "currency",
  }).format(value);
}

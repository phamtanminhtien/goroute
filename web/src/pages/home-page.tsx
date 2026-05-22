import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link2, Plus, Save, SlidersHorizontal, X } from "lucide-react";
import type { FormEvent } from "react";
import { useMemo, useState } from "react";

import {
  getSettings,
  settingsQueryKey,
  updateSettings,
} from "@/features/settings/api";
import {
  createSystemAPIKey,
  listSystemAPIKeys,
  type SystemAPIKey,
  systemAPIKeysQueryKey,
  updateSystemAPIKey,
} from "@/features/system-api-keys/api";
import { Button, CopyButton } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { Input } from "@/shared/ui/input";
import {
  Modal,
  ModalBody,
  ModalClose,
  ModalContent,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "@/shared/ui/modal";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/shared/ui/status-badge";
import { SurfaceCard } from "@/shared/ui/surface-card";
import { Switch } from "@/shared/ui/switch";

const openAIEndpoints = [
  { label: "Models", value: "GET /v1/models" },
  { label: "Chat Completions", value: "POST /v1/chat/completions" },
  { label: "Responses", value: "POST /v1/responses" },
];

const anthropicEndpoints = [
  { label: "Anthropic Messages", value: "POST /v1/messages" },
];

export function HomePage() {
  const queryClient = useQueryClient();
  const settingsQuery = useQuery({
    queryFn: getSettings,
    queryKey: settingsQueryKey,
  });
  const keysQuery = useQuery({
    queryFn: listSystemAPIKeys,
    queryKey: systemAPIKeysQueryKey,
  });
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [draftName, setDraftName] = useState("");
  const [formError, setFormError] = useState("");

  const baseURL = useMemo(
    () => resolveOpenAIBaseURL(settingsQuery.data?.server.listen),
    [settingsQuery.data?.server.listen],
  );

  const createKeyMutation = useMutation({
    mutationFn: createSystemAPIKey,
    onError: (error) => {
      setFormError(error instanceof Error ? error.message : "Request failed");
    },
    onSuccess: (apiKey) => {
      queryClient.setQueryData<SystemAPIKey[]>(
        systemAPIKeysQueryKey,
        (current = []) => [apiKey, ...current],
      );
      setDraftName("");
      setFormError("");
      setIsAddOpen(false);
    },
  });

  const updateKeyMutation = useMutation({
    mutationFn: ({
      daily_token_limit,
      enabled,
      id,
      monthly_token_limit,
      name,
      requests_per_minute_limit,
    }: {
      daily_token_limit?: number | null;
      enabled?: boolean;
      id: string;
      monthly_token_limit?: number | null;
      name?: string;
      requests_per_minute_limit?: number | null;
    }) =>
      updateSystemAPIKey(id, {
        daily_token_limit,
        enabled,
        monthly_token_limit,
        name,
        requests_per_minute_limit,
      }),
    onSuccess: (updated) => {
      queryClient.setQueryData<SystemAPIKey[]>(
        systemAPIKeysQueryKey,
        (current = []) =>
          current.map((item) => (item.id === updated.id ? updated : item)),
      );
    },
  });

  const updateOpenAIAuthMutation = useMutation({
    mutationFn: async (enabled: boolean) => {
      const current = settingsQuery.data;
      if (!current) {
        throw new Error("Settings are not loaded yet.");
      }

      return updateSettings({
        llmLogging: current.llmLogging,
        openAICompatibleAuth: { enabled },
        rtk: current.rtk,
      });
    },
    onSuccess: (next) => {
      queryClient.setQueryData(settingsQueryKey, next);
    },
  });

  async function handleAddKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = draftName.trim();

    if (!name) {
      setFormError("Name is required.");
      return;
    }

    await createKeyMutation.mutateAsync({ name });
  }

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Connect OpenAI-compatible and Anthropic-compatible clients to GoRoute, manage system API keys, and keep routing policy inside the proxy."
        eyebrow="Home"
        title="AI connection hub"
      >
        <StatusBadge tone="info">Multi-protocol</StatusBadge>
      </PageHeader>

      <SectionCard
        description="Choose the protocol your client already speaks. Both use the same GoRoute runtime URL and system API keys."
        title="Client connections"
        tone="solid"
      >
        {settingsQuery.isPending ? (
          <div className="grid gap-4 lg:grid-cols-2">
            <Skeleton className="min-h-[420px]" />
            <Skeleton className="min-h-[420px]" />
          </div>
        ) : settingsQuery.isError || !baseURL ? (
          <InlineAlert tone="error">
            Cannot resolve server port from runtime settings.
          </InlineAlert>
        ) : (
          <div className="grid gap-4 lg:grid-cols-2">
            <ProtocolPanel
              description="Use Chat Completions or Responses from OpenAI-compatible SDKs and tools."
              endpoints={openAIEndpoints}
              example={`const client = new OpenAI({
  baseURL: "${baseURL}",
  apiKey: "<system-api-key>",
});

await client.chat.completions.create({
  model: "cx/gpt-5.4",
  messages: [{ role: "user", content: "Hello from GoRoute" }],
});`}
              title="OpenAI-compatible"
              value={baseURL}
            />
            <ProtocolPanel
              description="Use Anthropic Messages clients with GoRoute routing and provider fallback."
              endpoints={anthropicEndpoints}
              example={`const client = new Anthropic({
  baseURL: "${baseURL}",
  apiKey: "<system-api-key>",
});

await client.messages.create({
  model: "anthropic/claude-sonnet-4-5",
  messages: [{ role: "user", content: "Hello from GoRoute" }],
});`}
              title="Anthropic-compatible"
              value={baseURL}
            />
          </div>
        )}
      </SectionCard>

      <div>
        <SectionCard
          description="Create client-facing keys for applications that call GoRoute. Protection applies when at least one key exists and the toggle is on."
          headerAction={
            <Button
              disabled={keysQuery.isPending}
              leadingIcon={<Plus className="size-4" />}
              onClick={() => {
                setFormError("");
                setIsAddOpen(true);
              }}
              type="button"
            >
              Add API Key
            </Button>
          }
          title="System API keys"
          tone="solid"
        >
          <div className="mb-4">
            <SettingToggle
              checked={
                settingsQuery.data?.openAICompatibleAuth.enabled ?? false
              }
              disabled={
                settingsQuery.isPending || updateOpenAIAuthMutation.isPending
              }
              onCheckedChange={(enabled) =>
                updateOpenAIAuthMutation.mutate(enabled)
              }
            />
          </div>

          {keysQuery.isPending ? (
            <div className="space-y-3">
              <Skeleton className="min-h-[96px]" />
              <Skeleton className="min-h-[96px]" />
            </div>
          ) : keysQuery.isError ? (
            <InlineAlert tone="error">
              {keysQuery.error instanceof Error
                ? keysQuery.error.message
                : "Request failed"}
            </InlineAlert>
          ) : (keysQuery.data ?? []).length === 0 ? (
            <EmptyState
              action={
                <Button
                  leadingIcon={<Plus className="size-4" />}
                  onClick={() => setIsAddOpen(true)}
                  tone="secondary"
                  type="button"
                >
                  Add API Key
                </Button>
              }
              body="Add a system API key before connecting clients to the proxy."
              title="No API keys yet"
            />
          ) : (
            <div className="space-y-3">
              {(keysQuery.data ?? []).map((apiKey) => (
                <APIKeyRow
                  apiKey={apiKey}
                  key={apiKey.id}
                  onSaveQuotas={(quotas) =>
                    updateKeyMutation.mutate({
                      ...quotas,
                      id: apiKey.id,
                    })
                  }
                  onToggle={(enabled) =>
                    updateKeyMutation.mutate({
                      enabled,
                      id: apiKey.id,
                      name: apiKey.name,
                    })
                  }
                />
              ))}
            </div>
          )}
        </SectionCard>
      </div>

      <Modal onOpenChange={setIsAddOpen} open={isAddOpen}>
        <ModalContent>
          <ModalHeader>
            <ModalTitle>Add API Key</ModalTitle>
            <ModalDescription>
              Name the client-facing system key. The key value is generated by
              the server.
            </ModalDescription>
          </ModalHeader>
          <form onSubmit={handleAddKey}>
            <ModalBody className="space-y-4">
              {formError ? (
                <InlineAlert tone="error">{formError}</InlineAlert>
              ) : null}
              <Field label="Name">
                <Input
                  autoFocus
                  onChange={(event) => setDraftName(event.target.value)}
                  placeholder="Production app"
                  value={draftName}
                />
              </Field>
              <InlineAlert>
                The API key value will be generated when the key is created.
              </InlineAlert>
            </ModalBody>
            <ModalFooter>
              <ModalClose asChild>
                <Button tone="secondary" type="button">
                  Cancel
                </Button>
              </ModalClose>
              <Button type="submit">Save API Key</Button>
            </ModalFooter>
          </form>
        </ModalContent>
      </Modal>
    </section>
  );
}

function SettingToggle({
  checked,
  disabled,
  onCheckedChange,
}: {
  checked: boolean;
  disabled: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <SurfaceCard className="border-border/80 bg-bg-primary/70 p-4" tone="glass">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2 pr-4">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-fg-primary text-sm font-semibold">
              Require system API key for API-compatible clients
            </p>
            <StatusBadge size="sm" tone={checked ? "success" : "warning"}>
              {checked ? "On" : "Off"}
            </StatusBadge>
          </div>
          <p className="text-fg-secondary text-sm leading-6">
            When this is on and at least one system key exists, client calls to
            /v1 endpoints must include a matching Bearer token.
          </p>
        </div>
        <Switch
          aria-label="Require system API key for API-compatible clients"
          checked={checked}
          disabled={disabled}
          onCheckedChange={onCheckedChange}
        />
      </div>
    </SurfaceCard>
  );
}

export function resolveOpenAIBaseURL(listen?: string) {
  if (!listen?.trim() || typeof window === "undefined") {
    return null;
  }

  const trimmed = listen.trim();
  const protocol = window.location.protocol || "http:";
  const currentHostname = window.location.hostname || "localhost";

  if (trimmed.startsWith(":")) {
    const port = trimmed.slice(1);
    return port ? `${protocol}//${currentHostname}:${port}/v1` : null;
  }

  const withProtocol = /^[a-z][a-z\d+\-.]*:\/\//i.test(trimmed)
    ? trimmed
    : `${protocol}//${trimmed}`;

  try {
    const parsed = new URL(withProtocol);
    const hostname =
      parsed.hostname === "localhost" || parsed.hostname === "127.0.0.1"
        ? parsed.hostname
        : parsed.hostname || currentHostname;
    const port = parsed.port ? `:${parsed.port}` : "";

    return port ? `${protocol}//${hostname}${port}/v1` : null;
  } catch {
    return null;
  }
}

function CopyPanel({ label, value }: { label: string; value: string }) {
  return (
    <div className="border-border/80 bg-bg-tertiary/60 flex flex-col gap-3 rounded-[18px] border p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0 space-y-1">
        <p className="text-fg-muted text-[11px] font-semibold tracking-[0.18em] uppercase">
          {label}
        </p>
        <p className="text-fg-primary font-mono text-sm font-semibold break-all">
          {value}
        </p>
      </div>
      <CopyButton copyValue={value} tone="secondary" type="button" />
    </div>
  );
}

function ProtocolPanel({
  description,
  endpoints,
  example,
  title,
  value,
}: {
  description: string;
  endpoints: Array<{ label: string; value: string }>;
  example: string;
  title: string;
  value: string;
}) {
  return (
    <SurfaceCard className="flex min-h-[420px] flex-col gap-4 p-4" tone="glass">
      <div className="space-y-2">
        <p className="text-fg-primary text-base font-semibold">{title}</p>
        <p className="text-fg-secondary text-sm leading-6">{description}</p>
      </div>

      <CopyPanel label="Base URL" value={value} />

      <div className="grid min-h-[118px] content-start gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {endpoints.map((endpoint) => (
          <EndpointCard
            key={endpoint.value}
            label={endpoint.label}
            value={endpoint.value}
            wide={endpoints.length === 1}
          />
        ))}
      </div>

      <div className="bg-bg-tertiary/60 border-border/70 mt-auto overflow-x-auto rounded-[16px] border p-4">
        <p className="text-fg-muted mb-3 text-[11px] font-semibold tracking-[0.18em] uppercase">
          Example
        </p>
        <pre className="text-fg-secondary text-xs leading-6">{example}</pre>
      </div>
    </SurfaceCard>
  );
}

function EndpointCard({
  label,
  value,
  wide = false,
}: {
  label: string;
  value: string;
  wide?: boolean;
}) {
  return (
    <SurfaceCard
      className={wide ? "p-4 sm:col-span-2 xl:col-span-3" : "p-4"}
      tone="glass"
    >
      <div className="space-y-2">
        <div className="dashboard-icon-surface flex size-9 items-center justify-center rounded-[12px] border">
          <Link2 className="size-4" />
        </div>
        <p className="text-fg-primary text-sm font-semibold">{label}</p>
        <p className="text-fg-secondary overflow-hidden font-mono text-[11px] leading-5 text-ellipsis whitespace-nowrap">
          {value}
        </p>
      </div>
    </SurfaceCard>
  );
}

function APIKeyRow({
  apiKey,
  onSaveQuotas,
  onToggle,
}: {
  apiKey: SystemAPIKey;
  onSaveQuotas: (quotas: {
    daily_token_limit: number | null;
    monthly_token_limit: number | null;
    requests_per_minute_limit: number | null;
  }) => void;
  onToggle: (enabled: boolean) => void;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const [rpmLimit, setRPMLimit] = useState(
    limitToInputValue(apiKey.requests_per_minute_limit),
  );
  const [dailyLimit, setDailyLimit] = useState(
    limitToInputValue(apiKey.daily_token_limit),
  );
  const [monthlyLimit, setMonthlyLimit] = useState(
    limitToInputValue(apiKey.monthly_token_limit),
  );

  function resetDraft() {
    setRPMLimit(limitToInputValue(apiKey.requests_per_minute_limit));
    setDailyLimit(limitToInputValue(apiKey.daily_token_limit));
    setMonthlyLimit(limitToInputValue(apiKey.monthly_token_limit));
  }

  function handleSaveQuotas() {
    onSaveQuotas({
      daily_token_limit: inputValueToLimit(dailyLimit),
      monthly_token_limit: inputValueToLimit(monthlyLimit),
      requests_per_minute_limit: inputValueToLimit(rpmLimit),
    });
    setIsEditing(false);
  }

  return (
    <SurfaceCard className="p-4" tone="glass">
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <p className="text-fg-primary text-sm font-semibold">
                {apiKey.name}
              </p>
              <StatusBadge
                size="sm"
                tone={apiKey.enabled ? "success" : "warning"}
              >
                {apiKey.enabled ? "Enabled" : "Disabled"}
              </StatusBadge>
              {apiKey.usage?.rate_limit_reached ||
              apiKey.usage?.daily_limit_reached ||
              apiKey.usage?.monthly_limit_reached ? (
                <StatusBadge size="sm" tone="warning">
                  Limited
                </StatusBadge>
              ) : null}
            </div>
            <p className="text-fg-secondary font-mono text-xs break-all">
              {maskAPIKey(apiKey.key)}
            </p>
            <p className="text-fg-muted text-xs">
              Created {formatTimestamp(apiKey.created_at)} · Last used{" "}
              {formatTimestamp(apiKey.last_used_at)}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <CopyButton
              aria-label={`Copy ${apiKey.name} API key`}
              copyValue={apiKey.key}
              iconOnly
              tone="secondary"
              type="button"
            />
            <Button
              aria-label={`Edit ${apiKey.name} quotas`}
              leadingIcon={<SlidersHorizontal className="size-4" />}
              onClick={() => {
                resetDraft();
                setIsEditing((current) => !current);
              }}
              tone="secondary"
              type="button"
            >
              Quotas
            </Button>
            <Switch
              aria-label={`${apiKey.name} API key enabled`}
              checked={apiKey.enabled}
              onCheckedChange={onToggle}
              size="sm"
            />
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          <QuotaMetric
            label="Requests/min"
            limit={apiKey.requests_per_minute_limit}
            tone={apiKey.usage?.rate_limit_reached ? "warning" : "default"}
            used={apiKey.usage?.current_minute_requests ?? 0}
          />
          <QuotaMetric
            label="Daily tokens"
            limit={apiKey.daily_token_limit}
            remaining={apiKey.usage?.daily_tokens.remaining ?? null}
            tone={apiKey.usage?.daily_limit_reached ? "warning" : "default"}
            used={apiKey.usage?.daily_tokens.used ?? 0}
          />
          <QuotaMetric
            label="Monthly tokens"
            limit={apiKey.monthly_token_limit}
            remaining={apiKey.usage?.monthly_tokens.remaining ?? null}
            tone={apiKey.usage?.monthly_limit_reached ? "warning" : "default"}
            used={apiKey.usage?.monthly_tokens.used ?? 0}
          />
        </div>

        {isEditing ? (
          <div className="border-border/70 bg-bg-tertiary/40 grid gap-3 rounded-[12px] border p-3 lg:grid-cols-[1fr_1fr_1fr_auto] lg:items-end">
            <Field label="Requests/minute">
              <Input
                inputMode="numeric"
                min={1}
                onChange={(event) => setRPMLimit(event.target.value)}
                placeholder="Unlimited"
                type="number"
                value={rpmLimit}
              />
            </Field>
            <Field label="Daily tokens">
              <Input
                inputMode="numeric"
                min={1}
                onChange={(event) => setDailyLimit(event.target.value)}
                placeholder="Unlimited"
                type="number"
                value={dailyLimit}
              />
            </Field>
            <Field label="Monthly tokens">
              <Input
                inputMode="numeric"
                min={1}
                onChange={(event) => setMonthlyLimit(event.target.value)}
                placeholder="Unlimited"
                type="number"
                value={monthlyLimit}
              />
            </Field>
            <div className="flex gap-2">
              <Button
                leadingIcon={<Save className="size-4" />}
                onClick={handleSaveQuotas}
                type="button"
              >
                Save
              </Button>
              <Button
                aria-label={`Cancel ${apiKey.name} quota edits`}
                leadingIcon={<X className="size-4" />}
                onClick={() => {
                  resetDraft();
                  setIsEditing(false);
                }}
                tone="secondary"
                type="button"
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    </SurfaceCard>
  );
}

function QuotaMetric({
  label,
  limit,
  remaining,
  tone,
  used,
}: {
  label: string;
  limit: number | null;
  remaining?: number | null;
  tone: "default" | "warning";
  used: number;
}) {
  const limitLabel = limit == null ? "Unlimited" : formatNumber(limit);
  return (
    <div className="border-border/70 bg-bg-primary/60 rounded-[12px] border p-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-fg-muted text-[11px] font-semibold tracking-[0.14em] uppercase">
          {label}
        </p>
        <StatusBadge size="sm" tone={tone === "warning" ? "warning" : "info"}>
          {limitLabel}
        </StatusBadge>
      </div>
      <p className="text-fg-primary mt-2 text-lg font-semibold">
        {formatNumber(used)}
      </p>
      <p className="text-fg-muted text-xs">
        {remaining == null ? "No quota cap" : `${formatNumber(remaining)} left`}
      </p>
    </div>
  );
}

function limitToInputValue(value: number | null) {
  return value == null ? "" : String(value);
}

function inputValueToLimit(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  const parsed = Number.parseInt(trimmed, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

function formatNumber(value: number) {
  return new Intl.NumberFormat().format(value);
}

function maskAPIKey(value: string) {
  if (value.length <= 8) {
    return "••••";
  }

  return `${value.slice(0, Math.min(10, value.length - 4))}...${value.slice(-4)}`;
}

function formatTimestamp(value: number) {
  if (!value) {
    return "Never";
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value * 1000));
}

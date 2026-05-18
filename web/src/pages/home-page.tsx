import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Clipboard,
  KeyRound,
  Link2,
  Plus,
  ServerCog,
  Workflow,
} from "lucide-react";
import type { FormEvent, ReactNode } from "react";
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
import { Button } from "@/shared/ui/button";
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

const endpoints = [
  { label: "Models", value: "GET /v1/models" },
  { label: "Chat Completions", value: "POST /v1/chat/completions" },
  { label: "Responses", value: "POST /v1/responses" },
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
  const [copyFeedback, setCopyFeedback] = useState("");

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
      enabled,
      id,
      name,
    }: {
      enabled?: boolean;
      id: string;
      name?: string;
    }) => updateSystemAPIKey(id, { enabled, name }),
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

  async function copyText(value: string, message: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopyFeedback(message);
    } catch {
      setCopyFeedback("Clipboard is not available in this browser.");
    }
  }

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Connect OpenAI-compatible clients to GoRoute, manage system API keys, and keep routing policy inside the proxy."
        eyebrow="Home"
        title="AI connection hub"
      >
        <StatusBadge tone="info">OpenAI-compatible</StatusBadge>
      </PageHeader>

      {copyFeedback ? <InlineAlert>{copyFeedback}</InlineAlert> : null}

      <div className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <div className="space-y-6">
          <SectionCard
            description="Use this base URL in OpenAI-compatible SDKs and tools. The port is resolved from the current runtime settings."
            title="Connection details"
            tone="solid"
          >
            {settingsQuery.isPending ? (
              <div className="space-y-3">
                <Skeleton className="min-h-[76px]" />
                <Skeleton className="min-h-[120px]" />
              </div>
            ) : settingsQuery.isError || !baseURL ? (
              <InlineAlert tone="error">
                Cannot resolve server port from runtime settings.
              </InlineAlert>
            ) : (
              <div className="space-y-4">
                <CopyPanel
                  label="Base URL"
                  value={baseURL}
                  onCopy={() => copyText(baseURL, "Base URL copied.")}
                />
                <div className="grid gap-3 md:grid-cols-3">
                  {endpoints.map((endpoint) => (
                    <EndpointCard
                      key={endpoint.value}
                      label={endpoint.label}
                      value={endpoint.value}
                    />
                  ))}
                </div>
              </div>
            )}
          </SectionCard>

          <SectionCard
            description="A minimal request shape for clients that already speak the OpenAI API."
            title="Client example"
          >
            <div className="space-y-4">
              <div className="grid gap-3 md:grid-cols-3">
                <InfoTile
                  icon={<ServerCog className="size-4" />}
                  label="Base URL"
                  value={baseURL ?? "Unavailable"}
                />
                <InfoTile
                  icon={<KeyRound className="size-4" />}
                  label="API key"
                  value="Use a system API key"
                />
                <InfoTile
                  icon={<Workflow className="size-4" />}
                  label="Model"
                  value="cx/gpt-5.4"
                />
              </div>
              <div className="bg-bg-tertiary/60 border-border/70 overflow-x-auto rounded-[16px] border p-4">
                <pre className="text-fg-secondary text-xs leading-6">
                  {`const client = new OpenAI({
  baseURL: "${baseURL ?? "<runtime-base-url>"}",
  apiKey: "<system-api-key>",
});

await client.chat.completions.create({
  model: "cx/gpt-5.4",
  messages: [{ role: "user", content: "Hello from GoRoute" }],
});`}
                </pre>
              </div>
            </div>
          </SectionCard>
        </div>

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
                  onCopy={() =>
                    copyText(apiKey.key, `${apiKey.name} key copied.`)
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
              Require system API key for OpenAI-compatible clients
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
          aria-label="Require system API key for OpenAI-compatible clients"
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

function CopyPanel({
  label,
  onCopy,
  value,
}: {
  label: string;
  onCopy: () => void;
  value: string;
}) {
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
      <Button
        leadingIcon={<Clipboard className="size-4" />}
        onClick={onCopy}
        tone="secondary"
        type="button"
      >
        Copy
      </Button>
    </div>
  );
}

function EndpointCard({ label, value }: { label: string; value: string }) {
  return (
    <SurfaceCard className="p-4" tone="glass">
      <div className="space-y-2">
        <div className="dashboard-icon-surface flex size-9 items-center justify-center rounded-[12px] border">
          <Link2 className="size-4" />
        </div>
        <p className="text-fg-primary text-sm font-semibold">{label}</p>
        <p className="text-fg-secondary font-mono text-xs break-all">{value}</p>
      </div>
    </SurfaceCard>
  );
}

function InfoTile({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="border-border/70 bg-bg-tertiary/45 rounded-[16px] border p-4">
      <div className="text-primary mb-3">{icon}</div>
      <p className="text-fg-muted text-[11px] font-semibold tracking-[0.18em] uppercase">
        {label}
      </p>
      <p className="text-fg-primary mt-1 text-sm font-semibold break-words">
        {value}
      </p>
    </div>
  );
}

function APIKeyRow({
  apiKey,
  onCopy,
  onToggle,
}: {
  apiKey: SystemAPIKey;
  onCopy: () => void;
  onToggle: (enabled: boolean) => void;
}) {
  return (
    <SurfaceCard className="p-4" tone="glass">
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
          <Button
            aria-label={`Copy ${apiKey.name} API key`}
            iconOnly
            leadingIcon={<Clipboard className="size-4" />}
            onClick={onCopy}
            tone="secondary"
            type="button"
          />
          <Switch
            aria-label={`${apiKey.name} API key enabled`}
            checked={apiKey.enabled}
            onCheckedChange={onToggle}
            size="sm"
          />
        </div>
      </div>
    </SurfaceCard>
  );
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

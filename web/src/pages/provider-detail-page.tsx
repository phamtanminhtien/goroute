import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Pencil, Play, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import {
  type ConnectionPayload,
  createConnection,
  deleteConnection,
  listProviders,
  type ProviderConnection,
  type ProviderItem,
  providersQueryKey,
  updateConnection,
} from "@/features/providers/api";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { StatusBadge } from "@/shared/ui/status-badge";

type FeedbackState = {
  text: string;
  tone: "error" | "success";
} | null;

type ConnectionFormValues = {
  accessToken: string;
  apiKey: string;
  id: string;
  name: string;
  refreshToken: string;
};

const emptyFormValues: ConnectionFormValues = {
  accessToken: "",
  apiKey: "",
  id: "",
  name: "",
  refreshToken: "",
};

export function ProviderDetailPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { providerId } = useParams<{ providerId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const mode = searchParams.get("mode");
  const [editingConnectionID, setEditingConnectionID] = useState<string | null>(
    null,
  );
  const [isCreating, setIsCreating] = useState(() => mode === "create");
  const [feedback, setFeedback] = useState<FeedbackState>(null);

  const providersQuery = useQuery({
    queryFn: listProviders,
    queryKey: providersQueryKey,
  });

  const provider = (providersQuery.data ?? []).find(
    (item) => item.id === providerId,
  );
  const editingConnection =
    provider?.connections.find((item) => item.id === editingConnectionID) ??
    null;

  const createConnectionMutation = useMutation({
    mutationFn: createConnection,
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setFeedback({ text: "Connection saved.", tone: "success" });
      setIsCreating(false);
      clearModeParam(setSearchParams);
    },
  });

  const updateConnectionMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: ConnectionPayload }) =>
      updateConnection(id, payload),
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setFeedback({ text: "Connection saved.", tone: "success" });
      setEditingConnectionID(null);
    },
  });

  const deleteConnectionMutation = useMutation({
    mutationFn: deleteConnection,
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setFeedback({ text: "Connection deleted.", tone: "success" });
      setEditingConnectionID(null);
    },
  });

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Inspect provider capabilities, manage connection credentials, and review the models this provider exposes."
        eyebrow="Providers"
        title={provider?.name ?? "Provider detail"}
      >
        <Button
          leadingIcon={<ArrowLeft className="size-[15px]" />}
          onClick={() => navigate("/providers")}
          tone="secondary"
        >
          Back to registry
        </Button>
      </PageHeader>

      {providersQuery.isPending ? (
        <SectionCard
          description="Loading provider metadata and grouped connections."
          title="Loading provider"
          tone="solid"
        >
          <div className="border-border/85 bg-bg-primary/72 min-h-[220px] animate-pulse rounded-[22px] border" />
        </SectionCard>
      ) : null}

      {providersQuery.isError ? (
        <SectionCard
          description="The provider detail view could not be loaded from the admin API."
          title="Provider detail unavailable"
          tone="solid"
        >
          <div className="space-y-4">
            <p className="text-sm leading-6 text-rose-700" role="alert">
              {providersQuery.error instanceof Error
                ? providersQuery.error.message
                : "Request failed"}
            </p>
            <Button onClick={() => providersQuery.refetch()} tone="secondary">
              Retry request
            </Button>
          </div>
        </SectionCard>
      ) : null}

      {!providersQuery.isPending && !providersQuery.isError && !provider ? (
        <SectionCard
          description="The requested provider was not returned by the admin API."
          title="Provider not found"
          tone="solid"
        >
          <Button onClick={() => navigate("/providers")} tone="secondary">
            Return to registry
          </Button>
        </SectionCard>
      ) : null}

      {!providersQuery.isPending && !providersQuery.isError && provider ? (
        <div className="space-y-6">
          <SectionCard
            description="Create, edit, and remove redacted provider connections without exposing stored secrets."
            headerAction={
              <Button
                leadingIcon={<Plus className="size-[15px]" />}
                onClick={() => {
                  setFeedback(null);
                  setEditingConnectionID(null);
                  setIsCreating(true);
                  setSearchParams({ mode: "create" });
                }}
              >
                Add connection
              </Button>
            }
            title="Connections"
            tone="solid"
          >
            <div className="space-y-5">
              <div className="flex flex-wrap items-center gap-3">
                <StatusBadge tone="info">
                  {buildConnectionStatus(provider.connection_count)}
                </StatusBadge>
                <StatusBadge tone="info">
                  {buildAuthLabel(provider.auth_type)}
                </StatusBadge>
              </div>

              {feedback ? (
                <FeedbackBanner text={feedback.text} tone={feedback.tone} />
              ) : null}

              {provider.connection_count === 0 ? (
                <EmptyState
                  body="This provider does not have any configured connections yet."
                  title="No connections"
                />
              ) : (
                <div className="space-y-3">
                  {provider.connections.map((connection) => (
                    <ConnectionCard
                      busy={
                        deleteConnectionMutation.isPending &&
                        deleteConnectionMutation.variables === connection.id
                      }
                      connection={connection}
                      isEditing={editingConnectionID === connection.id}
                      key={connection.id}
                      onDelete={async () => {
                        if (
                          !window.confirm(
                            `Delete connection "${connection.name}"?`,
                          )
                        ) {
                          return;
                        }

                        setFeedback(null);
                        await deleteConnectionMutation.mutateAsync(
                          connection.id,
                        );
                      }}
                      onEdit={() => {
                        clearModeParam(setSearchParams);
                        setFeedback(null);
                        setIsCreating(false);
                        setEditingConnectionID(connection.id);
                      }}
                    />
                  ))}
                </div>
              )}

              {isCreating ? (
                <ConnectionForm
                  busy={createConnectionMutation.isPending}
                  key="create-connection"
                  mode="create"
                  onCancel={() => {
                    setIsCreating(false);
                    clearModeParam(setSearchParams);
                  }}
                  onSubmit={async (values) => {
                    if (!provider) {
                      return;
                    }

                    setFeedback(null);
                    await createConnectionMutation.mutateAsync(
                      buildConnectionPayload(provider.id, values),
                    );
                  }}
                  provider={provider}
                />
              ) : null}

              {editingConnection ? (
                <ConnectionForm
                  busy={updateConnectionMutation.isPending}
                  initialValues={{
                    accessToken: "",
                    apiKey: "",
                    id: editingConnection.id,
                    name: editingConnection.name,
                    refreshToken: "",
                  }}
                  key={editingConnection.id}
                  mode="edit"
                  onCancel={() => setEditingConnectionID(null)}
                  onSubmit={async (values) => {
                    if (!provider) {
                      return;
                    }

                    setFeedback(null);
                    await updateConnectionMutation.mutateAsync({
                      id: editingConnection.id,
                      payload: buildConnectionPayload(provider.id, values),
                    });
                  }}
                  provider={provider}
                />
              ) : null}
            </div>
          </SectionCard>

          <SectionCard
            description="Quick visibility into the models this provider currently exposes."
            title="Available models"
            tone="solid"
          >
            {provider.models.length === 0 ? (
              <EmptyState
                body="No models are currently advertised for this provider."
                title="No available models"
              />
            ) : (
              <div className="flex flex-wrap gap-3">
                {provider.models.map((model) => (
                  <ModelChip
                    isDefault={model.id === provider.default_model}
                    key={model.id}
                    model={model}
                  />
                ))}
              </div>
            )}
          </SectionCard>
        </div>
      ) : null}
    </section>
  );
}

function ConnectionCard({
  busy,
  connection,
  isEditing,
  onDelete,
  onEdit,
}: {
  busy: boolean;
  connection: ProviderConnection;
  isEditing: boolean;
  onDelete: () => void;
  onEdit: () => void;
}) {
  return (
    <div className="border-border/85 bg-bg-primary/72 rounded-[20px] border px-4 py-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-fg-primary text-sm font-semibold">
            {connection.name}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {connection.has_access_token ? (
            <StatusBadge tone="success">Access token</StatusBadge>
          ) : null}
          {connection.has_api_key ? (
            <StatusBadge tone="success">API key</StatusBadge>
          ) : null}
          {connection.has_refresh_token ? (
            <StatusBadge tone="info">Refresh token</StatusBadge>
          ) : null}
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button
          leadingIcon={<Pencil className="size-[15px]" />}
          onClick={onEdit}
          tone={isEditing ? "primary" : "secondary"}
        >
          Edit connection
        </Button>
        <Button
          disabled={busy}
          leadingIcon={<Trash2 className="size-[15px]" />}
          onClick={onDelete}
          tone="ghost"
        >
          {busy ? "Deleting..." : "Delete"}
        </Button>
      </div>
    </div>
  );
}

function ConnectionForm({
  busy,
  initialValues = emptyFormValues,
  mode,
  onCancel,
  onSubmit,
  provider,
}: {
  busy: boolean;
  initialValues?: ConnectionFormValues;
  mode: "create" | "edit";
  onCancel: () => void;
  onSubmit: (values: ConnectionFormValues) => void | Promise<void>;
  provider: ProviderItem;
}) {
  const [values, setValues] = useState(initialValues);
  const [formError, setFormError] = useState<string | null>(null);

  return (
    <div className="border-border/85 bg-bg-primary/72 space-y-4 rounded-[20px] border px-4 py-4">
      <div className="space-y-1">
        <h3 className="text-fg-primary text-base font-semibold">
          {mode === "create" ? "Add connection" : "Edit connection"}
        </h3>
        <p className="text-fg-secondary text-sm leading-6">
          Secrets stay blank on edit. Enter only the values you want to replace.
        </p>
      </div>

      <form
        className="grid gap-4"
        onSubmit={async (event) => {
          event.preventDefault();

          if (!values.id.trim() || !values.name.trim()) {
            setFormError("Connection ID and display name are required.");
            return;
          }

          setFormError(null);
          await onSubmit(values);
        }}
      >
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Connection ID" required>
            <Input
              onChange={(event) =>
                setValues((current) => ({ ...current, id: event.target.value }))
              }
              placeholder="openai-1"
              value={values.id}
            />
          </Field>
          <Field label="Display name" required>
            <Input
              onChange={(event) =>
                setValues((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
              placeholder="user@example.com"
              value={values.name}
            />
          </Field>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Field
            help={
              provider.auth_type === "oauth"
                ? "Primary credential for OAuth-style providers."
                : "Optional token fallback for providers that accept bearer credentials."
            }
            label="Access token"
          >
            <Input
              onChange={(event) =>
                setValues((current) => ({
                  ...current,
                  accessToken: event.target.value,
                }))
              }
              placeholder="Enter a new access token"
              type="password"
              value={values.accessToken}
            />
          </Field>
          <Field
            help={
              provider.auth_type === "api_key"
                ? "Primary credential for API key providers."
                : "Optional fallback key if this provider supports it."
            }
            label="API key"
          >
            <Input
              onChange={(event) =>
                setValues((current) => ({
                  ...current,
                  apiKey: event.target.value,
                }))
              }
              placeholder="Enter a new API key"
              type="password"
              value={values.apiKey}
            />
          </Field>
        </div>

        <Field
          help="Only needed when the provider uses token refresh."
          label="Refresh token"
        >
          <Input
            onChange={(event) =>
              setValues((current) => ({
                ...current,
                refreshToken: event.target.value,
              }))
            }
            placeholder="Enter a new refresh token"
            type="password"
            value={values.refreshToken}
          />
        </Field>

        {formError ? (
          <p className="text-sm leading-6 text-rose-700" role="alert">
            {formError}
          </p>
        ) : null}

        <div className="flex flex-wrap gap-2">
          <Button disabled={busy} type="submit">
            {busy
              ? "Saving..."
              : mode === "create"
                ? "Create connection"
                : "Save changes"}
          </Button>
          <Button
            disabled={busy}
            onClick={onCancel}
            tone="secondary"
            type="button"
          >
            Cancel
          </Button>
        </div>
      </form>
    </div>
  );
}

function EmptyState({ body, title }: { body: string; title: string }) {
  return (
    <div className="border-border/85 bg-bg-primary/72 space-y-2 rounded-[20px] border px-4 py-4">
      <h3 className="text-fg-primary text-sm font-semibold">{title}</h3>
      <p className="text-fg-secondary text-sm leading-6">{body}</p>
    </div>
  );
}

function FeedbackBanner({
  text,
  tone,
}: {
  text: string;
  tone: "error" | "success";
}) {
  return (
    <div
      className={
        tone === "success"
          ? "rounded-[20px] border border-emerald-500/25 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-700 dark:text-emerald-300"
          : "rounded-[20px] border border-rose-500/25 bg-rose-500/10 px-4 py-3 text-sm text-rose-700"
      }
      role={tone === "error" ? "alert" : "status"}
    >
      {text}
    </div>
  );
}

function buildConnectionPayload(
  providerID: string,
  values: ConnectionFormValues,
): ConnectionPayload {
  const payload: ConnectionPayload = {
    id: values.id.trim(),
    name: values.name.trim(),
    provider_id: providerID,
  };

  if (values.accessToken.trim()) {
    payload.access_token = values.accessToken.trim();
  }
  if (values.apiKey.trim()) {
    payload.api_key = values.apiKey.trim();
  }
  if (values.refreshToken.trim()) {
    payload.refresh_token = values.refreshToken.trim();
  }

  return payload;
}

function buildConnectionStatus(count: number) {
  if (count === 0) {
    return "No connections";
  }
  if (count === 1) {
    return "1 Connected";
  }

  return `${count} Connected`;
}

function buildAuthLabel(authType: string) {
  return authType === "api_key" ? "API key provider" : "OAuth provider";
}

function ModelChip({
  isDefault,
  model,
}: {
  isDefault: boolean;
  model: ProviderItem["models"][number];
}) {
  return (
    <div className="border-border/85 bg-bg-primary/72 flex max-w-full min-w-[240px] items-center gap-3 rounded-[18px] border px-3.5 py-3">
      <div className="min-w-0 flex-1">
        <p className="text-fg-primary truncate text-sm font-semibold">
          {model.name}
        </p>
        <p className="text-fg-secondary truncate text-sm">{model.id}</p>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        {isDefault ? <StatusBadge tone="success">Default</StatusBadge> : null}
        {!isDefault ? <StatusBadge tone="info">Ready</StatusBadge> : null}
        <Button leadingIcon={<Play className="size-[15px]" />} tone="secondary">
          Test
        </Button>
      </div>
    </div>
  );
}

function clearModeParam(
  setSearchParams: ReturnType<typeof useSearchParams>[1],
) {
  setSearchParams((current) => {
    const nextParams = new URLSearchParams(current);
    nextParams.delete("mode");
    return nextParams;
  });
}

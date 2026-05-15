import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Pencil, Play, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

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
import {
  type ConnectionFormFeedback,
  type ConnectionFormValues,
  emptyConnectionFormValues,
  getProviderConnectionFormEntry,
} from "@/features/providers/connection-form-registry";
import {
  AlertDialog,
  AlertDialogActionButton,
  AlertDialogCancelButton,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/shared/ui/alert-dialog";
import { Button } from "@/shared/ui/button";
import { CardActionRow } from "@/shared/ui/card-action-row";
import { EmptyState } from "@/shared/ui/empty-state";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { Modal, ModalContent, ModalPanel } from "@/shared/ui/modal";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/shared/ui/status-badge";

type FeedbackState = ConnectionFormFeedback;

type ConnectionModalState =
  | { kind: "closed" }
  | { kind: "create"; providerId: string }
  | { connectionId: string; kind: "edit"; providerId: string };

export function ProviderDetailPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { providerId } = useParams<{ providerId: string }>();
  const [modalState, setModalState] = useState<ConnectionModalState>({
    kind: "closed",
  });
  const [feedback, setFeedback] = useState<FeedbackState>(null);

  const providersQuery = useQuery({
    queryFn: listProviders,
    queryKey: providersQueryKey,
  });

  const provider = (providersQuery.data ?? []).find(
    (item) => item.id === providerId,
  );
  const editingConnection =
    modalState.kind === "edit" && provider?.id === modalState.providerId
      ? (provider.connections.find(
          (item) => item.id === modalState.connectionId,
        ) ?? null)
      : null;
  const formRegistryEntry = provider
    ? getProviderConnectionFormEntry(provider.id)
    : null;

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
      setModalState({ kind: "closed" });
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
      setModalState({ kind: "closed" });
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
      setModalState({ kind: "closed" });
    },
  });

  const activeModal =
    provider && formRegistryEntry
      ? buildConnectionModal({
          createBusy: createConnectionMutation.isPending,
          editBusy: updateConnectionMutation.isPending,
          editingConnection,
          entry: formRegistryEntry,
          feedback,
          modalState,
          onCancel: () => setModalState({ kind: "closed" }),
          onCreate: async (values) => {
            setFeedback(null);
            await createConnectionMutation.mutateAsync(
              buildConnectionPayload(provider.id, values),
            );
          },
          onEdit: async (values) => {
            if (!editingConnection) {
              return;
            }

            setFeedback(null);
            await updateConnectionMutation.mutateAsync({
              id: editingConnection.id,
              payload: buildConnectionPayload(provider.id, values),
            });
          },
          provider,
        })
      : null;

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
          <Skeleton className="min-h-[220px]" />
        </SectionCard>
      ) : null}

      {providersQuery.isError ? (
        <SectionCard
          description="The provider detail view could not be loaded from the admin API."
          title="Provider detail unavailable"
          tone="solid"
        >
          <div className="space-y-4">
            <InlineAlert tone="error">
              {providersQuery.error instanceof Error
                ? providersQuery.error.message
                : "Request failed"}
            </InlineAlert>
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
                  setModalState({ kind: "create", providerId: provider.id });
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
                <InlineAlert
                  tone={feedback.tone === "success" ? "success" : "error"}
                >
                  {feedback.text}
                </InlineAlert>
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
                      isEditing={
                        modalState.kind === "edit" &&
                        modalState.connectionId === connection.id
                      }
                      key={connection.id}
                      onDelete={async () => {
                        setFeedback(null);
                        await deleteConnectionMutation.mutateAsync(
                          connection.id,
                        );
                      }}
                      onEdit={() => {
                        setFeedback(null);
                        setModalState({
                          connectionId: connection.id,
                          kind: "edit",
                          providerId: provider.id,
                        });
                      }}
                    />
                  ))}
                </div>
              )}
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

      <Modal
        onOpenChange={(open) => {
          if (!open) {
            setModalState({ kind: "closed" });
          }
        }}
        open={activeModal !== null}
      >
        {activeModal ? (
          <ModalContent>
            <ModalPanel
              description={activeModal.description}
              title={activeModal.title}
            >
              {activeModal.content}
            </ModalPanel>
          </ModalContent>
        ) : null}
      </Modal>
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
    <CardActionRow
      actions={
        <>
          <Button
            leadingIcon={<Pencil className="size-[15px]" />}
            onClick={onEdit}
            tone={isEditing ? "primary" : "secondary"}
          >
            Edit connection
          </Button>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                disabled={busy}
                leadingIcon={<Trash2 className="size-[15px]" />}
                tone="ghost"
              >
                {busy ? "Deleting..." : "Delete"}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete connection?</AlertDialogTitle>
                <AlertDialogDescription>
                  Delete connection "{connection.name}" from this provider. This
                  removes the stored credential reference and cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancelButton>
                  Keep connection
                </AlertDialogCancelButton>
                <AlertDialogActionButton onClick={onDelete} tone="primary">
                  Confirm delete
                </AlertDialogActionButton>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </>
      }
      description={
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
      }
      title={connection.name}
    />
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

function buildConnectionModal({
  createBusy,
  editBusy,
  editingConnection,
  entry,
  feedback,
  modalState,
  onCancel,
  onCreate,
  onEdit,
  provider,
}: {
  createBusy: boolean;
  editBusy: boolean;
  editingConnection: ProviderConnection | null;
  entry: ReturnType<typeof getProviderConnectionFormEntry>;
  feedback: FeedbackState;
  modalState: ConnectionModalState;
  onCancel: () => void;
  onCreate: (values: ConnectionFormValues) => void | Promise<void>;
  onEdit: (values: ConnectionFormValues) => void | Promise<void>;
  provider: ProviderItem;
}) {
  if (modalState.kind === "create" && modalState.providerId === provider.id) {
    const meta = entry.getCreateMeta?.(provider) ?? {
      description:
        "Add a new provider connection and store credentials securely.",
      title: "Add connection",
    };

    return {
      content: entry.renderCreate({
        busy: createBusy,
        feedback,
        onCancel,
        onSubmit: onCreate,
        provider,
      }),
      description: meta.description,
      title: meta.title,
    };
  }

  if (
    modalState.kind === "edit" &&
    modalState.providerId === provider.id &&
    editingConnection
  ) {
    const meta = entry.getEditMeta?.(provider, editingConnection) ?? {
      description:
        "Secrets stay blank on edit. Enter only the values you want to replace.",
      title: "Edit connection",
    };

    return {
      content: entry.renderEdit({
        busy: editBusy,
        connection: editingConnection,
        feedback,
        initialValues: {
          ...emptyConnectionFormValues,
          id: editingConnection.id,
          name: editingConnection.name,
        },
        onCancel,
        onSubmit: onEdit,
        provider,
      }),
      description: meta.description,
      title: meta.title,
    };
  }

  return null;
}

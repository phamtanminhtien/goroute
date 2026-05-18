import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Pencil, Play, Plus, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  completeOAuthConnection,
  type ConnectionPayload,
  createConnection,
  createProviderModel,
  deleteConnection,
  deleteProviderModel,
  listProviders,
  type ProviderConnection,
  type ProviderItem,
  type ProviderModelTestResult,
  providersQueryKey,
  testProviderModel,
  updateConnection,
  updateProviderModel,
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
import { Field } from "@/shared/ui/field";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { Input } from "@/shared/ui/input";
import {
  Modal,
  ModalContent,
  ModalFooter,
  ModalPanel,
} from "@/shared/ui/modal";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/shared/ui/status-badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/shared/ui/tooltip";

type FeedbackState = ConnectionFormFeedback;
type ModelTestState = {
  pending: boolean;
  result: ProviderModelTestResult | null;
  error: string | null;
};
type ModelFormState = {
  description: string;
  id: string;
  name: string;
};

type ConnectionModalState =
  | { kind: "closed" }
  | { kind: "create"; providerId: string }
  | { connectionId: string; kind: "edit"; providerId: string };
type ModelModalState =
  | { kind: "closed" }
  | { kind: "create" }
  | { kind: "edit"; modelId: string };

export function ProviderDetailPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { providerId } = useParams<{ providerId: string }>();
  const [modalState, setModalState] = useState<ConnectionModalState>({
    kind: "closed",
  });
  const [feedback, setFeedback] = useState<FeedbackState>(null);
  const [modelTests, setModelTests] = useState<Record<string, ModelTestState>>(
    {},
  );
  const [modelForm, setModelForm] = useState<ModelFormState>({
    description: "",
    id: "",
    name: "",
  });
  const [modelFeedback, setModelFeedback] = useState<FeedbackState>(null);
  const [modelModalState, setModelModalState] = useState<ModelModalState>({
    kind: "closed",
  });

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
    mutationFn: async (values: ConnectionFormValues) => {
      if (values.oauthSessionID && values.callbackURL) {
        return completeOAuthConnection(
          values.oauthSessionID,
          values.callbackURL,
        );
      }

      return createConnection(buildConnectionPayload(providerId ?? "", values));
    },
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

  const createProviderModelMutation = useMutation({
    mutationFn: async (values: ModelFormState) => {
      if (!provider) {
        throw new Error("Provider not found");
      }

      return createProviderModel(provider.id, {
        description: values.description.trim(),
        id: values.id.trim(),
        name: values.name.trim(),
      });
    },
    onError: (error) => {
      setModelFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setModelFeedback({ text: "Model added.", tone: "success" });
      setModelForm({ description: "", id: "", name: "" });
      setModelModalState({ kind: "closed" });
    },
  });

  const updateProviderModelMutation = useMutation({
    mutationFn: async ({
      modelID,
      values,
    }: {
      modelID: string;
      values: ModelFormState;
    }) => {
      if (!provider) {
        throw new Error("Provider not found");
      }

      return updateProviderModel(provider.id, modelID, {
        description: values.description.trim(),
        id: values.id.trim(),
        name: values.name.trim(),
      });
    },
    onError: (error) => {
      setModelFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setModelFeedback({ text: "Model updated.", tone: "success" });
      setModelForm({ description: "", id: "", name: "" });
      setModelModalState({ kind: "closed" });
    },
  });

  const deleteProviderModelMutation = useMutation({
    mutationFn: async (modelID: string) => {
      if (!provider) {
        throw new Error("Provider not found");
      }

      await deleteProviderModel(provider.id, modelID);
    },
    onError: (error) => {
      setModelFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setModelFeedback({ text: "Model deleted.", tone: "success" });
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
            await createConnectionMutation.mutateAsync(values);
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

  async function handleModelTest(modelID: string) {
    if (!provider) {
      return;
    }

    setModelTests((currentState) => ({
      ...currentState,
      [modelID]: {
        error: null,
        pending: true,
        result: null,
      },
    }));

    try {
      const result = await testProviderModel(provider.id, { model: modelID });
      setModelTests((currentState) => ({
        ...currentState,
        [modelID]: {
          error: null,
          pending: false,
          result,
        },
      }));
    } catch (error) {
      setModelTests((currentState) => ({
        ...currentState,
        [modelID]: {
          error:
            typeof error === "object" &&
            error !== null &&
            "message" in error &&
            typeof error.message === "string"
              ? error.message
              : "Request failed",
          pending: false,
          result: null,
        },
      }));
    }
  }

  async function handleCreateModel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!provider) {
      return;
    }

    const modelID = modelForm.id.trim();
    const requiredPrefix = `${provider.id}/`;
    if (!modelID.startsWith(requiredPrefix) || modelID === requiredPrefix) {
      setModelFeedback({
        text: `Model ID must start with ${requiredPrefix}`,
        tone: "error",
      });
      return;
    }

    setModelFeedback(null);
    if (modelModalState.kind === "edit") {
      await updateProviderModelMutation.mutateAsync({
        modelID: modelModalState.modelId,
        values: modelForm,
      });
      return;
    }

    await createProviderModelMutation.mutateAsync(modelForm);
  }

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
            headerAction={
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge tone="info">Prefix {provider.id}/</StatusBadge>
                <Button
                  leadingIcon={<Plus className="size-[15px]" />}
                  onClick={() => {
                    setModelFeedback(null);
                    setModelForm({ description: "", id: "", name: "" });
                    setModelModalState({ kind: "create" });
                  }}
                >
                  Add model
                </Button>
              </div>
            }
            title="Available models"
            tone="solid"
          >
            <div className="space-y-5">
              {modelFeedback?.tone === "success" ? (
                <InlineAlert
                  tone={modelFeedback.tone === "success" ? "success" : "error"}
                >
                  {modelFeedback.text}
                </InlineAlert>
              ) : null}

              {provider.models.length === 0 ? (
                <EmptyState
                  body="No models are currently advertised for this provider."
                  title="No available models"
                />
              ) : (
                <div className="flex flex-wrap gap-3">
                  {provider.models.map((model) => (
                    <ModelChip
                      hasConnections={provider.connection_count > 0}
                      isDefault={model.id === provider.default_model}
                      key={model.id}
                      model={model}
                      onDelete={async () => {
                        setModelFeedback(null);
                        await deleteProviderModelMutation.mutateAsync(model.id);
                      }}
                      onEdit={() => {
                        setModelFeedback(null);
                        setModelForm({
                          description: model.description,
                          id: model.id,
                          name: model.name,
                        });
                        setModelModalState({
                          kind: "edit",
                          modelId: model.id,
                        });
                      }}
                      onTest={() => void handleModelTest(model.id)}
                      testState={modelTests[model.id] ?? null}
                    />
                  ))}
                </div>
              )}
            </div>
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

      <Modal
        onOpenChange={(open) => {
          if (!open) {
            setModelModalState({ kind: "closed" });
          }
          if (!open && modelFeedback?.tone !== "success") {
            setModelFeedback(null);
          }
        }}
        open={modelModalState.kind !== "closed"}
      >
        {provider ? (
          <ModalContent>
            <ModalPanel
              description={`${modelModalState.kind === "edit" ? "Edit this custom model for" : "Add a model to"} ${provider.name}. The model ID must use the ${provider.id}/ prefix.`}
              title={
                modelModalState.kind === "edit" ? "Edit model" : "Add model"
              }
            >
              <form
                className="space-y-4"
                onSubmit={(event) => void handleCreateModel(event)}
              >
                <Field
                  help={`Use ${provider.id}/ before the upstream model name.`}
                  label="Model ID"
                  required
                >
                  <Input
                    autoFocus
                    onChange={(event) =>
                      setModelForm((current) => ({
                        ...current,
                        id: event.target.value,
                      }))
                    }
                    placeholder={`${provider.id}/gpt-5.5`}
                    value={modelForm.id}
                  />
                </Field>
                <Field help="Optional catalog label." label="Name">
                  <Input
                    onChange={(event) =>
                      setModelForm((current) => ({
                        ...current,
                        name: event.target.value,
                      }))
                    }
                    placeholder="GPT-5.5"
                    value={modelForm.name}
                  />
                </Field>
                <Field
                  help="Optional short note for the model catalog."
                  label="Description"
                >
                  <Input
                    onChange={(event) =>
                      setModelForm((current) => ({
                        ...current,
                        description: event.target.value,
                      }))
                    }
                    placeholder="Latest routing target for this provider"
                    value={modelForm.description}
                  />
                </Field>

                {modelFeedback?.tone === "error" ? (
                  <InlineAlert tone="error">{modelFeedback.text}</InlineAlert>
                ) : null}

                <ModalFooter>
                  <Button
                    onClick={() => setModelModalState({ kind: "closed" })}
                    tone="secondary"
                    type="button"
                  >
                    Cancel
                  </Button>
                  <Button
                    disabled={
                      createProviderModelMutation.isPending ||
                      updateProviderModelMutation.isPending
                    }
                    leadingIcon={<Plus className="size-[15px]" />}
                    type="submit"
                  >
                    {modelModalState.kind === "edit"
                      ? updateProviderModelMutation.isPending
                        ? "Saving..."
                        : "Save model"
                      : createProviderModelMutation.isPending
                        ? "Adding..."
                        : "Add model"}
                  </Button>
                </ModalFooter>
              </form>
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
  hasConnections,
  isDefault,
  model,
  onDelete,
  onEdit,
  onTest,
  testState,
}: {
  hasConnections: boolean;
  isDefault: boolean;
  model: ProviderItem["models"][number];
  onDelete: () => void;
  onEdit: () => void;
  onTest: () => void;
  testState: ModelTestState | null;
}) {
  const isTesting = testState?.pending === true;
  const isDisabled = isTesting || !hasConnections;
  const isCustom = model.source === "custom";

  return (
    <div className="border-border/85 bg-bg-primary/72 flex max-w-full min-w-[240px] flex-col gap-3 rounded-[18px] border px-3.5 py-3">
      <div className="flex items-center gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-fg-primary truncate text-sm font-semibold">
            {model.name}
          </p>
          <p className="text-fg-secondary truncate text-sm">{model.id}</p>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {isDefault ? <StatusBadge tone="success">Default</StatusBadge> : null}
          <StatusBadge tone={isCustom ? "warning" : "info"}>
            {isCustom ? "Custom" : "System"}
          </StatusBadge>
          <TooltipProvider delayDuration={0}>
            <Tooltip>
              <TooltipTrigger asChild>
                <span
                  aria-disabled={isDisabled}
                  className="inline-flex"
                  tabIndex={isDisabled ? 0 : -1}
                >
                  <Button
                    disabled={isDisabled}
                    leadingIcon={<Play className="size-[15px]" />}
                    onClick={onTest}
                    tone="secondary"
                  >
                    {isTesting ? "Testing..." : "Test"}
                  </Button>
                </span>
              </TooltipTrigger>
              {!hasConnections ? (
                <TooltipContent>
                  Add at least one connection to test this model.
                </TooltipContent>
              ) : null}
            </Tooltip>
          </TooltipProvider>
          {isCustom ? (
            <>
              <Button
                leadingIcon={<Pencil className="size-[15px]" />}
                onClick={onEdit}
                tone="secondary"
              >
                Edit
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    leadingIcon={<Trash2 className="size-[15px]" />}
                    tone="ghost"
                  >
                    Delete
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Delete model?</AlertDialogTitle>
                    <AlertDialogDescription>
                      Delete custom model "{model.name}" from this provider.
                      System models are left untouched.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancelButton>
                      Keep model
                    </AlertDialogCancelButton>
                    <AlertDialogActionButton onClick={onDelete} tone="primary">
                      Confirm delete
                    </AlertDialogActionButton>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </>
          ) : null}
        </div>
      </div>

      {testState?.error ? (
        <InlineAlert tone="error">{testState.error}</InlineAlert>
      ) : null}

      {testState?.result ? (
        <InlineAlert tone="success">
          <div className="space-y-1">
            <p>{testState.result.message}</p>
            {testState.result.output_text ? (
              <p className="text-xs break-words opacity-80">
                {testState.result.output_text}
              </p>
            ) : null}
          </div>
        </InlineAlert>
      ) : null}
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

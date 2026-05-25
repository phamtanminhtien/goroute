import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Pencil, Play, Plus, RotateCcw, Trash2 } from "lucide-react";
import {
  type Dispatch,
  type FormEvent,
  type SetStateAction,
  useState,
} from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  completeOAuthConnection,
  type ConnectionPayload,
  createConnection,
  createProviderModel,
  defaultProviderRuntimeSettings,
  deleteConnection,
  deleteProvider,
  deleteProviderModel,
  listProviders,
  type ProviderConnection,
  type ProviderItem,
  type ProviderModelTestResult,
  type ProviderPayload,
  type ProviderRuntimeSettings,
  type ProviderRuntimeSettingsPayload,
  providersQueryKey,
  resetProviderRuntimeSettings,
  testProviderModel,
  updateConnection,
  updateProvider,
  updateProviderConnectionsEnabled,
  updateProviderModel,
  updateProviderRuntimeSettings,
} from "@/features/providers/api";
import {
  type ConnectionFormFeedback,
  type ConnectionFormValues,
  emptyConnectionFormValues,
  getProviderConnectionFormEntry,
} from "@/features/providers/connection-form-registry";
import { ConnectionRuntimeStatus } from "@/features/providers/connection-runtime-status";
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
import { Switch } from "@/shared/ui/switch";
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
  inputPricePerMillionUSD: string;
  name: string;
  outputPricePerMillionUSD: string;
};
type ProviderFormState = {
  apiKey: string;
  baseURL: string;
  defaultModel: string;
  enabled: boolean;
  name: string;
};
type RuntimeSettingsFormState = Record<keyof ProviderRuntimeSettings, string>;

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
    inputPricePerMillionUSD: "",
    name: "",
    outputPricePerMillionUSD: "",
  });
  const [modelFeedback, setModelFeedback] = useState<FeedbackState>(null);
  const [modelModalState, setModelModalState] = useState<ModelModalState>({
    kind: "closed",
  });
  const [providerFormOpen, setProviderFormOpen] = useState(false);
  const [providerForm, setProviderForm] = useState<ProviderFormState>({
    apiKey: "",
    baseURL: "",
    defaultModel: "",
    enabled: true,
    name: "",
  });
  const [providerFeedback, setProviderFeedback] = useState<FeedbackState>(null);
  const [runtimeSettingsDraft, setRuntimeSettingsDraft] =
    useState<RuntimeSettingsFormState | null>(null);
  const [runtimeSettingsFeedback, setRuntimeSettingsFeedback] =
    useState<FeedbackState>(null);

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
  const isCustomProvider = provider?.category === "custom";
  const managedConnection = isCustomProvider
    ? (provider?.connections[0] ?? null)
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

  const updateProviderConnectionsEnabledMutation = useMutation({
    mutationFn: ({
      enabled,
      providerID,
    }: {
      enabled: boolean;
      providerID: string;
    }) => updateProviderConnectionsEnabled(providerID, { enabled }),
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
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
        input_price_per_million_usd: parseModelPrice(
          values.inputPricePerMillionUSD,
        ),
        name: values.name.trim(),
        output_price_per_million_usd: parseModelPrice(
          values.outputPricePerMillionUSD,
        ),
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
      setModelForm(emptyModelFormState());
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
        input_price_per_million_usd: parseModelPrice(
          values.inputPricePerMillionUSD,
        ),
        name: values.name.trim(),
        output_price_per_million_usd: parseModelPrice(
          values.outputPricePerMillionUSD,
        ),
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
      setModelForm(emptyModelFormState());
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

  const updateProviderMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: ProviderPayload }) =>
      updateProvider(id, payload),
    onError: (error) => {
      setProviderFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setProviderFeedback({ text: "Provider saved.", tone: "success" });
      setProviderFormOpen(false);
    },
  });

  const deleteProviderMutation = useMutation({
    mutationFn: deleteProvider,
    onError: (error) => {
      setProviderFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      navigate("/providers");
    },
  });

  const updateProviderRuntimeSettingsMutation = useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: string;
      payload: ProviderRuntimeSettingsPayload;
    }) => updateProviderRuntimeSettings(id, payload),
    onError: (error) => {
      setRuntimeSettingsFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setRuntimeSettingsDraft(null);
      setRuntimeSettingsFeedback({
        text: "Runtime settings saved.",
        tone: "success",
      });
    },
  });

  const resetProviderRuntimeSettingsMutation = useMutation({
    mutationFn: resetProviderRuntimeSettings,
    onError: (error) => {
      setRuntimeSettingsFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setRuntimeSettingsDraft(null);
      setRuntimeSettingsFeedback({
        text: "Runtime settings reset to defaults.",
        tone: "success",
      });
    },
  });

  function openProviderForm() {
    if (!provider) {
      return;
    }
    setProviderFeedback(null);
    setProviderForm({
      apiKey: "",
      baseURL: provider.base_url ?? "",
      defaultModel: provider.default_model,
      enabled: managedConnection?.enabled ?? true,
      name: provider.name,
    });
    setProviderFormOpen(true);
  }

  async function handleProviderUpdate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!provider) {
      return;
    }
    const payload: ProviderPayload = {
      adapter_type: provider.adapter_type ?? "openai_compatible",
      base_url: providerForm.baseURL.trim(),
      default_model: providerForm.defaultModel.trim(),
      enabled: providerForm.enabled,
      id: provider.id,
      name: providerForm.name.trim(),
    };
    if (providerForm.apiKey.trim()) {
      payload.api_key = providerForm.apiKey.trim();
    }
    setProviderFeedback(null);
    await updateProviderMutation.mutateAsync({ id: provider.id, payload });
  }

  async function handleRuntimeSettingsUpdate(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();
    if (!provider) {
      return;
    }

    try {
      setRuntimeSettingsFeedback(null);
      await updateProviderRuntimeSettingsMutation.mutateAsync({
        id: provider.id,
        payload: parseRuntimeSettingsForm(
          runtimeSettingsDraft ??
            runtimeSettingsToForm(providerRuntimeSettings(provider)),
        ),
      });
    } catch (error) {
      if (
        error instanceof Error &&
        !updateProviderRuntimeSettingsMutation.error
      ) {
        setRuntimeSettingsFeedback({ text: error.message, tone: "error" });
      }
    }
  }

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
              payload: buildConnectionPayload(
                provider.id,
                values,
                editingConnection.enabled,
              ),
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
            description={
              isCustomProvider
                ? "Custom provider credentials are managed together with the provider."
                : "Create, edit, and remove redacted provider connections without exposing stored secrets."
            }
            headerAction={
              isCustomProvider ? (
                <div className="flex flex-wrap items-center gap-2">
                  <Button
                    leadingIcon={<Pencil className="size-[15px]" />}
                    onClick={openProviderForm}
                  >
                    Edit provider
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        disabled={deleteProviderMutation.isPending}
                        leadingIcon={<Trash2 className="size-[15px]" />}
                        tone="ghost"
                      >
                        {deleteProviderMutation.isPending
                          ? "Deleting..."
                          : "Delete"}
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete provider?</AlertDialogTitle>
                        <AlertDialogDescription>
                          Delete "{provider.name}" and its managed connection.
                          This cannot be undone.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancelButton>
                          Keep provider
                        </AlertDialogCancelButton>
                        <AlertDialogActionButton
                          onClick={() =>
                            void deleteProviderMutation.mutateAsync(provider.id)
                          }
                          tone="primary"
                        >
                          Confirm delete
                        </AlertDialogActionButton>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ) : (
                <div className="flex flex-wrap items-center gap-3">
                  <Button
                    leadingIcon={<Plus className="size-[15px]" />}
                    onClick={() => {
                      setFeedback(null);
                      setModalState({
                        kind: "create",
                        providerId: provider.id,
                      });
                    }}
                  >
                    Add connection
                  </Button>
                  <div className="flex items-center gap-2">
                    <span className="text-fg-secondary text-xs font-semibold">
                      Provider
                    </span>
                    <Switch
                      aria-label={`Enable ${provider.name} connections`}
                      checked={enabledConnectionCount(provider) > 0}
                      disabled={
                        updateProviderConnectionsEnabledMutation.isPending
                      }
                      onCheckedChange={(enabled) =>
                        updateProviderConnectionsEnabledMutation.mutate({
                          enabled,
                          providerID: provider.id,
                        })
                      }
                    />
                  </div>
                </div>
              )
            }
            title="Connections"
            tone="solid"
          >
            <div className="space-y-5">
              <div className="flex flex-wrap items-center gap-3">
                <StatusBadge tone="info">
                  {buildConnectionStatus(
                    enabledConnectionCount(provider),
                    provider.connection_count,
                  )}
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

              {providerFeedback ? (
                <InlineAlert
                  tone={
                    providerFeedback.tone === "success" ? "success" : "error"
                  }
                >
                  {providerFeedback.text}
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
                      locked={isCustomProvider}
                      busy={
                        deleteConnectionMutation.isPending &&
                        deleteConnectionMutation.variables === connection.id
                      }
                      connection={connection}
                      toggling={
                        updateConnectionMutation.isPending &&
                        updateConnectionMutation.variables?.id === connection.id
                      }
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
                      onToggleEnabled={async (enabled) => {
                        setFeedback(null);
                        await updateConnectionMutation.mutateAsync({
                          id: connection.id,
                          payload: buildConnectionTogglePayload(
                            connection,
                            enabled,
                          ),
                        });
                      }}
                    />
                  ))}
                </div>
              )}
            </div>
          </SectionCard>

          <SectionCard
            description="Provider-specific upstream request timing, timeout retry, and retryable error cooldown."
            title="Runtime settings"
            tone="solid"
          >
            <form
              className="space-y-5"
              noValidate
              onSubmit={(event) => void handleRuntimeSettingsUpdate(event)}
            >
              {runtimeSettingsFeedback ? (
                <InlineAlert
                  tone={
                    runtimeSettingsFeedback.tone === "success"
                      ? "success"
                      : "error"
                  }
                >
                  {runtimeSettingsFeedback.text}
                </InlineAlert>
              ) : null}

              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                <RuntimeSettingsInput
                  help="Default 10000 ms."
                  label="Dial timeout"
                  name="dial_timeout_ms"
                  provider={provider}
                  setDraft={setRuntimeSettingsDraft}
                  value={
                    (
                      runtimeSettingsDraft ??
                      runtimeSettingsToForm(providerRuntimeSettings(provider))
                    ).dial_timeout_ms
                  }
                />
                <RuntimeSettingsInput
                  help="Default 10000 ms."
                  label="TLS handshake timeout"
                  name="tls_handshake_timeout_ms"
                  provider={provider}
                  setDraft={setRuntimeSettingsDraft}
                  value={
                    (
                      runtimeSettingsDraft ??
                      runtimeSettingsToForm(providerRuntimeSettings(provider))
                    ).tls_handshake_timeout_ms
                  }
                />
                <RuntimeSettingsInput
                  help="Default 30000 ms."
                  label="Response header timeout"
                  name="response_header_timeout_ms"
                  provider={provider}
                  setDraft={setRuntimeSettingsDraft}
                  value={
                    (
                      runtimeSettingsDraft ??
                      runtimeSettingsToForm(providerRuntimeSettings(provider))
                    ).response_header_timeout_ms
                  }
                />
                <RuntimeSettingsInput
                  help="Default 3 retries."
                  label="Timeout retry count"
                  min={0}
                  name="timeout_retry_count"
                  provider={provider}
                  setDraft={setRuntimeSettingsDraft}
                  value={
                    (
                      runtimeSettingsDraft ??
                      runtimeSettingsToForm(providerRuntimeSettings(provider))
                    ).timeout_retry_count
                  }
                />
                <RuntimeSettingsInput
                  help="Default 60000 ms."
                  label="Retryable cooldown"
                  name="retryable_cooldown_ms"
                  provider={provider}
                  setDraft={setRuntimeSettingsDraft}
                  value={
                    (
                      runtimeSettingsDraft ??
                      runtimeSettingsToForm(providerRuntimeSettings(provider))
                    ).retryable_cooldown_ms
                  }
                />
              </div>

              <div className="flex flex-wrap gap-3">
                <Button
                  disabled={
                    updateProviderRuntimeSettingsMutation.isPending ||
                    runtimeSettingsDraft === null
                  }
                  type="submit"
                >
                  Save runtime settings
                </Button>
                <Button
                  disabled={resetProviderRuntimeSettingsMutation.isPending}
                  leadingIcon={<RotateCcw className="size-[15px]" />}
                  onClick={() => {
                    setRuntimeSettingsFeedback(null);
                    void resetProviderRuntimeSettingsMutation.mutateAsync(
                      provider.id,
                    );
                  }}
                  tone="secondary"
                  type="button"
                >
                  {resetProviderRuntimeSettingsMutation.isPending
                    ? "Resetting..."
                    : "Reset to defaults"}
                </Button>
              </div>
            </form>
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
                    setModelForm(emptyModelFormState());
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
                      hasConnections={enabledConnectionCount(provider) > 0}
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
                          inputPricePerMillionUSD: formatModelPrice(
                            model.input_price_per_million_usd,
                          ),
                          name: model.name,
                          outputPricePerMillionUSD: formatModelPrice(
                            model.output_price_per_million_usd,
                          ),
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
          setProviderFormOpen(open);
          if (!open && providerFeedback?.tone !== "success") {
            setProviderFeedback(null);
          }
        }}
        open={providerFormOpen}
      >
        {provider ? (
          <ModalContent>
            <ModalPanel
              description="Update this custom provider and its managed connection together."
              title="Edit custom provider"
            >
              <form
                className="space-y-4"
                onSubmit={(event) => void handleProviderUpdate(event)}
              >
                <Field label="Name" required>
                  <Input
                    autoFocus
                    onChange={(event) =>
                      setProviderForm((current) => ({
                        ...current,
                        name: event.target.value,
                      }))
                    }
                    value={providerForm.name}
                  />
                </Field>
                <Field label="Base URL" required>
                  <Input
                    onChange={(event) =>
                      setProviderForm((current) => ({
                        ...current,
                        baseURL: event.target.value,
                      }))
                    }
                    value={providerForm.baseURL}
                  />
                </Field>
                <Field label="Default model" required>
                  <Input
                    onChange={(event) =>
                      setProviderForm((current) => ({
                        ...current,
                        defaultModel: event.target.value,
                      }))
                    }
                    value={providerForm.defaultModel}
                  />
                </Field>
                <Field
                  help="Leave blank to keep the current key."
                  label="API key"
                >
                  <Input
                    onChange={(event) =>
                      setProviderForm((current) => ({
                        ...current,
                        apiKey: event.target.value,
                      }))
                    }
                    placeholder="Enter a new API key"
                    type="password"
                    value={providerForm.apiKey}
                  />
                </Field>
                <div className="flex items-center gap-2">
                  <Switch
                    aria-label={`Enable ${provider.name}`}
                    checked={providerForm.enabled}
                    onCheckedChange={(enabled) =>
                      setProviderForm((current) => ({ ...current, enabled }))
                    }
                  />
                  <span className="text-fg-secondary text-xs font-semibold">
                    Enabled
                  </span>
                </div>

                {providerFeedback?.tone === "error" ? (
                  <InlineAlert tone="error">
                    {providerFeedback.text}
                  </InlineAlert>
                ) : null}

                <ModalFooter>
                  <Button
                    onClick={() => setProviderFormOpen(false)}
                    tone="secondary"
                    type="button"
                  >
                    Cancel
                  </Button>
                  <Button
                    disabled={updateProviderMutation.isPending}
                    type="submit"
                  >
                    {updateProviderMutation.isPending
                      ? "Saving..."
                      : "Save provider"}
                  </Button>
                </ModalFooter>
              </form>
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
                <div className="grid gap-4 sm:grid-cols-2">
                  <Field
                    help="USD per 1M input tokens. Leave blank for 0."
                    label="Input price"
                  >
                    <Input
                      min="0"
                      onChange={(event) =>
                        setModelForm((current) => ({
                          ...current,
                          inputPricePerMillionUSD: event.target.value,
                        }))
                      }
                      placeholder="0"
                      step="0.000001"
                      type="number"
                      value={modelForm.inputPricePerMillionUSD}
                    />
                  </Field>
                  <Field
                    help="USD per 1M output tokens. Leave blank for 0."
                    label="Output price"
                  >
                    <Input
                      min="0"
                      onChange={(event) =>
                        setModelForm((current) => ({
                          ...current,
                          outputPricePerMillionUSD: event.target.value,
                        }))
                      }
                      placeholder="0"
                      step="0.000001"
                      type="number"
                      value={modelForm.outputPricePerMillionUSD}
                    />
                  </Field>
                </div>

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
  locked,
  onDelete,
  onEdit,
  onToggleEnabled,
  toggling,
}: {
  busy: boolean;
  connection: ProviderConnection;
  isEditing: boolean;
  locked: boolean;
  onDelete: () => void;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  toggling: boolean;
}) {
  return (
    <CardActionRow
      actions={
        <div className="flex items-center">
          {locked ? (
            <StatusBadge tone="info">Managed by provider</StatusBadge>
          ) : (
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
                      Delete connection "{connection.name}" from this provider.
                      This removes the stored credential reference and cannot be
                      undone.
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
              <Switch
                aria-label={`Enable ${connection.name}`}
                checked={connection.enabled ?? true}
                disabled={toggling}
                onCheckedChange={onToggleEnabled}
              />
            </>
          )}
        </div>
      }
      description={
        <div className="space-y-2">
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
            {connection.enabled === false ? (
              <StatusBadge tone="warning">Disabled</StatusBadge>
            ) : null}
          </div>
          <ConnectionRuntimeStatus connection={connection} />
        </div>
      }
      title={connection.name}
    />
  );
}

function RuntimeSettingsInput({
  help,
  label,
  min = 1,
  name,
  provider,
  setDraft,
  value,
}: {
  help: string;
  label: string;
  min?: number;
  name: keyof ProviderRuntimeSettings;
  provider: ProviderItem;
  setDraft: Dispatch<SetStateAction<RuntimeSettingsFormState | null>>;
  value: string;
}) {
  return (
    <Field help={help} label={label}>
      <Input
        min={min}
        onChange={(event) =>
          setDraft((current) => ({
            ...(current ??
              runtimeSettingsToForm(providerRuntimeSettings(provider))),
            [name]: event.target.value,
          }))
        }
        step="1"
        type="number"
        value={value}
      />
    </Field>
  );
}

function buildConnectionPayload(
  providerID: string,
  values: ConnectionFormValues,
  enabled = true,
): ConnectionPayload {
  const payload: ConnectionPayload = {
    enabled,
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

function runtimeSettingsToForm(
  settings: ProviderRuntimeSettings,
): RuntimeSettingsFormState {
  return {
    dial_timeout_ms: String(settings.dial_timeout_ms),
    retryable_cooldown_ms: String(settings.retryable_cooldown_ms),
    response_header_timeout_ms: String(settings.response_header_timeout_ms),
    timeout_retry_count: String(settings.timeout_retry_count),
    tls_handshake_timeout_ms: String(settings.tls_handshake_timeout_ms),
  };
}

function providerRuntimeSettings(provider: ProviderItem) {
  return provider.runtime_settings ?? defaultProviderRuntimeSettings();
}

function parseRuntimeSettingsForm(
  values: RuntimeSettingsFormState,
): ProviderRuntimeSettingsPayload {
  return {
    dial_timeout_ms: parseRuntimeSetting(
      values.dial_timeout_ms,
      "Dial timeout",
    ),
    retryable_cooldown_ms: parseRuntimeSetting(
      values.retryable_cooldown_ms,
      "Retryable cooldown",
    ),
    response_header_timeout_ms: parseRuntimeSetting(
      values.response_header_timeout_ms,
      "Response header timeout",
    ),
    timeout_retry_count: parseRuntimeSetting(
      values.timeout_retry_count,
      "Timeout retry count",
      0,
    ),
    tls_handshake_timeout_ms: parseRuntimeSetting(
      values.tls_handshake_timeout_ms,
      "TLS handshake timeout",
    ),
  };
}

function parseRuntimeSetting(value: string, label: string, min = 1) {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < min) {
    throw new Error(
      `${label} must be an integer greater than or equal to ${min}.`,
    );
  }
  return parsed;
}

function buildConnectionTogglePayload(
  connection: ProviderConnection,
  enabled: boolean,
): ConnectionPayload {
  return {
    enabled,
    id: connection.id,
    name: connection.name,
    provider_id: connection.provider_id,
  };
}

function enabledConnectionCount(provider: ProviderItem) {
  return (
    provider.enabled_connection_count ??
    provider.connections.filter((connection) => connection.enabled ?? true)
      .length
  );
}

function buildConnectionStatus(enabledCount: number, totalCount: number) {
  if (totalCount === 0) {
    return "No connections";
  }

  return `${enabledCount}/${totalCount} enabled`;
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
                    size="sm"
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
                size="sm"
              >
                Edit
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    leadingIcon={<Trash2 className="size-[15px]" />}
                    tone="ghost"
                    size="sm"
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

function emptyModelFormState(): ModelFormState {
  return {
    description: "",
    id: "",
    inputPricePerMillionUSD: "",
    name: "",
    outputPricePerMillionUSD: "",
  };
}

function formatModelPrice(value: number | undefined) {
  return typeof value === "number" && Number.isFinite(value)
    ? String(value)
    : "";
}

function parseModelPrice(value: string) {
  const trimmed = value.trim();
  if (trimmed === "") {
    return 0;
  }

  const parsed = Number(trimmed);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0;
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

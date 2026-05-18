import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowDown,
  ArrowUp,
  Boxes,
  Pencil,
  Plus,
  Trash2,
  X,
} from "lucide-react";
import { type FormEvent, useState } from "react";

import {
  createModelCombo,
  deleteModelCombo,
  listModelCombos,
  listProviders,
  type ModelCombo,
  type ModelComboPayload,
  type ProviderItem,
  providersQueryKey,
  updateModelCombo,
} from "@/features/providers/api";
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
import { Select } from "@/shared/ui/select";
import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/shared/ui/status-badge";
import { Switch } from "@/shared/ui/switch";

const modelCombosQueryKey = ["model-combos"] as const;
const anyConnectionValue = "__any_connection__";

type ComboFormTarget = {
  connectionID: string;
  enabled: boolean;
  modelID: string;
  providerID: string;
};

type ComboFormState = {
  alias: string;
  description: string;
  enabled: boolean;
  name: string;
  originalAlias?: string;
  targets: ComboFormTarget[];
};

type ModalState =
  | { kind: "closed" }
  | { kind: "create" }
  | { combo: ModelCombo; kind: "edit" };

const emptyForm: ComboFormState = {
  alias: "",
  description: "",
  enabled: true,
  name: "",
  targets: [{ connectionID: "", enabled: true, modelID: "", providerID: "" }],
};

export function CombosPage() {
  const queryClient = useQueryClient();
  const [modalState, setModalState] = useState<ModalState>({ kind: "closed" });
  const [form, setForm] = useState<ComboFormState>(emptyForm);
  const [feedback, setFeedback] = useState<{
    text: string;
    tone: "error" | "success";
  } | null>(null);

  const providersQuery = useQuery({
    queryFn: listProviders,
    queryKey: providersQueryKey,
  });
  const combosQuery = useQuery({
    queryFn: listModelCombos,
    queryKey: modelCombosQueryKey,
  });

  const providers = providersQuery.data ?? [];
  const combos = combosQuery.data ?? [];
  const isLoading = providersQuery.isPending || combosQuery.isPending;
  const loadError = providersQuery.error ?? combosQuery.error;

  const saveComboMutation = useMutation({
    mutationFn: ({
      editAlias,
      payload,
    }: {
      editAlias?: string;
      payload: ModelComboPayload;
    }) => {
      if (editAlias) {
        return updateModelCombo(editAlias, payload);
      }
      return createModelCombo(payload);
    },
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: modelCombosQueryKey });
      setFeedback({ text: "Combo saved.", tone: "success" });
      closeModal();
    },
  });

  const deleteComboMutation = useMutation({
    mutationFn: deleteModelCombo,
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: modelCombosQueryKey });
      setFeedback({ text: "Combo deleted.", tone: "success" });
    },
  });

  const toggleComboMutation = useMutation({
    mutationFn: ({ combo, enabled }: { combo: ModelCombo; enabled: boolean }) =>
      updateModelCombo(combo.alias, buildComboPayload(combo, enabled)),
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: modelCombosQueryKey });
    },
  });

  function openCreateModal() {
    setFeedback(null);
    setForm(emptyForm);
    setModalState({ kind: "create" });
  }

  function openEditModal(combo: ModelCombo) {
    setFeedback(null);
    setForm({
      alias: combo.alias,
      description: combo.description,
      enabled: combo.enabled ?? true,
      name: combo.name,
      originalAlias: combo.alias,
      targets: combo.targets.map((target) => ({
        connectionID: target.connection_id ?? "",
        enabled: target.enabled,
        modelID: target.model_id,
        providerID: target.provider_id,
      })),
    });
    setModalState({ combo, kind: "edit" });
  }

  function closeModal() {
    setModalState({ kind: "closed" });
    setForm(emptyForm);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await saveForm();
  }

  async function saveForm() {
    const validationError = validateComboForm(form);
    if (validationError) {
      setFeedback({ text: validationError, tone: "error" });
      return;
    }

    setFeedback(null);
    await saveComboMutation.mutateAsync({
      editAlias: form.originalAlias,
      payload: {
        alias: form.alias.trim(),
        description: form.description.trim(),
        enabled: form.enabled,
        name: form.name.trim(),
        targets: form.targets.map((target) => ({
          enabled: target.enabled,
          model_id: target.modelID,
          provider_id: target.providerID,
          ...(target.connectionID
            ? { connection_id: target.connectionID }
            : {}),
        })),
      },
    });
  }

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Create model aliases that try ordered provider/model targets with fallback."
        eyebrow="Routing"
        title="Model combos"
      >
        <Button
          leadingIcon={<Plus className="size-[15px]" />}
          onClick={openCreateModal}
        >
          Add combo
        </Button>
      </PageHeader>

      {feedback && modalState.kind === "closed" ? (
        <InlineAlert tone={feedback.tone}>{feedback.text}</InlineAlert>
      ) : null}

      {isLoading ? (
        <SectionCard
          description="Loading combo aliases and provider models."
          title="Loading combos"
          tone="solid"
        >
          <Skeleton className="min-h-[220px]" />
        </SectionCard>
      ) : null}

      {loadError ? (
        <SectionCard
          description="The combo registry could not be loaded from the admin API."
          title="Combo registry unavailable"
          tone="solid"
        >
          <InlineAlert tone="error">
            {loadError instanceof Error ? loadError.message : "Request failed"}
          </InlineAlert>
        </SectionCard>
      ) : null}

      {!isLoading && !loadError ? (
        <SectionCard
          description="Aliases are exact matches and take precedence over provider-prefix routing."
          headerAction={
            <StatusBadge tone="info">{combos.length} combos</StatusBadge>
          }
          title="Combo aliases"
          tone="solid"
        >
          {combos.length === 0 ? (
            <EmptyState
              body="Create a combo alias to route one model name across multiple provider targets."
              title="No combos"
            />
          ) : (
            <div className="space-y-3">
              {combos.map((combo) => (
                <ComboRow
                  combo={combo}
                  deleting={
                    deleteComboMutation.isPending &&
                    deleteComboMutation.variables === combo.alias
                  }
                  key={combo.alias}
                  onDelete={() => deleteComboMutation.mutate(combo.alias)}
                  onEdit={() => openEditModal(combo)}
                  onToggleEnabled={(enabled) =>
                    toggleComboMutation.mutate({ combo, enabled })
                  }
                  providers={providers}
                  toggling={
                    toggleComboMutation.isPending &&
                    toggleComboMutation.variables?.combo.alias === combo.alias
                  }
                />
              ))}
            </div>
          )}
        </SectionCard>
      ) : null}

      <Modal
        onOpenChange={(open) => {
          if (!open) {
            closeModal();
          }
        }}
        open={modalState.kind !== "closed"}
      >
        <ModalContent className="flex max-h-[88vh] w-[min(94vw,1440px)] flex-col">
          <ModalPanel
            description="Targets are attempted from top to bottom using the same retry eligibility as provider fallback."
            title={modalState.kind === "edit" ? "Edit combo" : "Add combo"}
          >
            <form
              aria-label="Combo form"
              className="flex max-h-[calc(88vh-120px)] flex-col"
              onSubmit={(event) => void handleSubmit(event)}
            >
              <div className="grid min-h-0 flex-1 gap-5 lg:grid-cols-[minmax(280px,0.9fr)_minmax(420px,1.4fr)]">
                <div className="min-h-0 space-y-4 pr-1">
                  <Field label="Alias" required>
                    <Input
                      autoFocus
                      disabled={modalState.kind === "edit"}
                      onChange={(event) =>
                        setForm((current) => ({
                          ...current,
                          alias: event.target.value,
                        }))
                      }
                      placeholder="combo/fast"
                      value={form.alias}
                    />
                  </Field>
                  <Field help="Optional display label." label="Name">
                    <Input
                      onChange={(event) =>
                        setForm((current) => ({
                          ...current,
                          name: event.target.value,
                        }))
                      }
                      placeholder="Fast Combo"
                      value={form.name}
                    />
                  </Field>
                  <Field help="Optional catalog note." label="Description">
                    <Input
                      onChange={(event) =>
                        setForm((current) => ({
                          ...current,
                          description: event.target.value,
                        }))
                      }
                      placeholder="Try low-latency targets first"
                      value={form.description}
                    />
                  </Field>
                  <div className="border-border/70 bg-bg-primary/60 flex items-center justify-between gap-3 rounded-[14px] border px-3 py-2">
                    <div className="min-w-0">
                      <p className="text-fg-primary text-sm font-medium">
                        Enabled
                      </p>
                      <p className="text-fg-secondary text-xs">
                        Disabled combos are hidden from model routing.
                      </p>
                    </div>
                    <Switch
                      aria-label="Enable combo"
                      checked={form.enabled}
                      onCheckedChange={(enabled) =>
                        setForm((current) => ({ ...current, enabled }))
                      }
                    />
                  </div>

                  {feedback && modalState.kind !== "closed" ? (
                    <InlineAlert tone={feedback.tone}>
                      {feedback.text}
                    </InlineAlert>
                  ) : null}
                </div>

                <div className="flex min-h-0 flex-1 flex-col space-y-3 pr-1">
                  <div className="flex items-center justify-between gap-3">
                    <h3 className="text-fg-primary text-sm font-semibold">
                      Targets
                    </h3>
                    <Button
                      leadingIcon={<Plus className="size-[15px]" />}
                      onClick={() =>
                        setForm((current) => ({
                          ...current,
                          targets: [
                            ...current.targets,
                            {
                              connectionID: "",
                              enabled: true,
                              modelID: "",
                              providerID: "",
                            },
                          ],
                        }))
                      }
                      tone="secondary"
                      type="button"
                    >
                      Add target
                    </Button>
                  </div>

                  <div className="min-h-0 space-y-3 overflow-y-auto pr-1">
                    {form.targets.map((target, index) => (
                      <ComboTargetEditor
                        index={index}
                        key={index}
                        onChange={(nextTarget) =>
                          setForm((current) => ({
                            ...current,
                            targets: current.targets.map((item, itemIndex) =>
                              itemIndex === index ? nextTarget : item,
                            ),
                          }))
                        }
                        onMoveDown={() =>
                          setForm((current) => ({
                            ...current,
                            targets: moveTarget(
                              current.targets,
                              index,
                              index + 1,
                            ),
                          }))
                        }
                        onMoveUp={() =>
                          setForm((current) => ({
                            ...current,
                            targets: moveTarget(
                              current.targets,
                              index,
                              index - 1,
                            ),
                          }))
                        }
                        onRemove={() =>
                          setForm((current) => ({
                            ...current,
                            targets: current.targets.filter(
                              (_item, itemIndex) => itemIndex !== index,
                            ),
                          }))
                        }
                        providers={providers}
                        showEnabledToggle={modalState.kind === "edit"}
                        target={target}
                        total={form.targets.length}
                      />
                    ))}
                  </div>
                </div>
              </div>

              <ModalFooter className="border-border/70 mt-4 border-t pt-4">
                <Button onClick={closeModal} tone="secondary" type="button">
                  Cancel
                </Button>
                <Button
                  disabled={saveComboMutation.isPending}
                  leadingIcon={<Boxes className="size-[15px]" />}
                  onClick={() => void saveForm()}
                  type="button"
                >
                  {saveComboMutation.isPending ? "Saving..." : "Save combo"}
                </Button>
              </ModalFooter>
            </form>
          </ModalPanel>
        </ModalContent>
      </Modal>
    </section>
  );
}

function ComboRow({
  combo,
  deleting,
  onDelete,
  onEdit,
  onToggleEnabled,
  providers,
  toggling,
}: {
  combo: ModelCombo;
  deleting: boolean;
  onDelete: () => void;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  providers: ProviderItem[];
  toggling: boolean;
}) {
  const targetText = combo.targets
    .map((target) => {
      const providerName =
        providers.find((provider) => provider.id === target.provider_id)
          ?.name ?? target.provider_id;
      const connectionLabel = target.connection_id
        ? ` via ${target.connection_id}`
        : "";
      const enabledLabel = target.enabled ? "" : " (disabled)";
      return `${providerName}: ${target.model_id}${connectionLabel}${enabledLabel}`;
    })
    .join(" -> ");

  return (
    <CardActionRow
      actions={
        <div className="flex items-center">
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
                disabled={deleting}
                leadingIcon={<Trash2 className="size-[15px]" />}
                tone="ghost"
              >
                {deleting ? "Deleting..." : "Delete"}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete combo?</AlertDialogTitle>
                <AlertDialogDescription>
                  Delete combo "{combo.alias}" from model routing.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancelButton>Keep combo</AlertDialogCancelButton>
                <AlertDialogActionButton onClick={onDelete} tone="primary">
                  Confirm delete
                </AlertDialogActionButton>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          <Switch
            aria-label={`Enable ${combo.alias}`}
            checked={combo.enabled ?? true}
            disabled={toggling}
            onCheckedChange={onToggleEnabled}
          />
        </div>
      }
      description={
        <div className="space-y-2">
          <div className="flex flex-wrap gap-2">
            <StatusBadge tone="info">
              {combo.targets.filter((target) => target.enabled).length}/
              {combo.targets.length} enabled
            </StatusBadge>
            {combo.enabled === false ? (
              <StatusBadge tone="warning">Disabled</StatusBadge>
            ) : null}
            {combo.description ? (
              <StatusBadge tone="info">{combo.description}</StatusBadge>
            ) : null}
          </div>
          <p className="text-fg-secondary text-xs break-words">{targetText}</p>
        </div>
      }
      title={`${combo.name || combo.alias} · ${combo.alias}`}
    />
  );
}

function buildComboPayload(
  combo: ModelCombo,
  enabled = combo.enabled ?? true,
): ModelComboPayload {
  return {
    alias: combo.alias,
    description: combo.description,
    enabled,
    name: combo.name,
    targets: combo.targets.map((target) => ({
      enabled: target.enabled,
      model_id: target.model_id,
      provider_id: target.provider_id,
      ...(target.connection_id ? { connection_id: target.connection_id } : {}),
    })),
  };
}

function ComboTargetEditor({
  index,
  onChange,
  onMoveDown,
  onMoveUp,
  onRemove,
  providers,
  showEnabledToggle,
  target,
  total,
}: {
  index: number;
  onChange: (target: ComboFormTarget) => void;
  onMoveDown: () => void;
  onMoveUp: () => void;
  onRemove: () => void;
  providers: ProviderItem[];
  showEnabledToggle: boolean;
  target: ComboFormTarget;
  total: number;
}) {
  const providerOptions = providers.map((provider) => ({
    label: provider.name,
    value: provider.id,
  }));
  const selectedProvider = providers.find(
    (provider) => provider.id === target.providerID,
  );
  const modelOptions = (selectedProvider?.models ?? []).map((model) => ({
    label: model.name ? `${model.name} (${model.id})` : model.id,
    value: model.id,
  }));
  const connectionOptions = [
    { label: "Any connection", value: anyConnectionValue },
    ...(selectedProvider?.connections ?? []).map((connection) => ({
      label: connection.enabled
        ? connection.name
        : `${connection.name} (disabled)`,
      value: connection.id,
    })),
  ];

  return (
    <div className="border-border/85 bg-bg-primary/72 grid gap-3 rounded-[18px] border p-3 md:grid-cols-[1fr_1fr_1fr_auto] md:items-end">
      <Field label={`Provider ${index + 1}`} required>
        <Select
          onValueChange={(providerID) =>
            onChange({
              ...target,
              connectionID: "",
              modelID: "",
              providerID,
            })
          }
          options={providerOptions}
          placeholder="Select provider"
          value={target.providerID}
        />
      </Field>
      <Field label="Model" required>
        <Select
          disabled={!selectedProvider}
          onValueChange={(modelID) => onChange({ ...target, modelID })}
          options={modelOptions}
          placeholder="Select model"
          value={target.modelID}
        />
      </Field>
      <Field label="Connection">
        <Select
          disabled={!selectedProvider}
          onValueChange={(connectionID) =>
            onChange({
              ...target,
              connectionID:
                connectionID === anyConnectionValue ? "" : connectionID,
            })
          }
          options={connectionOptions}
          placeholder="Any connection"
          value={target.connectionID || anyConnectionValue}
        />
      </Field>
      <div className="flex items-center gap-2">
        {showEnabledToggle ? (
          <Switch
            aria-label={`Enable target ${index + 1}`}
            checked={target.enabled}
            onCheckedChange={(enabled) => onChange({ ...target, enabled })}
          />
        ) : null}
        <Button
          aria-label={`Move target ${index + 1} up`}
          disabled={index === 0}
          iconOnly
          leadingIcon={<ArrowUp className="size-4" />}
          onClick={onMoveUp}
          tone="secondary"
          type="button"
        />
        <Button
          aria-label={`Move target ${index + 1} down`}
          disabled={index === total - 1}
          iconOnly
          leadingIcon={<ArrowDown className="size-4" />}
          onClick={onMoveDown}
          tone="secondary"
          type="button"
        />
        <Button
          aria-label={`Remove target ${index + 1}`}
          disabled={total === 1}
          iconOnly
          leadingIcon={<X className="size-4" />}
          onClick={onRemove}
          tone="ghost"
          type="button"
        />
      </div>
    </div>
  );
}

function validateComboForm(form: ComboFormState) {
  if (!form.alias.trim()) {
    return "Alias is required.";
  }
  if (form.targets.length === 0) {
    return "Add at least one target.";
  }
  if (!form.targets.some((target) => target.enabled)) {
    return "Enable at least one target.";
  }
  for (const [index, target] of form.targets.entries()) {
    if (!target.providerID) {
      return `Target ${index + 1} provider is required.`;
    }
    if (!target.modelID) {
      return `Target ${index + 1} model is required.`;
    }
  }

  return null;
}

function moveTarget<T>(items: T[], fromIndex: number, toIndex: number) {
  if (toIndex < 0 || toIndex >= items.length) {
    return items;
  }
  const nextItems = [...items];
  const [item] = nextItems.splice(fromIndex, 1);
  nextItems.splice(toIndex, 0, item);
  return nextItems;
}

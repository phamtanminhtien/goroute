import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, RefreshCcw, Trash2 } from "lucide-react";
import { useState } from "react";

import {
  type ConnectionPayload,
  connectionUsageQueryKey,
  deleteConnection,
  getConnectionUsage,
  listProviders,
  type ProviderConnection,
  providersQueryKey,
  type ProviderUsage,
  updateConnection,
} from "@/features/providers/api";
import {
  type ConnectionFormFeedback,
  emptyConnectionFormValues,
  getProviderConnectionFormEntry,
} from "@/features/providers/connection-form-registry";
import { cn } from "@/shared/lib/cn";
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
import { InlineAlert } from "@/shared/ui/inline-alert";
import { Modal, ModalContent, ModalPanel } from "@/shared/ui/modal";
import { PageHeader } from "@/shared/ui/page-header";
import { Progress } from "@/shared/ui/progress";
import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/shared/ui/status-badge";
import { SurfaceCard } from "@/shared/ui/surface-card";

type FeedbackState = ConnectionFormFeedback;

type ConnectionModalState =
  | { kind: "closed" }
  | { connectionId: string; kind: "edit" };

export function CodexUsagePage() {
  const queryClient = useQueryClient();
  const [modalState, setModalState] = useState<ConnectionModalState>({
    kind: "closed",
  });
  const [feedback, setFeedback] = useState<FeedbackState>(null);

  const providersQuery = useQuery({
    queryFn: listProviders,
    queryKey: providersQueryKey,
  });

  const codexProvider = (providersQuery.data ?? []).find(
    (item) => item.id === "cx",
  );
  const connections = codexProvider?.connections ?? [];
  const editingConnection =
    modalState.kind === "edit"
      ? (connections.find((item) => item.id === modalState.connectionId) ??
        null)
      : null;
  const formRegistryEntry = codexProvider
    ? getProviderConnectionFormEntry(codexProvider.id)
    : null;

  const updateConnectionMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: ConnectionPayload }) =>
      updateConnection(id, payload),
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: async (_, variables) => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      await queryClient.invalidateQueries({
        queryKey: connectionUsageQueryKey(variables.id),
      });
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
    codexProvider && formRegistryEntry && editingConnection
      ? {
          content: formRegistryEntry.renderEdit({
            busy: updateConnectionMutation.isPending,
            connection: editingConnection,
            feedback,
            initialValues: {
              ...emptyConnectionFormValues,
              id: editingConnection.id,
              name: editingConnection.name,
            },
            onCancel: () => setModalState({ kind: "closed" }),
            onSubmit: async (values) => {
              setFeedback(null);
              await updateConnectionMutation.mutateAsync({
                id: editingConnection.id,
                payload: {
                  id: values.id.trim(),
                  name: values.name.trim(),
                  provider_id: codexProvider.id,
                  ...(values.accessToken.trim()
                    ? { access_token: values.accessToken.trim() }
                    : {}),
                  ...(values.apiKey.trim()
                    ? { api_key: values.apiKey.trim() }
                    : {}),
                  ...(values.refreshToken.trim()
                    ? { refresh_token: values.refreshToken.trim() }
                    : {}),
                },
              });
            },
            provider: codexProvider,
          }),
          description:
            formRegistryEntry.getEditMeta?.(codexProvider, editingConnection)
              .description ?? "Update this Codex connection.",
          title:
            formRegistryEntry.getEditMeta?.(codexProvider, editingConnection)
              .title ?? "Edit connection",
        }
      : null;

  return (
    <section className="space-y-4 pb-5">
      <PageHeader
        description="Live quota snapshots for your Codex connections."
        eyebrow="Quota"
        title="Codex usage"
      >
        <StatusBadge tone="info" size="sm">
          On-demand only
        </StatusBadge>
      </PageHeader>

      {feedback ? (
        <InlineAlert tone={feedback.tone === "success" ? "success" : "error"}>
          {feedback.text}
        </InlineAlert>
      ) : null}

      {providersQuery.isPending ? (
        <div className="grid gap-3 md:grid-cols-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton className="min-h-[220px] rounded-[24px]" key={index} />
          ))}
        </div>
      ) : null}

      {providersQuery.isError ? (
        <SurfaceCard className="p-4" tone="solid">
          <InlineAlert tone="error">
            {providersQuery.error instanceof Error
              ? providersQuery.error.message
              : "Request failed"}
          </InlineAlert>
        </SurfaceCard>
      ) : null}

      {!providersQuery.isPending &&
      !providersQuery.isError &&
      (!codexProvider || connections.length === 0) ? (
        <SurfaceCard className="p-5" tone="solid">
          <div className="space-y-2">
            <p className="text-fg-primary text-sm font-semibold">
              No Codex connections
            </p>
            <p className="text-fg-secondary text-sm leading-6">
              Add a Codex connection from the provider registry, then return
              here to inspect quota.
            </p>
          </div>
        </SurfaceCard>
      ) : null}

      {!providersQuery.isPending &&
      !providersQuery.isError &&
      connections.length > 0 ? (
        <div className="grid gap-3 md:grid-cols-2">
          {connections.map((connection) => (
            <CodexUsageCard
              busyDeleting={
                deleteConnectionMutation.isPending &&
                deleteConnectionMutation.variables === connection.id
              }
              connection={connection}
              key={connection.id}
              onDelete={async () => {
                setFeedback(null);
                await deleteConnectionMutation.mutateAsync(connection.id);
              }}
              onEdit={() => {
                setFeedback(null);
                setModalState({ connectionId: connection.id, kind: "edit" });
              }}
            />
          ))}
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

function CodexUsageCard({
  busyDeleting,
  connection,
  onDelete,
  onEdit,
}: {
  busyDeleting: boolean;
  connection: ProviderConnection;
  onDelete: () => void | Promise<void>;
  onEdit: () => void;
}) {
  const queryClient = useQueryClient();
  const usageQuery = useQuery({
    queryFn: () => getConnectionUsage(connection.id),
    queryKey: connectionUsageQueryKey(connection.id),
    staleTime: 60_000,
  });

  return (
    <SurfaceCard className="space-y-2.5 rounded-[20px] p-3" tone="solid">
      <div className="flex items-start gap-2.5">
        <div className="border-border/80 overflow-hidden rounded-[10px] border bg-[#080808] p-1">
          <img
            alt=""
            className="size-7 rounded-[7px] object-cover"
            src="/images/providers/cx.png"
          />
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <p className="text-fg-primary truncate text-[12px] font-semibold tracking-[-0.03em]">
                {connection.name}
              </p>
              <p className="text-fg-muted mt-0.5 truncate text-[10px]">
                {connection.id}
              </p>
            </div>

            <div className="flex shrink-0 items-center gap-0.5">
              <Button
                aria-label={`Refresh ${connection.name} usage`}
                className="min-h-0 rounded-[10px] px-1.5 py-1 text-[10px]"
                disabled={usageQuery.isFetching}
                leadingIcon={
                  <RefreshCcw
                    className={cn(
                      "size-3.5",
                      usageQuery.isFetching ? "animate-spin" : "",
                    )}
                  />
                }
                onClick={() => {
                  void usageQuery.refetch();
                }}
                ripple={false}
                size="md"
                tone="ghost"
              >
                {usageQuery.isFetching ? "Refreshing" : "Refresh"}
              </Button>
              <Button
                aria-label={`Edit ${connection.name}`}
                className="min-h-0 rounded-[10px] px-1.5 py-1 text-[10px]"
                leadingIcon={<Pencil className="size-3" />}
                onClick={onEdit}
                ripple={false}
                tone="ghost"
              >
                Edit
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    aria-label={`Delete ${connection.name}`}
                    className="min-h-0 rounded-[10px] px-1.5 py-1 text-[10px] text-rose-600 hover:bg-rose-500/10 hover:text-rose-700 dark:text-rose-300 dark:hover:bg-rose-500/12 dark:hover:text-rose-200"
                    disabled={busyDeleting}
                    leadingIcon={<Trash2 className="size-3" />}
                    ripple={false}
                    tone="ghost"
                  >
                    Delete
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Delete connection?</AlertDialogTitle>
                    <AlertDialogDescription>
                      Delete connection "{connection.name}" from Codex usage.
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
            </div>
          </div>

          {!usageQuery.isPending &&
          !usageQuery.isError &&
          usageQuery.data?.plan ? (
            <div className="mt-1.5 flex flex-wrap gap-1">
              <StatusBadge size="sm" tone="info">
                Plan {usageQuery.data.plan}
              </StatusBadge>
              <StatusBadge
                size="sm"
                tone={usageQuery.data.limitReached ? "warning" : "success"}
              >
                {usageQuery.data.limitReached
                  ? "Normal limited"
                  : "Normal ready"}
              </StatusBadge>
            </div>
          ) : null}
        </div>
      </div>

      {usageQuery.isPending ? <SessionQuotaSkeleton /> : null}

      {usageQuery.isError ? (
        <InlineAlert tone="warning">
          Usage check failed for this Codex connection.
        </InlineAlert>
      ) : null}

      {!usageQuery.isPending && !usageQuery.isError && usageQuery.data ? (
        <UsageSummary usage={usageQuery.data} />
      ) : null}
    </SurfaceCard>
  );
}

function UsageSummary({ usage }: { usage: ProviderUsage }) {
  if (usage.message) {
    return (
      <InlineAlert className="rounded-[16px] px-3 py-2 text-xs" tone="info">
        {usage.message}
      </InlineAlert>
    );
  }

  const sessionQuota = usage.quotas?.session;
  if (!sessionQuota) {
    return null;
  }

  return <SessionQuotaRow label="Session" quota={sessionQuota} />;
}

function SessionQuotaRow({
  label,
  quota,
}: {
  label: string;
  quota: NonNullable<ProviderUsage["quotas"]>[string];
}) {
  const tone = quotaTone(quota);

  return (
    <div className="border-border/60 border-t pt-2.5">
      <div className="flex items-center justify-between gap-2">
        <p className="text-fg-muted text-[10px] font-semibold tracking-[0.18em] uppercase">
          {label}
        </p>
        <p className="text-fg-muted shrink-0 text-[10px] font-medium">
          {formatCountdown(quota.resetAt)}
        </p>
      </div>

      <div className="mt-2 flex items-end justify-between gap-3">
        <div className="min-w-0">
          <p
            className={cn(
              "text-[20px] leading-none font-semibold tracking-[-0.04em]",
              tone.text,
            )}
          >
            {quota.used}%
          </p>
          <p className="text-fg-secondary mt-0.5 text-[10px]">used quota</p>
        </div>

        <div className="min-w-0 text-right">
          <p className="text-fg-primary text-[11px] leading-none font-semibold">
            {quota.unlimited ? "Unlimited" : `${quota.used}/${quota.total}`}
          </p>
          <p className="text-fg-muted mt-0.5 text-[10px]">
            {quota.unlimited ? "No cap reported" : `${quota.remaining} left`}
          </p>
        </div>
      </div>

      <div className="mt-2">
        <Progress
          className="h-1 bg-black/6 dark:bg-white/[0.06]"
          indicatorClassName={tone.progress}
          value={quota.unlimited ? 100 : quota.remaining}
        />
      </div>
    </div>
  );
}

function SessionQuotaSkeleton() {
  return (
    <div className="border-border/60 border-t pt-2.5">
      <div className="flex items-center justify-between gap-2">
        <Skeleton className="h-3 w-14 rounded-full" />
        <Skeleton className="h-3 w-16 rounded-full" />
      </div>

      <div className="mt-2 flex items-end justify-between gap-3">
        <div>
          <Skeleton className="h-7 w-14 rounded-full" />
          <Skeleton className="mt-1 h-3 w-14 rounded-full" />
        </div>
        <div className="text-right">
          <Skeleton className="h-3 w-12 rounded-full" />
          <Skeleton className="mt-1 h-3 w-10 rounded-full" />
        </div>
      </div>

      <Skeleton className="mt-2 h-1 w-full rounded-full" />
    </div>
  );
}

function quotaTone(quota: NonNullable<ProviderUsage["quotas"]>[string]) {
  if (quota.unlimited) {
    return {
      pill: "bg-primary/10 text-primary",
      progress: "bg-primary",
      text: "text-primary",
    };
  }
  if (quota.used >= 85) {
    return {
      pill: "bg-rose-500/12 text-rose-600 dark:text-rose-300",
      progress: "bg-rose-500",
      text: "text-rose-600 dark:text-rose-300",
    };
  }
  if (quota.used >= 60) {
    return {
      pill: "bg-amber-500/12 text-amber-700 dark:text-amber-300",
      progress: "bg-amber-500",
      text: "text-amber-700 dark:text-amber-300",
    };
  }

  return {
    pill: "bg-emerald-500/12 text-emerald-700 dark:text-emerald-300",
    progress: "bg-emerald-500",
    text: "text-emerald-700 dark:text-emerald-300",
  };
}

function formatCountdown(resetAt?: string) {
  if (!resetAt) {
    return "No reset";
  }

  const date = new Date(resetAt);
  if (Number.isNaN(date.getTime())) {
    return "No reset";
  }

  const diffMs = date.getTime() - Date.now();
  if (diffMs <= 0) {
    return "0m";
  }

  const totalMinutes = Math.floor(diffMs / 60_000);
  const days = Math.floor(totalMinutes / (24 * 60));
  const hours = Math.floor((totalMinutes % (24 * 60)) / 60);
  const minutes = totalMinutes % 60;

  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }

  return `${minutes}m`;
}

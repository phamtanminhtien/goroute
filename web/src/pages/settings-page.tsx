import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { DatabaseZap, Save, ServerCog } from "lucide-react";
import { type ReactNode, useState } from "react";

import {
  getSettings,
  settingsQueryKey,
  updateSettings,
  type UpdateSettingsPayload,
} from "@/features/settings/api";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/shared/ui/status-badge";
import { Switch } from "@/shared/ui/switch";

type LoggingDraft = UpdateSettingsPayload["llmLogging"]["enabled"];
type FeedbackState = null | { text: string; tone: "error" | "success" };

export function SettingsPage() {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<LoggingDraft | null>(null);
  const [feedback, setFeedback] = useState<FeedbackState>(null);

  const settingsQuery = useQuery({
    queryFn: getSettings,
    queryKey: settingsQueryKey,
  });

  const updateSettingsMutation = useMutation({
    mutationFn: updateSettings,
    onError: (error) => {
      setFeedback({
        text: error instanceof Error ? error.message : "Request failed",
        tone: "error",
      });
    },
    onSuccess: (next) => {
      queryClient.setQueryData(settingsQueryKey, next);
      setDraft(null);
      setFeedback({ text: "Logging settings saved.", tone: "success" });
    },
  });

  const current = settingsQuery.data?.llmLogging.enabled ?? null;
  const effectiveDraft = draft ?? current;
  const updateDraft = (
    updater: (currentDraft: LoggingDraft) => LoggingDraft,
  ) => {
    setDraft((currentDraft) => {
      const baseDraft = currentDraft ?? current;
      if (!baseDraft) {
        return currentDraft;
      }

      return updater(baseDraft);
    });
  };
  const hasChanges =
    draft !== null &&
    current !== null &&
    (draft.flow !== current.flow || draft.thirdParty !== current.thirdParty);

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Control how much request detail goroute persists for LLM traffic without restarting the process."
        eyebrow="Runtime"
        title="System configuration"
      >
        <StatusBadge tone="info">Live admin data</StatusBadge>
      </PageHeader>

      {feedback ? (
        <InlineAlert tone={feedback.tone === "success" ? "success" : "error"}>
          {feedback.text}
        </InlineAlert>
      ) : null}

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <SectionCard
          description="Current runtime details that frame how the logging policy applies."
          title="Runtime overview"
          tone="solid"
        >
          {settingsQuery.isPending ? (
            <div className="grid gap-3 sm:grid-cols-2">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton className="min-h-[110px]" key={index} />
              ))}
            </div>
          ) : settingsQuery.isError ? (
            <InlineAlert tone="error">
              {settingsQuery.error instanceof Error
                ? settingsQuery.error.message
                : "Request failed"}
            </InlineAlert>
          ) : settingsQuery.data ? (
            <div className="grid gap-3 sm:grid-cols-2">
              <RuntimePanel
                description="Admin HTTP bind address loaded from the current config."
                icon={<ServerCog className="size-4" />}
                label="Listen address"
                value={settingsQuery.data.server.listen}
              />
              <RuntimePanel
                description="Static admin UI directory currently configured for the local runtime."
                icon={<DatabaseZap className="size-4" />}
                label="Web UI directory"
                value={settingsQuery.data.server.web_ui_dir}
              />
              <RuntimePanel
                description="When both switches are off, only ai_request_runs remain persisted."
                icon={<DatabaseZap className="size-4" />}
                label="Flow log state"
                value={
                  settingsQuery.data.llmLogging.enabled.flow
                    ? "Enabled"
                    : "Disabled"
                }
              />
              <RuntimePanel
                description="Controls upstream attempt persistence in third_party_request_logs."
                icon={<DatabaseZap className="size-4" />}
                label="Third-party log state"
                value={
                  settingsQuery.data.llmLogging.enabled.thirdParty
                    ? "Enabled"
                    : "Disabled"
                }
              />
            </div>
          ) : null}
        </SectionCard>

        <SectionCard
          description="These switches apply immediately to new requests after save."
          title="LLM logging"
          tone="solid"
        >
          {settingsQuery.isPending ? (
            <div className="space-y-3">
              <Skeleton className="min-h-[88px]" />
              <Skeleton className="min-h-[88px]" />
              <Skeleton className="min-h-[56px]" />
            </div>
          ) : settingsQuery.isError ? (
            <div className="space-y-4">
              <InlineAlert tone="error">
                {settingsQuery.error instanceof Error
                  ? settingsQuery.error.message
                  : "Request failed"}
              </InlineAlert>
              <Button
                onClick={() => settingsQuery.refetch()}
                tone="secondary"
                type="button"
              >
                Retry request
              </Button>
            </div>
          ) : effectiveDraft ? (
            <form
              className="space-y-4"
              onSubmit={async (event) => {
                event.preventDefault();
                setFeedback(null);
                try {
                  await updateSettingsMutation.mutateAsync({
                    llmLogging: { enabled: effectiveDraft },
                  });
                } catch {
                  // onError already surfaces the failure to the user.
                }
              }}
            >
              <Field
                help="Controls persistence for ai_request_flows. Turning it off still keeps ai_request_runs."
                label="Flow log"
              >
                <div className="border-border/85 bg-bg-primary/72 flex items-center justify-between rounded-[20px] border px-4 py-3">
                  <div className="space-y-1 pr-4">
                    <p className="text-fg-primary text-sm font-semibold">
                      Persist `ai_request_flows`
                    </p>
                    <p className="text-fg-secondary text-sm leading-6">
                      Save redacted request and response flow bodies for each
                      completed request.
                    </p>
                  </div>
                  <Switch
                    aria-label="Flow log toggle"
                    checked={effectiveDraft.flow}
                    onCheckedChange={(checked) =>
                      updateDraft((currentDraft) => ({
                        ...currentDraft,
                        flow: checked,
                      }))
                    }
                  />
                </div>
              </Field>

              <Field
                help="Controls persistence for third_party_request_logs. This can stay on even when flow log is off."
                label="Third-party log"
              >
                <div className="border-border/85 bg-bg-primary/72 flex items-center justify-between rounded-[20px] border px-4 py-3">
                  <div className="space-y-1 pr-4">
                    <p className="text-fg-primary text-sm font-semibold">
                      Persist `third_party_request_logs`
                    </p>
                    <p className="text-fg-secondary text-sm leading-6">
                      Keep upstream attempt metadata and redacted raw provider
                      exchanges for each try.
                    </p>
                  </div>
                  <Switch
                    aria-label="Third-party log toggle"
                    checked={effectiveDraft.thirdParty}
                    onCheckedChange={(checked) =>
                      updateDraft((currentDraft) => ({
                        ...currentDraft,
                        thirdParty: checked,
                      }))
                    }
                  />
                </div>
              </Field>

              <div className="border-border/85 bg-bg-primary/72 rounded-[20px] border px-4 py-3">
                <p className="text-fg-primary text-sm font-semibold">
                  Persistence summary
                </p>
                <p className="text-fg-secondary mt-1 text-sm leading-6">
                  If both switches are off, goroute persists only
                  `ai_request_runs`. Changes here affect new requests right
                  after save.
                </p>
              </div>

              <div className="flex flex-wrap gap-3">
                <Button
                  disabled={!hasChanges || updateSettingsMutation.isPending}
                  type="submit"
                >
                  <Save className="size-4" />
                  {updateSettingsMutation.isPending
                    ? "Saving..."
                    : "Save settings"}
                </Button>
                <Button
                  disabled={!hasChanges || updateSettingsMutation.isPending}
                  onClick={() => {
                    setDraft(null);
                    setFeedback(null);
                  }}
                  tone="secondary"
                  type="button"
                >
                  Reset changes
                </Button>
              </div>
            </form>
          ) : null}
        </SectionCard>
      </div>
    </section>
  );
}

function RuntimePanel({
  description,
  icon,
  label,
  value,
}: {
  description: string;
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="border-border/90 bg-bg-primary/78 rounded-[24px] border px-4 py-4">
      <p className="text-fg-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.2em] uppercase">
        {icon}
        {label}
      </p>
      <p className="text-fg-primary mt-3 text-lg font-semibold tracking-tight">
        {value}
      </p>
      <p className="text-fg-secondary mt-2 text-sm leading-6">{description}</p>
    </div>
  );
}

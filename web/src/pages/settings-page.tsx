import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowUpRight,
  DatabaseZap,
  Save,
  ServerCog,
  ShieldCheck,
  Waypoints,
} from "lucide-react";
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
import { SurfaceCard } from "@/shared/ui/surface-card";
import { Switch } from "@/shared/ui/switch";

type LoggingDraft = UpdateSettingsPayload["llmLogging"]["enabled"];
type RTKDraft = UpdateSettingsPayload["rtk"];
type OpenAICompatibleAuthDraft = UpdateSettingsPayload["openAICompatibleAuth"];
type FeedbackState = null | { text: string; tone: "error" | "success" };

export function SettingsPage() {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<{
    llmLogging: LoggingDraft;
    openAICompatibleAuth: OpenAICompatibleAuthDraft;
    rtk: RTKDraft;
  } | null>(null);
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
      setFeedback({
        text: "Settings saved. New requests will use them right away.",
        tone: "success",
      });
    },
  });

  const current = settingsQuery.data
    ? {
        llmLogging: settingsQuery.data.llmLogging.enabled,
        openAICompatibleAuth: settingsQuery.data.openAICompatibleAuth,
        rtk: settingsQuery.data.rtk,
      }
    : null;
  const effectiveDraft = draft ?? current;
  const updateDraft = (
    updater: (
      currentDraft: NonNullable<typeof effectiveDraft>,
    ) => NonNullable<typeof effectiveDraft>,
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
    (draft.llmLogging.flow !== current.llmLogging.flow ||
      draft.llmLogging.thirdParty !== current.llmLogging.thirdParty ||
      draft.openAICompatibleAuth.enabled !==
        current.openAICompatibleAuth.enabled ||
      draft.rtk.enabled !== current.rtk.enabled);

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Choose how much request detail to keep for troubleshooting and whether RTK should reduce noisy payloads before sending them upstream."
        eyebrow="Runtime"
        title="Request settings"
      >
        <StatusBadge tone="info">Applies without restart</StatusBadge>
      </PageHeader>

      {feedback ? (
        <InlineAlert tone={feedback.tone === "success" ? "success" : "error"}>
          {feedback.text}
        </InlineAlert>
      ) : null}

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <div className="space-y-6">
          <SectionCard
            description="A quick summary of what goroute will keep for new requests."
            title="At a glance"
            tone="solid"
          >
            {settingsQuery.isPending ? (
              <div className="grid gap-3 md:grid-cols-3">
                {Array.from({ length: 3 }).map((_, index) => (
                  <Skeleton className="min-h-[118px]" key={index} />
                ))}
              </div>
            ) : settingsQuery.isError ? (
              <InlineAlert tone="error">
                {settingsQuery.error instanceof Error
                  ? settingsQuery.error.message
                  : "Request failed"}
              </InlineAlert>
            ) : settingsQuery.data ? (
              <div className="grid gap-3 md:grid-cols-3">
                <StatusPanel
                  description={
                    settingsQuery.data.llmLogging.enabled.flow
                      ? "Request and response details will be available for troubleshooting."
                      : "Only basic request records will be kept."
                  }
                  icon={<ShieldCheck className="size-4" />}
                  label="Request history details"
                  tone={
                    settingsQuery.data.llmLogging.enabled.flow
                      ? "success"
                      : "warning"
                  }
                  value={
                    settingsQuery.data.llmLogging.enabled.flow
                      ? "Saved"
                      : "Basic only"
                  }
                />
                <StatusPanel
                  description={
                    settingsQuery.data.llmLogging.enabled.thirdParty
                      ? "Provider-side request attempts and responses will be kept."
                      : "Provider-side exchanges will not be stored."
                  }
                  icon={<Waypoints className="size-4" />}
                  label="Provider exchange details"
                  tone={
                    settingsQuery.data.llmLogging.enabled.thirdParty
                      ? "success"
                      : "warning"
                  }
                  value={
                    settingsQuery.data.llmLogging.enabled.thirdParty
                      ? "Saved"
                      : "Not saved"
                  }
                />
                <StatusPanel
                  description={
                    settingsQuery.data.rtk.enabled
                      ? "Large machine-generated payloads may be reduced before sending upstream."
                      : "Requests are sent upstream without RTK compression."
                  }
                  icon={<DatabaseZap className="size-4" />}
                  label="RTK compression"
                  tone={settingsQuery.data.rtk.enabled ? "success" : "info"}
                  value={settingsQuery.data.rtk.enabled ? "On" : "Off"}
                />
              </div>
            ) : null}
          </SectionCard>

          <SectionCard
            description="These settings affect only new requests and take effect as soon as you save."
            title="Request detail controls"
            tone="solid"
          >
            {settingsQuery.isPending ? (
              <div className="space-y-3">
                <Skeleton className="min-h-[132px]" />
                <Skeleton className="min-h-[132px]" />
                <Skeleton className="min-h-[156px]" />
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
                      llmLogging: { enabled: effectiveDraft.llmLogging },
                      openAICompatibleAuth: effectiveDraft.openAICompatibleAuth,
                      rtk: effectiveDraft.rtk,
                    });
                  } catch {
                    // onError already surfaces the failure to the user.
                  }
                }}
              >
                <Field
                  help="Keep enough detail to inspect what happened during a request without exposing internal storage names."
                  label="Request history details"
                >
                  <SettingToggleCard
                    ariaLabel="Request history details toggle"
                    checked={effectiveDraft.llmLogging.flow}
                    description="Save request and response details for each completed request so troubleshooting is easier."
                    summaryOff="Only the basic request record is kept."
                    summaryOn="Detailed request history will be stored for new requests."
                    title="Save request and response details"
                    onCheckedChange={(checked) =>
                      updateDraft((currentDraft) => ({
                        ...currentDraft,
                        llmLogging: {
                          ...currentDraft.llmLogging,
                          flow: checked,
                        },
                      }))
                    }
                  />
                </Field>

                <Field
                  help="Useful when you need to review how upstream providers responded, without showing database terms to users."
                  label="Provider exchange details"
                >
                  <SettingToggleCard
                    ariaLabel="Provider exchange details toggle"
                    checked={effectiveDraft.llmLogging.thirdParty}
                    description="Save provider request attempts and redacted upstream responses for each try."
                    summaryOff="Provider-side exchanges will not be stored."
                    summaryOn="Provider-side exchanges will be available for troubleshooting."
                    title="Save provider interactions"
                    onCheckedChange={(checked) =>
                      updateDraft((currentDraft) => ({
                        ...currentDraft,
                        llmLogging: {
                          ...currentDraft.llmLogging,
                          thirdParty: checked,
                        },
                      }))
                    }
                  />
                </Field>

                <Field
                  help="RTK can reduce large machine-generated payloads before they are forwarded upstream."
                  label="RTK compression"
                >
                  <SettingToggleCard
                    ariaLabel="RTK compression toggle"
                    checked={effectiveDraft.rtk.enabled}
                    description="Reduce repetitive or noisy machine-generated text before goroute forwards the request."
                    extra={
                      <a
                        className="text-primary inline-flex items-center gap-1.5 text-sm font-medium hover:underline"
                        href="https://github.com/rtk-ai/rtk"
                        rel="noreferrer"
                        target="_blank"
                      >
                        Learn more about RTK
                        <ArrowUpRight className="size-3.5" />
                      </a>
                    }
                    summaryOff="Requests will be forwarded without RTK compression."
                    summaryOn="New requests may be compressed before they are sent upstream."
                    title="Reduce noisy payloads before sending"
                    onCheckedChange={(checked) =>
                      updateDraft((currentDraft) => ({
                        ...currentDraft,
                        rtk: { enabled: checked },
                      }))
                    }
                  />
                </Field>

                <SurfaceCard
                  className="border-border/80 bg-bg-primary/70 p-4"
                  tone="glass"
                >
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="text-fg-primary text-sm font-semibold">
                        What happens if everything is off?
                      </p>
                      <StatusBadge size="sm" tone="warning">
                        Basic records only
                      </StatusBadge>
                    </div>
                    <p className="text-fg-secondary text-sm leading-6">
                      If you turn off both detail options, goroute still keeps a
                      basic record for each request. You can enable RTK
                      separately at any time.
                    </p>
                  </div>
                </SurfaceCard>

                <div className="flex flex-wrap gap-3">
                  <Button
                    disabled={!hasChanges || updateSettingsMutation.isPending}
                    type="submit"
                  >
                    <Save className="size-4" />
                    {updateSettingsMutation.isPending
                      ? "Saving..."
                      : "Save changes"}
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

        <SectionCard
          description="Read-only runtime details for the admin server and current UI bundle."
          title="Current environment"
          tone="solid"
        >
          {settingsQuery.isPending ? (
            <div className="grid gap-3">
              {Array.from({ length: 5 }).map((_, index) => (
                <Skeleton className="min-h-[98px]" key={index} />
              ))}
            </div>
          ) : settingsQuery.isError ? (
            <InlineAlert tone="error">
              {settingsQuery.error instanceof Error
                ? settingsQuery.error.message
                : "Request failed"}
            </InlineAlert>
          ) : settingsQuery.data ? (
            <div className="grid gap-3">
              <RuntimePanel
                description="Where the admin server is currently listening."
                icon={<ServerCog className="size-4" />}
                label="Listen address"
                value={settingsQuery.data.server.listen}
              />
              <RuntimePanel
                description="The folder currently serving the admin interface."
                icon={<DatabaseZap className="size-4" />}
                label="Web UI directory"
                value={settingsQuery.data.server.web_ui_dir}
              />
              <RuntimePanel
                description="Current setting for saved request and response details."
                icon={<ShieldCheck className="size-4" />}
                label="Request history details"
                value={
                  settingsQuery.data.llmLogging.enabled.flow
                    ? "Saved"
                    : "Basic only"
                }
              />
              <RuntimePanel
                description="Current setting for saved provider-side request attempts."
                icon={<Waypoints className="size-4" />}
                label="Provider exchange details"
                value={
                  settingsQuery.data.llmLogging.enabled.thirdParty
                    ? "Saved"
                    : "Not saved"
                }
              />
              <RuntimePanel
                description="Current RTK request compression status."
                icon={<DatabaseZap className="size-4" />}
                label="RTK compression"
                value={settingsQuery.data.rtk.enabled ? "On" : "Off"}
              />
            </div>
          ) : null}
        </SectionCard>
      </div>
    </section>
  );
}

function SettingToggleCard({
  ariaLabel,
  checked,
  description,
  extra,
  onCheckedChange,
  summaryOff,
  summaryOn,
  title,
}: {
  ariaLabel: string;
  checked: boolean;
  description: string;
  extra?: ReactNode;
  onCheckedChange: (checked: boolean) => void;
  summaryOff: string;
  summaryOn: string;
  title: string;
}) {
  return (
    <SurfaceCard className="border-border/85 bg-bg-primary/72 p-4" tone="glass">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2 pr-4">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-fg-primary text-sm font-semibold">{title}</p>
            <StatusBadge size="sm" tone={checked ? "success" : "warning"}>
              {checked ? "On" : "Off"}
            </StatusBadge>
          </div>
          <p className="text-fg-secondary text-sm leading-6">{description}</p>
          <p className="text-fg-muted text-xs leading-5">
            {checked ? summaryOn : summaryOff}
          </p>
          {extra ? <div>{extra}</div> : null}
        </div>
        <Switch
          aria-label={ariaLabel}
          checked={checked}
          onCheckedChange={onCheckedChange}
        />
      </div>
    </SurfaceCard>
  );
}

function StatusPanel({
  description,
  icon,
  label,
  tone,
  value,
}: {
  description: string;
  icon: ReactNode;
  label: string;
  tone: "info" | "success" | "warning";
  value: string;
}) {
  return (
    <SurfaceCard className="p-4" tone="glass">
      <div className="space-y-3">
        <p className="text-fg-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.18em] uppercase">
          {icon}
          {label}
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-fg-primary text-2xl font-semibold tracking-[-0.04em]">
            {value}
          </p>
          <StatusBadge size="sm" tone={tone}>
            {value}
          </StatusBadge>
        </div>
        <p className="text-fg-secondary text-sm leading-6">{description}</p>
      </div>
    </SurfaceCard>
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

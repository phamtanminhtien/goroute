import {
  ArrowUpRight,
  Cable,
  CheckCircle2,
  KeyRound,
  Plus,
  Route,
  ShieldAlert,
} from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/shared/ui/button";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { StatusBadge } from "@/shared/ui/status-badge";

const providers = [
  {
    authMode: "Bearer token",
    driver: "opena/*",
    endpoint: "https://api.openai.com/v1",
    fallback: "1 alternate target",
    models: "gpt-4.1, gpt-4o, o4-mini",
    name: "OpenAI primary",
    status: "Healthy",
    tone: "success" as const,
    type: "OpenAI upstream",
  },
  {
    authMode: "Local token relay",
    driver: "cx/*",
    endpoint: "https://api.codex.example/v1",
    fallback: "No fallback",
    models: "gpt-5.4, gpt-5.5",
    name: "Codex control",
    status: "Needs review",
    tone: "warning" as const,
    type: "Codex upstream",
  },
];

const postureCards = [
  { label: "Registered providers", value: "02" },
  { label: "Healthy routes", value: "01" },
  { label: "Fallback chains ready", value: "01" },
  { label: "Credentials flagged", value: "01" },
];

export function ProvidersPage() {
  return (
    <section className="space-y-6">
      <PageHeader
        description="Track provider readiness, credential posture, model coverage, and fallback exposure from a familiar admin dashboard."
        eyebrow="Providers"
        title="Provider registry"
      >
        <Button leadingIcon={<Plus className="size-4" />}>Add provider</Button>
      </PageHeader>

      <SectionCard
        description="Operational summary cards stay dense and scannable so status changes are visible before you drill into the registry."
        title="Registry overview"
        tone="solid"
      >
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {postureCards.map((item) => (
            <div
              className="border-border/90 bg-bg-secondary rounded-[24px] border px-4 py-4 shadow-[var(--shadow-sm)]"
              key={item.label}
            >
              <p className="text-fg-muted text-[11px] font-semibold tracking-[0.22em] uppercase">
                {item.label}
              </p>
              <p className="text-fg-primary mt-3 text-3xl font-semibold tracking-tight">
                {item.value}
              </p>
            </div>
          ))}
        </div>
      </SectionCard>

      <div className="grid gap-6 xl:grid-cols-[1.3fr_0.7fr]">
        <SectionCard
          description="Configured providers are shown in a table-like registry so operational metadata is easier to compare line by line."
          title="Configured providers"
          tone="solid"
        >
          <div className="border-border/90 overflow-hidden rounded-[24px] border">
            <div className="bg-bg-tertiary/55 text-fg-muted hidden grid-cols-[minmax(0,1.5fr)_minmax(0,0.8fr)_minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1fr)_auto] gap-4 px-4 py-3 text-[11px] font-semibold tracking-[0.2em] uppercase xl:grid">
              <span>Provider</span>
              <span>Driver</span>
              <span>Endpoint</span>
              <span>Auth</span>
              <span>Fallback</span>
              <span>Actions</span>
            </div>
            <div className="divide-border/85 bg-bg-secondary divide-y">
              {providers.map((provider) => (
                <article
                  className="grid gap-4 px-4 py-4 xl:grid-cols-[minmax(0,1.5fr)_minmax(0,0.8fr)_minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1fr)_auto] xl:items-center"
                  key={provider.name}
                >
                  <div className="min-w-0">
                    <div className="flex items-start gap-3">
                      <span className="border-border/90 bg-bg-primary text-primary inline-flex size-10 shrink-0 items-center justify-center rounded-2xl border">
                        <Cable className="size-4" />
                      </span>
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <h2 className="text-fg-primary text-sm font-semibold">
                            {provider.name}
                          </h2>
                          <StatusBadge size="sm" tone={provider.tone}>
                            {provider.status}
                          </StatusBadge>
                        </div>
                        <p className="text-fg-secondary mt-1 text-sm">
                          {provider.type}
                        </p>
                      </div>
                    </div>
                  </div>

                  <ProviderCell label="Driver" value={provider.driver} />
                  <ProviderCell label="Endpoint" value={provider.endpoint} />
                  <ProviderCell label="Auth mode" value={provider.authMode} />
                  <ProviderCell label="Fallback" value={provider.fallback} />

                  <div className="flex flex-wrap gap-2 xl:justify-end">
                    <Button tone="secondary">Inspect</Button>
                    <Button
                      leadingIcon={<KeyRound className="size-4" />}
                      tone="secondary"
                    >
                      Rotate token
                    </Button>
                  </div>

                  <div className="xl:col-span-6">
                    <div className="border-border/75 bg-bg-primary/72 grid gap-3 rounded-[20px] border px-3.5 py-3 md:grid-cols-2">
                      <ProviderMeta
                        icon={<Route className="size-3.5" />}
                        label="Model scope"
                        value={provider.models}
                      />
                      <ProviderMeta
                        icon={<ArrowUpRight className="size-3.5" />}
                        label="Live operations"
                        value="Health probes and mutation flows remain disabled until admin API writes are connected."
                      />
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </SectionCard>

        <SectionCard
          description="A compact view of where the next admin API integration should write status and risk signals."
          title="Readiness checks"
          tone="solid"
        >
          <div className="space-y-3">
            {[
              {
                detail:
                  "Primary OpenAI route exposes a defined alternate target.",
                icon: (
                  <CheckCircle2 className="size-4 text-emerald-600 dark:text-emerald-300" />
                ),
                title: "Fallback path documented",
              },
              {
                detail:
                  "Codex control target is listed without a secondary route and should be reviewed before production enablement.",
                icon: (
                  <ShieldAlert className="size-4 text-amber-600 dark:text-amber-300" />
                ),
                title: "Single-path dependency remains",
              },
              {
                detail:
                  "Credential rotation is represented in the dashboard, but backend mutation is intentionally not wired yet.",
                icon: <KeyRound className="text-primary size-4" />,
                title: "Rotation workflow pending API",
              },
            ].map((item) => (
              <div
                className="border-border/85 bg-bg-primary/75 flex gap-3 rounded-[22px] border px-4 py-4"
                key={item.title}
              >
                <div className="mt-0.5">{item.icon}</div>
                <div>
                  <h3 className="text-fg-primary text-sm font-semibold">
                    {item.title}
                  </h3>
                  <p className="text-fg-secondary mt-1 text-sm leading-6">
                    {item.detail}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </SectionCard>
      </div>
    </section>
  );
}

function ProviderMeta({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="border-border/85 bg-bg-secondary rounded-[20px] border px-3.5 py-3.5">
      <p className="text-fg-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.18em] uppercase">
        {icon}
        {label}
      </p>
      <p className="text-fg-primary mt-2 text-sm leading-6">{value}</p>
    </div>
  );
}

function ProviderCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-fg-muted text-[11px] font-semibold tracking-[0.2em] uppercase xl:hidden">
        {label}
      </p>
      <p className="text-fg-primary mt-1 text-sm leading-6 break-words xl:mt-0">
        {value}
      </p>
    </div>
  );
}

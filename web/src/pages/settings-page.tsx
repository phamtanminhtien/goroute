import {
  ArrowRightLeft,
  BadgeCheck,
  LockKeyhole,
  Network,
  Radar,
  ServerCog,
} from "lucide-react";
import type { ReactNode } from "react";

import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { StatusBadge } from "@/shared/ui/status-badge";

export function SettingsPage() {
  return (
    <section className="space-y-6">
      <PageHeader
        description="Review ingress, auth posture, fallback policy, and runtime defaults from a classic admin dashboard layout."
        eyebrow="Runtime"
        title="System configuration"
      >
        <StatusBadge tone="info">Read-only phase</StatusBadge>
      </PageHeader>

      <SectionCard
        description="Fast operational summary for the current runtime posture."
        title="Runtime overview"
        tone="solid"
      >
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <RuntimeSummaryCard label="Listen address" value=":2232" />
          <RuntimeSummaryCard label="Auth mode" value="Bearer token" />
          <RuntimeSummaryCard label="Primary ingress" value="/v1 compatible" />
          <RuntimeSummaryCard label="Mutation support" value="Deferred" />
        </div>
      </SectionCard>

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <SectionCard
          description="Core service posture and network shape presented as grouped system information."
          title="Ingress and server binding"
          tone="solid"
        >
          <div className="grid gap-3 lg:grid-cols-2">
            <RuntimePanel
              description="HTTP listener currently exposed on the local admin port."
              icon={<ServerCog className="size-4" />}
              label="Listen address"
              value=":2232"
            />
            <RuntimePanel
              description="Current admin surface assumes a direct OpenAI-compatible ingress."
              icon={<Network className="size-4" />}
              label="Ingress profile"
              value="HTTP /v1 compatibility"
            />
            <RuntimePanel
              description="Routing stays server-side so clients do not carry provider-specific logic."
              icon={<ArrowRightLeft className="size-4" />}
              label="Request routing"
              value="Provider-resolved upstream selection"
            />
            <RuntimePanel
              description="No mutable admin API writes are available in this frontend pass."
              icon={<Radar className="size-4" />}
              label="Control surface mode"
              value="Observe-first"
            />
          </div>
        </SectionCard>

        <SectionCard
          description="Session handling is intentionally simple today, but exposed with the right operational language."
          title="Auth and session posture"
          tone="solid"
        >
          <div className="space-y-3">
            <AuthSummaryRow label="Admin auth mode" value="Bearer token" />
            <AuthSummaryRow
              label="Session storage"
              value="Local browser storage"
            />
            <AuthSummaryRow
              label="Route protection"
              value="Frontend guard with redirect to /login"
            />
            <AuthSummaryRow
              label="Theme support"
              value="Light and dark console modes"
            />
          </div>
        </SectionCard>
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <SectionCard
          description="Policy is still mock-backed, but the information hierarchy is ready for live admin data."
          title="Routing defaults"
          tone="solid"
        >
          <div className="space-y-3">
            {[
              {
                label: "Model targeting",
                text: "Use explicit provider prefixes like cx/... or opena/... to keep upstream intent obvious.",
              },
              {
                label: "Fallback boundary",
                text: "Fallback should advance only for retryable upstream conditions, not client validation failures.",
              },
              {
                label: "Operator expectation",
                text: "The admin UI should surface route availability and auth posture before it offers mutation.",
              },
            ].map((item) => (
              <div
                className="border-border/85 bg-bg-primary/75 rounded-[22px] border px-4 py-4"
                key={item.label}
              >
                <p className="text-fg-primary text-sm font-semibold">
                  {item.label}
                </p>
                <p className="text-fg-secondary mt-1 text-sm leading-6">
                  {item.text}
                </p>
              </div>
            ))}
          </div>
        </SectionCard>

        <SectionCard
          description="Operational notes explain the current implementation boundaries without softening the message."
          title="Runtime notes"
          tone="solid"
        >
          <div className="space-y-3">
            {[
              {
                icon: <LockKeyhole className="text-primary size-4" />,
                title: "Auth remains local-first",
                text: "The token is stored locally and reused for admin requests until backend-issued sessions exist.",
              },
              {
                icon: (
                  <BadgeCheck className="size-4 text-emerald-600 dark:text-emerald-300" />
                ),
                title: "Route paths remain stable",
                text: "The UX shifts from generic settings language to runtime language without changing the /settings path.",
              },
              {
                icon: (
                  <Radar className="size-4 text-amber-600 dark:text-amber-300" />
                ),
                title: "Mutation is intentionally deferred",
                text: "This frontend pass does not fake persistence or invent write flows that the admin API does not yet support.",
              },
            ].map((item) => (
              <div
                className="border-border/85 bg-bg-primary/75 flex gap-3 rounded-[22px] border px-4 py-4"
                key={item.title}
              >
                <div className="mt-0.5">{item.icon}</div>
                <div>
                  <h2 className="text-fg-primary text-sm font-semibold">
                    {item.title}
                  </h2>
                  <p className="text-fg-secondary mt-1 text-sm leading-6">
                    {item.text}
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

function AuthSummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="border-border/85 bg-bg-primary/75 flex flex-col gap-2 rounded-[22px] border px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-fg-muted text-[11px] font-semibold tracking-[0.2em] uppercase">
        {label}
      </p>
      <p className="text-fg-primary text-sm font-medium">{value}</p>
    </div>
  );
}

function RuntimeSummaryCard({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="border-border/90 bg-bg-primary/72 rounded-[22px] border px-4 py-4">
      <p className="text-fg-muted text-[11px] font-semibold tracking-[0.2em] uppercase">
        {label}
      </p>
      <p className="text-fg-primary mt-3 text-2xl font-semibold tracking-tight">
        {value}
      </p>
    </div>
  );
}

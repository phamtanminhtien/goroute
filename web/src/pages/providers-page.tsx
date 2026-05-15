import { useQuery } from "@tanstack/react-query";
import {
  Bot,
  Braces,
  Cloud,
  Code2,
  Cpu,
  Gamepad2,
  Gem,
  KeyRound,
  Layers3,
  type LucideIcon,
  Shield,
  Sparkles,
  Star,
} from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  listProviders,
  type ProviderItem,
  providersQueryKey,
} from "@/features/providers/api";
import { Button } from "@/shared/ui/button";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { StatusBadge } from "@/shared/ui/status-badge";

type ProviderSection = {
  items: ProviderItem[];
  key: string;
  title: string;
};

const categoryTitles: Record<string, string> = {
  api_key: "API Key Providers",
  custom: "Custom Providers",
  free_tier: "Free Tier Providers",
  oauth: "OAuth Providers",
};

export function ProvidersPage() {
  const navigate = useNavigate();
  const providersQuery = useQuery({
    queryFn: listProviders,
    queryKey: providersQueryKey,
  });

  const providers = providersQuery.data ?? [];
  const sections = buildProviderSections(providers);

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Browse provider categories, inspect connection readiness, and open each provider for detailed connection management."
        eyebrow="Providers"
        title="Provider registry"
      >
        <StatusBadge tone="info">Live admin data</StatusBadge>
      </PageHeader>

      {providersQuery.isPending ? (
        <SectionCard
          description="Loading the latest provider catalog and connection groups."
          title="Loading providers"
          tone="solid"
        >
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 3 }).map((_, index) => (
              <div
                className="border-border/85 bg-bg-primary/72 min-h-[126px] animate-pulse rounded-[22px] border"
                key={index}
              />
            ))}
          </div>
        </SectionCard>
      ) : null}

      {providersQuery.isError ? (
        <SectionCard
          description="The provider registry could not be loaded from the admin API."
          title="Provider catalog unavailable"
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

      {!providersQuery.isPending &&
      !providersQuery.isError &&
      sections.length === 0 ? (
        <SectionCard
          description="No providers were returned by the admin API."
          title="Provider catalog is empty"
          tone="solid"
        >
          <p className="text-fg-secondary text-sm leading-6">
            Add providers to the system catalog before using the registry UI.
          </p>
        </SectionCard>
      ) : null}

      {!providersQuery.isPending && !providersQuery.isError ? (
        <div className="space-y-8">
          {sections.map((section) => (
            <section className="space-y-4" key={section.key}>
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <h2 className="text-[1rem] font-semibold tracking-[-0.03em] text-[var(--dashboard-title)]">
                  {section.title}
                </h2>
                <p className="text-[13px] text-[var(--dashboard-muted-soft)]">
                  {section.items.length} provider
                  {section.items.length === 1 ? "" : "s"}
                </p>
              </div>

              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {section.items.map((provider) => (
                  <ProviderCard
                    key={provider.id}
                    onOpen={() => navigate(`/providers/${provider.id}`)}
                    provider={provider}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function ProviderCard({
  onOpen,
  provider,
}: {
  onOpen: () => void;
  provider: ProviderItem;
}) {
  const [logoMissing, setLogoMissing] = useState(false);
  const logoPath = `/images/providers/${provider.id}.png`;

  return (
    <button
      className="dashboard-panel dashboard-panel-hover group flex min-h-[74px] w-full cursor-pointer items-center justify-start gap-3 rounded-[20px] border px-3.5 py-2.5 text-left shadow-[var(--shadow-sm)] transition-colors duration-150"
      onClick={onOpen}
      type="button"
    >
      <div className="dashboard-icon-surface flex size-[46px] shrink-0 items-center justify-center overflow-hidden rounded-[13px] border bg-[#080808]">
        {!logoMissing ? (
          <img
            alt=""
            className="h-full w-full object-cover"
            onError={() => setLogoMissing(true)}
            src={logoPath}
          />
        ) : null}
        {logoMissing ? (
          <div className="flex h-full w-full items-center justify-center bg-[#080808] text-white">
            <ProviderLogoFallback provider={provider} />
          </div>
        ) : null}
      </div>

      <div className="min-w-0 space-y-1">
        <h3 className="truncate text-[14px] font-semibold tracking-[-0.03em] text-[var(--dashboard-title)]">
          {provider.name}
        </h3>
        <ConnectionStatusPill count={provider.connection_count} />
      </div>
    </button>
  );
}

function buildProviderSections(providers: ProviderItem[]) {
  const groupedSections = new Map<string, ProviderItem[]>();

  for (const provider of providers) {
    const category = provider.category || "uncategorized";
    const sectionProviders = groupedSections.get(category) ?? [];
    sectionProviders.push(provider);
    groupedSections.set(category, sectionProviders);
  }

  const sections: ProviderSection[] = [];
  for (const [key, items] of groupedSections.entries()) {
    sections.push({
      items,
      key,
      title: categoryTitles[key] ?? humanizeCategory(key),
    });
  }

  return sections;
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

function ConnectionStatusPill({ count }: { count: number }) {
  const connected = count > 0;

  return (
    <div
      className={
        connected
          ? "inline-flex items-center gap-1.5 rounded-full bg-emerald-500/14 px-2.5 py-1 text-[10px] font-semibold text-emerald-500"
          : "inline-flex items-center gap-1.5 rounded-full bg-white/6 px-2.5 py-1 text-[10px] font-semibold text-[var(--dashboard-muted-soft)]"
      }
    >
      <span
        className={
          connected
            ? "size-2 rounded-full bg-emerald-500"
            : "size-2 rounded-full bg-[var(--dashboard-muted-soft)]/70"
        }
      />
      <span>{buildConnectionStatus(count)}</span>
    </div>
  );
}

function resolveProviderIcon(provider: ProviderItem): LucideIcon {
  switch (provider.id) {
    case "openai":
      return KeyRound;
    case "cx":
      return Sparkles;
  }

  if (provider.category === "oauth") {
    return Bot;
  }
  if (provider.category === "api_key") {
    return Braces;
  }
  if (provider.category === "free_tier") {
    return Cloud;
  }
  if (provider.category === "custom") {
    return Layers3;
  }

  const fallbackIcons: LucideIcon[] = [
    Bot,
    Code2,
    Cpu,
    Gamepad2,
    Gem,
    Shield,
    Sparkles,
    Star,
  ];
  const hash = provider.id
    .split("")
    .reduce((total, character) => total + character.charCodeAt(0), 0);

  return fallbackIcons[hash % fallbackIcons.length];
}

function ProviderLogoFallback({ provider }: { provider: ProviderItem }) {
  switch (provider.id) {
    case "openai":
      return <KeyRound className="size-5" />;
    case "cx":
      return <Sparkles className="size-5" />;
  }

  if (provider.category === "oauth") {
    return <Bot className="size-5" />;
  }
  if (provider.category === "api_key") {
    return <Braces className="size-5" />;
  }
  if (provider.category === "free_tier") {
    return <Cloud className="size-5" />;
  }
  if (provider.category === "custom") {
    return <Layers3 className="size-5" />;
  }

  return <Code2 className="size-5" />;
}

function humanizeCategory(value: string) {
  return value
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

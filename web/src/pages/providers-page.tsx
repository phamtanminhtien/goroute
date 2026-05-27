import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Bot,
  Braces,
  Cloud,
  Code2,
  Layers3,
  Plus,
  Sparkles,
} from "lucide-react";
import { type FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  createProvider,
  listProviders,
  type ProviderItem,
  type ProviderPayload,
  providersQueryKey,
} from "@/features/providers/api";
import { Button } from "@/shared/ui/button";
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

type ProviderSection = {
  items: ProviderItem[];
  key: string;
  title: string;
};

const categoryTitles: Record<string, string> = {
  custom: "Custom Providers",
  oauth: "OAuth Providers",
  api_key: "API Key Providers",
  free_tier: "Free Tier Providers",
};

const categoryOrder = ["custom", "oauth", "api_key", "free_tier"];

const customProviderAdapterOptions = [
  {
    icon: <ProviderBrandIcon providerID="openai" size="sm" />,
    label: "OpenAI compatible",
    value: "openai_compatible",
  },
  {
    icon: <ProviderBrandIcon providerID="anthropic" size="sm" />,
    label: "Anthropic compatible",
    value: "anthropic_compatible",
  },
];

export function ProvidersPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState({
    adapterType: "openai_compatible",
    apiKey: "",
    baseURL: "",
    defaultModel: "",
    id: "",
    name: "",
  });
  const [formError, setFormError] = useState<string | null>(null);
  const providersQuery = useQuery({
    queryFn: listProviders,
    queryKey: providersQueryKey,
  });

  const providers = providersQuery.data ?? [];
  const sections = buildProviderSections(providers);
  const createProviderMutation = useMutation({
    mutationFn: (payload: ProviderPayload) => createProvider(payload),
    onError: (error) => {
      setFormError(error instanceof Error ? error.message : "Request failed");
    },
    onSuccess: async (provider) => {
      await queryClient.invalidateQueries({ queryKey: providersQueryKey });
      setCreateOpen(false);
      setForm({
        adapterType: "openai_compatible",
        apiKey: "",
        baseURL: "",
        defaultModel: "",
        id: "",
        name: "",
      });
      navigate(`/providers/${provider.id}`);
    },
  });

  async function handleCreateProvider(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const id = form.id.trim();
    const name = form.name.trim();
    const baseURL = form.baseURL.trim();
    const defaultModel = form.defaultModel.trim();
    const apiKey = form.apiKey.trim();
    if (!id || !name || !baseURL || !defaultModel || !apiKey) {
      setFormError(
        "Provider ID, name, base URL, default model, and API key are required.",
      );
      return;
    }
    setFormError(null);
    await createProviderMutation.mutateAsync({
      adapter_type: form.adapterType,
      api_key: apiKey,
      base_url: baseURL,
      default_model: defaultModel,
      enabled: true,
      id,
      name,
    });
  }

  return (
    <section className="space-y-6 pb-6">
      <PageHeader
        description="Browse provider categories, inspect connection readiness, and open each provider for detailed connection management."
        eyebrow="Providers"
        title="Provider registry"
      >
        <Button
          leadingIcon={<Plus className="size-[15px]" />}
          onClick={() => {
            setFormError(null);
            setCreateOpen(true);
          }}
        >
          Add provider
        </Button>
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
              <Skeleton className="min-h-[126px]" key={index} />
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

      {!providersQuery.isPending &&
      !providersQuery.isError &&
      sections.length === 0 ? (
        <SectionCard
          description="No providers were returned by the admin API."
          title="Provider catalog is empty"
          tone="solid"
        >
          <EmptyState
            body="Add providers to the system catalog before using the registry UI."
            title="No providers available"
          />
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

      <Modal onOpenChange={setCreateOpen} open={createOpen}>
        <ModalContent>
          <ModalPanel
            description="Create a custom compatible provider and its managed connection."
            title="Add custom provider"
          >
            <form
              className="space-y-4"
              onSubmit={(event) => void handleCreateProvider(event)}
            >
              <div className="grid gap-4 md:grid-cols-2">
                <Field label="Provider ID" required>
                  <Input
                    autoFocus
                    onChange={(event) =>
                      setForm((current) => ({
                        ...current,
                        id: event.target.value,
                      }))
                    }
                    placeholder="openrouter"
                    value={form.id}
                  />
                </Field>
                <Field label="Name" required>
                  <Input
                    onChange={(event) =>
                      setForm((current) => ({
                        ...current,
                        name: event.target.value,
                      }))
                    }
                    placeholder="OpenRouter"
                    value={form.name}
                  />
                </Field>
              </div>
              <Field label="Adapter" required>
                <Select
                  onValueChange={(value) =>
                    setForm((current) => ({
                      ...current,
                      adapterType: value,
                    }))
                  }
                  options={customProviderAdapterOptions}
                  value={form.adapterType}
                />
              </Field>
              <Field label="Base URL" required>
                <Input
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      baseURL: event.target.value,
                    }))
                  }
                  placeholder={
                    providerAdapterPlaceholder(form.adapterType).baseURL
                  }
                  value={form.baseURL}
                />
              </Field>
              <Field label="Default model" required>
                <Input
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      defaultModel: event.target.value,
                    }))
                  }
                  placeholder={
                    providerAdapterPlaceholder(form.adapterType).defaultModel
                  }
                  value={form.defaultModel}
                />
              </Field>
              <Field label="API key" required>
                <Input
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      apiKey: event.target.value,
                    }))
                  }
                  placeholder="Enter API key"
                  type="password"
                  value={form.apiKey}
                />
              </Field>
              {formError ? (
                <InlineAlert tone="error">{formError}</InlineAlert>
              ) : null}
              <ModalFooter>
                <Button
                  disabled={createProviderMutation.isPending}
                  onClick={() => setCreateOpen(false)}
                  tone="secondary"
                  type="button"
                >
                  Cancel
                </Button>
                <Button
                  disabled={createProviderMutation.isPending}
                  type="submit"
                >
                  {createProviderMutation.isPending
                    ? "Creating..."
                    : "Create provider"}
                </Button>
              </ModalFooter>
            </form>
          </ModalPanel>
        </ModalContent>
      </Modal>
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
        <div className="flex flex-wrap items-center gap-1.5">
          <ConnectionStatusPill
            enabledCount={enabledConnectionCount(provider)}
            totalCount={provider.connection_count}
          />
          <CustomProviderAdapterBadge provider={provider} />
        </div>
      </div>
    </button>
  );
}

function providerAdapterPlaceholder(adapterType: string) {
  if (adapterType === "anthropic_compatible") {
    return {
      baseURL: "https://api.anthropic.com",
      defaultModel: "myanthropic/claude-sonnet-4-5",
    };
  }

  return {
    baseURL: "https://openrouter.ai/api",
    defaultModel: "openrouter/openai/gpt-4.1",
  };
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
  for (const key of categoryOrder) {
    if (!groupedSections.has(key)) {
      continue;
    }
    const items = groupedSections.get(key)!;
    sections.push({
      items,
      key,
      title: categoryTitles[key] ?? humanizeCategory(key),
    });
  }

  return sections;
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

function ConnectionStatusPill({
  enabledCount,
  totalCount,
}: {
  enabledCount: number;
  totalCount: number;
}) {
  const connected = enabledCount > 0;

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
      <span>{buildConnectionStatus(enabledCount, totalCount)}</span>
    </div>
  );
}

function CustomProviderAdapterBadge({ provider }: { provider: ProviderItem }) {
  if (provider.category !== "custom") {
    return null;
  }

  const adapter = customProviderAdapterDetails(provider);

  return (
    <span
      aria-hidden="true"
      className="inline-flex items-center gap-1 rounded-full bg-white/6 px-2 py-1 text-[10px] font-semibold text-[var(--dashboard-muted-soft)]"
    >
      {adapter.icon}
      <span>{adapter.label}</span>
    </span>
  );
}

function customProviderAdapterDetails(provider: ProviderItem) {
  if (provider.adapter_type === "anthropic_compatible") {
    return {
      icon: <ProviderBrandIcon providerID="anthropic" size="xs" />,
      label: "Anthropic",
    };
  }

  if (provider.adapter_type === "openai_compatible" || !provider.adapter_type) {
    return {
      icon: <ProviderBrandIcon providerID="openai" size="xs" />,
      label: "OpenAI",
    };
  }

  return {
    icon: <Layers3 className="size-3" />,
    label: humanizeCategory(provider.adapter_type),
  };
}

function ProviderLogoFallback({ provider }: { provider: ProviderItem }) {
  switch (provider.id) {
    case "anthropic":
      return <ProviderBrandIcon providerID="anthropic" size="md" />;
    case "openai":
      return <ProviderBrandIcon providerID="openai" size="md" />;
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
    if (provider.adapter_type === "anthropic_compatible") {
      return <ProviderBrandIcon providerID="anthropic" size="md" />;
    }
    if (
      provider.adapter_type === "openai_compatible" ||
      !provider.adapter_type
    ) {
      return <ProviderBrandIcon providerID="openai" size="md" />;
    }
    return <Layers3 className="size-5" />;
  }

  return <Code2 className="size-5" />;
}

function ProviderBrandIcon({
  providerID,
  size,
}: {
  providerID: "anthropic" | "openai";
  size: "md" | "sm" | "xs";
}) {
  const sizeClassName = {
    md: "size-6",
    sm: "size-4",
    xs: "size-3",
  }[size];

  return (
    <img
      alt=""
      aria-hidden="true"
      className={`${sizeClassName} shrink-0 rounded-[4px] bg-white/90 object-contain p-[1px]`}
      src={`/images/providers/${providerID}.png`}
    />
  );
}

function humanizeCategory(value: string) {
  return value
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

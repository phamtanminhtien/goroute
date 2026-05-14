import type { LucideIcon } from "lucide-react";
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
  Play,
  Plus,
  Shield,
  Sparkles,
  Star,
} from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/shared/ui/button";
import { StatusBadge } from "@/shared/ui/status-badge";

type ProviderTone = "default" | "success";

type ProviderItem = {
  icon: LucideIcon;
  label?: string;
  name: string;
  status: string;
  tone?: ProviderTone;
};

type ProviderSection = {
  action?: ReactNode;
  providers: ProviderItem[];
  title: string;
};

const sections: ProviderSection[] = [
  {
    action: (
      <div className="flex flex-wrap items-center gap-2.5">
        <Button
          className="min-w-[210px] justify-center rounded-[12px] border-0 px-4 text-[13px] text-white"
          leadingIcon={<Plus className="size-[15px]" />}
        >
          Add Anthropic Compatible
        </Button>
        <Button
          className="bg-bg-secondary text-fg-primary hover:bg-bg-tertiary min-w-[196px] justify-center rounded-[12px] border border-[var(--dashboard-sidebar-border)] text-[13px] shadow-none"
          leadingIcon={<Plus className="size-[15px]" />}
        >
          Add OpenAI Compatible
        </Button>
      </div>
    ),
    providers: [
      {
        icon: Bot,
        name: "Wokushop",
        status: "1 Connected",
        tone: "success",
      },
    ],
    title: "Custom Providers (OpenAI/Anthropic Compatible)",
  },
  {
    action: (
      <Button
        className="bg-bg-secondary hover:bg-bg-tertiary hover:text-fg-primary rounded-[12px] border-[var(--dashboard-sidebar-border)] px-3.5 text-[13px] text-[var(--dashboard-subtle-text)] shadow-none"
        leadingIcon={<Play className="size-[15px]" />}
        tone="secondary"
      >
        Test All
      </Button>
    ),
    providers: [
      { icon: Sparkles, name: "Claude Code", status: "No connections" },
      { icon: Gem, name: "Antigravity", status: "No connections" },
      {
        icon: KeyRound,
        name: "OpenAI Codex",
        status: "9 Connected",
        tone: "success",
      },
      { icon: Bot, name: "GitHub Copilot", status: "No connections" },
      { icon: Code2, name: "Cursor IDE", status: "No connections" },
      {
        icon: Braces,
        name: "Kilo Code",
        status: "1 Connected",
        tone: "success",
      },
      { icon: Bot, name: "Cline", status: "No connections" },
    ],
    title: "OAuth Providers",
  },
  {
    action: (
      <Button
        className="bg-bg-secondary hover:bg-bg-tertiary hover:text-fg-primary rounded-[12px] border-[var(--dashboard-sidebar-border)] px-3.5 text-[13px] text-[var(--dashboard-subtle-text)] shadow-none"
        leadingIcon={<Play className="size-[15px]" />}
        tone="secondary"
      >
        Test All
      </Button>
    ),
    providers: [
      { icon: Bot, name: "Kiro AI", status: "No connections" },
      { icon: Sparkles, name: "Qwen Code", status: "No connections" },
      { icon: Code2, name: "Gemini CLI", status: "No connections" },
      { icon: Bot, name: "iFlow AI", status: "No connections" },
      {
        icon: Layers3,
        name: "OpenCode Free",
        status: "Ready",
        tone: "success",
      },
      { icon: Braces, name: "OpenRouter", status: "No connections" },
      { icon: Cpu, name: "NVIDIA NIM", status: "No connections" },
      { icon: Bot, name: "Ollama Cloud", status: "No connections" },
      { icon: Shield, name: "Vertex AI", status: "No connections" },
      { icon: Sparkles, name: "Gemini", status: "No connections" },
      {
        icon: Cloud,
        name: "Cloudflare",
        status: "1 Connected",
        tone: "success",
      },
      { icon: Braces, name: "BytePlus ModelArk", status: "No connections" },
    ],
    title: "Free Tier Providers",
  },
  {
    action: (
      <Button
        className="bg-bg-secondary hover:bg-bg-tertiary hover:text-fg-primary rounded-[12px] border-[var(--dashboard-sidebar-border)] px-3.5 text-[13px] text-[var(--dashboard-subtle-text)] shadow-none"
        leadingIcon={<Play className="size-[15px]" />}
        tone="secondary"
      >
        Test All
      </Button>
    ),
    providers: [
      { icon: Code2, name: "GLM Coding", status: "No connections" },
      { icon: Code2, name: "GLM (China)", status: "No connections" },
      { icon: Star, name: "Kimi", status: "No connections" },
      { icon: Gamepad2, name: "Minimax Coding", status: "No connections" },
      { icon: Gamepad2, name: "Minimax (China)", status: "No connections" },
      { icon: Braces, name: "Alibaba", status: "No connections" },
      { icon: Braces, name: "Alibaba Intl", status: "No connections" },
      { icon: Bot, name: "Xiaomi MiMo", status: "No connections" },
    ],
    title: "API Key Providers",
  },
];

export function ProvidersPage() {
  return (
    <section className="space-y-6 pb-6">
      <h1 className="sr-only">Provider registry</h1>
      <div className="space-y-8">
        {sections.map((section) => (
          <section className="space-y-4" key={section.title}>
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <h2 className="text-[1rem] font-semibold tracking-[-0.03em] text-[var(--dashboard-title)]">
                {section.title}
              </h2>
              {section.action}
            </div>

            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              {section.providers.map((provider) => (
                <ProviderCard key={provider.name} provider={provider} />
              ))}
            </div>
          </section>
        ))}
      </div>
    </section>
  );
}

function ProviderCard({ provider }: { provider: ProviderItem }) {
  const Icon = provider.icon;

  return (
    <article className="dashboard-panel dashboard-panel-hover group flex min-h-[68px] items-center gap-3 rounded-[16px] border px-3.5 py-3 shadow-[var(--shadow-sm)] transition-colors duration-150">
      <div className="dashboard-icon-surface flex size-10 shrink-0 items-center justify-center rounded-[14px] border">
        {provider.label ? (
          <span className="text-[13px] font-semibold tracking-[-0.02em]">
            {provider.label}
          </span>
        ) : (
          <Icon className="size-[17px]" />
        )}
      </div>

      <div className="min-w-0">
        <h3 className="truncate text-[15px] font-semibold tracking-[-0.025em] text-[var(--dashboard-title)]">
          {provider.name}
        </h3>
        <div className="mt-1">
          {provider.tone === "success" ? (
            <StatusBadge size="sm" tone="success">
              {provider.status}
            </StatusBadge>
          ) : (
            <p className="text-[13px] text-[var(--dashboard-muted-soft)]">
              {provider.status}
            </p>
          )}
        </div>
      </div>
    </article>
  );
}

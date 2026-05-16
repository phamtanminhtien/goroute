import {
  Box,
  Cable,
  ChartNoAxesColumn,
  Layers3,
  PanelsTopLeft,
  Settings,
  TerminalSquare,
  Workflow,
  X,
} from "lucide-react";

import { SidebarNavItem } from "@/app/layout/sidebar-nav-item";
import { Button } from "@/shared/ui/button";

const navItems = [
  {
    description: "Registry, health posture, and fallback readiness",
    icon: Workflow,
    label: "Providers",
    to: "/providers",
  },
  {
    description: "On-demand Codex session and review quota checks",
    icon: ChartNoAxesColumn,
    label: "Codex usage",
    to: "/quota/codex",
  },
  {
    description: "Ingress, auth posture, and runtime defaults",
    icon: PanelsTopLeft,
    label: "Runtime",
    to: "/settings",
  },
];

export function SidebarNav({
  isMobileOpen = false,
  onClose,
}: {
  isMobileOpen?: boolean;
  onClose?: () => void;
}) {
  return (
    <>
      <aside className="admin-sidebar fixed inset-y-0 left-0 z-30 hidden w-[300px] lg:block">
        <SidebarNavContent />
      </aside>

      {isMobileOpen ? (
        <div
          aria-label="Navigation menu"
          className="fixed inset-0 z-50 flex lg:hidden"
          role="dialog"
        >
          <button
            aria-label="Close navigation menu"
            className="flex-1 bg-slate-950/56 backdrop-blur-[2px]"
            onClick={onClose}
            type="button"
          />
          <div className="relative h-full w-[292px] max-w-[86vw]">
            <SidebarNavContent mobile onClose={onClose} onNavigate={onClose} />
          </div>
        </div>
      ) : null}
    </>
  );
}

function SidebarNavContent({
  mobile = false,
  onClose,
  onNavigate,
}: {
  mobile?: boolean;
  onClose?: () => void;
  onNavigate?: () => void;
}) {
  return (
    <div className="dashboard-sidebar-frame flex h-full flex-col border-r">
      <div className="flex min-h-[60px] items-center justify-between gap-3 border-b border-[var(--dashboard-sidebar-border)] px-4">
        <div className="flex min-w-0 items-center gap-3">
          <div className="bg-primary flex size-8 shrink-0 items-center justify-center rounded-[14px] text-white shadow-[var(--shadow-button)]">
            <Cable className="size-4" />
          </div>
          <h1 className="truncate text-[14px] font-semibold tracking-[-0.03em] text-[var(--dashboard-title)]">
            GoRoute Proxy
          </h1>
        </div>
        <div className="shrink-0">
          {mobile ? (
            <Button
              aria-label="Close navigation menu"
              className="text-[var(--dashboard-muted-soft)] hover:text-[var(--dashboard-title)]"
              iconOnly
              leadingIcon={<X className="size-4" />}
              onClick={onClose}
              ripple={false}
              tone="ghost"
            />
          ) : null}
        </div>
      </div>

      <div className="flex-1 px-4 py-5">
        <nav
          aria-label="Primary admin navigation"
          className="mt-2.5 space-y-1.5"
        >
          {navItems.map((item) => (
            <SidebarNavItem key={item.to} onNavigate={onNavigate} {...item} />
          ))}
        </nav>

        <div className="mt-7">
          <p className="px-2.5 text-[10px] font-semibold tracking-[0.22em] text-[var(--dashboard-muted-strong)] uppercase">
            System
          </p>
          <div className="mt-2.5 space-y-1">
            {[
              { icon: Layers3, label: "Model Catalog" },
              { icon: TerminalSquare, label: "Console Log" },
              { icon: Box, label: "Proxy Pools" },
              { icon: Settings, label: "Settings" },
            ].map((item) => {
              const Icon = item.icon;

              return (
                <div
                  className="dashboard-static-item flex items-center gap-2.5 rounded-[14px] px-2.5 py-2 text-[13px] text-[var(--dashboard-subtle-text)] transition-colors duration-150"
                  key={item.label}
                >
                  <Icon className="size-[15px]" />
                  <span>{item.label}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

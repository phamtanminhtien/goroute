import {
  Boxes,
  ChartNoAxesColumn,
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
    description: "Ordered model aliases across provider targets",
    icon: Boxes,
    label: "Combos",
    to: "/combos",
  },
  {
    description: "Mock request, token, and cost analytics across the proxy",
    icon: ChartNoAxesColumn,
    label: "Usage & Analytics",
    to: "/usage",
  },
  {
    description: "On-demand Codex session and review quota tracking",
    icon: PanelsTopLeft,
    label: "Quota Tracker",
    to: "/quota",
  },
  {
    description: "Ingress, auth posture, and runtime defaults",
    icon: Settings,
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
          <img
            alt=""
            aria-hidden="true"
            className="size-9 shrink-0 object-contain"
            src="/images/goroute-logo.svg"
          />
          <h1 className="truncate text-[14px] font-semibold tracking-[-0.03em] text-[var(--dashboard-title)]">
            GoRoute
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
              // { icon: Layers3, label: "Model Catalog" },
              {
                description: "Live backend app and request logging stream",
                icon: TerminalSquare,
                label: "Console Log",
                to: "/logs",
              },
              // { icon: Box, label: "Proxy Pools" },
            ].map((item) => {
              return (
                <SidebarNavItem
                  description={item.description}
                  icon={item.icon}
                  key={item.label}
                  label={item.label}
                  onNavigate={onNavigate}
                  to={item.to}
                />
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

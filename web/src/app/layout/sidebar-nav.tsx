import { PanelsTopLeft, ShieldCheck, Workflow, X } from "lucide-react";

import { SidebarNavItem } from "@/app/layout/sidebar-nav-item";
import { Button } from "@/shared/ui/button";
import { StatusBadge } from "@/shared/ui/status-badge";

const navItems = [
  {
    description: "Registry, health posture, and fallback readiness",
    icon: Workflow,
    label: "Providers",
    to: "/providers",
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
      <aside className="admin-sidebar fixed inset-y-0 left-0 z-30 hidden w-[288px] lg:block">
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
          <div className="relative h-full w-[286px] max-w-[86vw]">
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
    <div className="border-border/90 bg-bg-secondary text-fg-primary flex h-full flex-col border-r">
      <div className="border-border/90 border-b px-5 py-5">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-fg-muted text-[11px] font-semibold tracking-[0.24em] uppercase">
              goroute
            </p>
            <h1 className="mt-2 text-xl font-semibold tracking-tight">
              Admin Dashboard
            </h1>
            <p className="text-fg-secondary mt-2 text-sm leading-6">
              Providers, routing policy, and runtime posture in one place.
            </p>
          </div>
          {mobile ? (
            <Button
              aria-label="Close navigation menu"
              className="text-fg-secondary hover:text-fg-primary"
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
        <p className="text-fg-muted px-2 text-[11px] font-semibold tracking-[0.22em] uppercase">
          Navigation
        </p>
        <nav aria-label="Primary admin navigation" className="mt-3 space-y-2">
          {navItems.map((item) => (
            <SidebarNavItem key={item.to} onNavigate={onNavigate} {...item} />
          ))}
        </nav>
      </div>

      <div className="border-border/90 border-t px-4 py-4">
        <div className="border-border/90 bg-bg-primary rounded-[22px] border px-4 py-4">
          <div className="flex items-center gap-2">
            <StatusBadge size="sm" tone="success">
              <ShieldCheck className="size-3.5" />
              Session active
            </StatusBadge>
          </div>
          <p className="text-fg-secondary mt-3 text-sm leading-6">
            This dashboard is operating in observe-first mode until write
            endpoints are available.
          </p>
        </div>
      </div>
    </div>
  );
}

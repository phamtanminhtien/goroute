import type { LucideIcon } from "lucide-react";
import { NavLink } from "react-router-dom";

import { cn } from "@/shared/lib/cn";

export function SidebarNavItem({
  description,
  icon: Icon,
  label,
  onNavigate,
  to,
}: {
  description: string;
  icon: LucideIcon;
  label: string;
  onNavigate?: () => void;
  to: string;
}) {
  return (
    <NavLink onClick={onNavigate} to={to}>
      {({ isActive }) => (
        <span
          className={cn(
            "flex items-start gap-2.5 rounded-[14px] border px-3 py-2.5 transition-[background-color,border-color,color,box-shadow]",
            isActive
              ? "dashboard-active-item shadow-none"
              : "dashboard-static-item border-transparent text-[var(--dashboard-subtle-text)]",
          )}
        >
          <span
            className={cn(
              "mt-0.5 inline-flex size-9 shrink-0 items-center justify-center rounded-[12px] border",
              isActive
                ? "dashboard-active-icon"
                : "dashboard-icon-surface text-[var(--dashboard-muted-strong)]",
            )}
          >
            <Icon className="size-[15px]" />
          </span>
          <span className="min-w-0">
            <span className="block text-[13px] font-semibold">{label}</span>
            <span
              className={cn(
                "mt-0.5 block text-[11px] leading-4.5",
                isActive
                  ? "text-[var(--dashboard-active-muted)]"
                  : "text-[var(--dashboard-muted-strong)]",
              )}
            >
              {description}
            </span>
          </span>
        </span>
      )}
    </NavLink>
  );
}

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
            "flex items-start gap-3 rounded-[18px] border px-3.5 py-3 transition-[background-color,border-color,color,box-shadow]",
            isActive
              ? "border-primary/22 bg-primary text-white shadow-[var(--shadow-sm)]"
              : "text-fg-secondary hover:border-border hover:bg-bg-primary hover:text-fg-primary border-transparent",
          )}
        >
          <span
            className={cn(
              "mt-0.5 inline-flex size-10 shrink-0 items-center justify-center rounded-2xl border",
              isActive
                ? "border-white/12 bg-white/12 text-white"
                : "border-border/90 bg-bg-secondary text-fg-muted",
            )}
          >
            <Icon className="size-4" />
          </span>
          <span className="min-w-0">
            <span className="block text-sm font-semibold">{label}</span>
            <span
              className={cn(
                "mt-1 block text-xs leading-5",
                isActive ? "text-blue-100/90" : "text-fg-muted",
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

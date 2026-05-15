import { LogOut, Menu, MoonStar, SunMedium } from "lucide-react";
import { useLocation } from "react-router-dom";

import { useAuthStore } from "@/features/auth/auth-store";
import { useUIStore } from "@/shared/store/ui-store";
import { Button } from "@/shared/ui/button";

const topbarMeta = {
  "/providers": {
    eyebrow: "Providers",
    title: "Manage your AI provider connections",
  },
  "/providers/detail": {
    eyebrow: "Providers",
    title: "Inspect provider capabilities and manage credentials",
  },
  "/settings": {
    eyebrow: "Runtime",
    title: "Review ingress, auth, and default routing behavior",
  },
} as const;

export function Topbar({ onOpenNavigation }: { onOpenNavigation: () => void }) {
  const location = useLocation();
  const signOut = useAuthStore((state) => state.signOut);
  const theme = useUIStore((state) => state.theme);
  const toggleTheme = useUIStore((state) => state.toggleTheme);

  const pageMeta = resolveTopbarMeta(location.pathname);

  return (
    <header className="admin-topbar dashboard-topbar-shell z-20 shrink-0 border-b backdrop-blur-[14px]">
      <div className="flex min-h-[60px] items-center justify-between gap-3 px-4 sm:px-5 lg:px-7">
        <div className="flex min-w-0 items-center gap-2.5">
          <Button
            aria-label="Open navigation menu"
            className="bg-bg-secondary hover:bg-bg-tertiary hover:text-fg-primary h-8 border-[var(--dashboard-sidebar-border)] text-[var(--dashboard-subtle-text)] lg:hidden"
            iconOnly
            leadingIcon={<Menu className="size-[15px]" />}
            onClick={onOpenNavigation}
            ripple={false}
            tone="secondary"
          />
          <div className="min-w-0">
            <p className="text-[10px] font-semibold tracking-[0.22em] text-[var(--dashboard-muted-strong)] uppercase">
              {pageMeta.eyebrow}
            </p>
            <p className="truncate text-[14px] font-semibold tracking-[-0.03em] text-[var(--dashboard-title)] sm:text-[16px]">
              {pageMeta.title}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            aria-label={
              theme === "light"
                ? "Switch to dark theme"
                : "Switch to light theme"
            }
            className="hover:bg-bg-secondary hover:text-fg-primary h-8 border-[var(--dashboard-sidebar-border)] bg-transparent text-[var(--dashboard-subtle-text)] shadow-none"
            iconOnly
            leadingIcon={
              theme === "light" ? (
                <MoonStar className="size-[15px]" />
              ) : (
                <SunMedium className="size-[15px]" />
              )
            }
            onClick={toggleTheme}
            tone="secondary"
          />
          <Button
            aria-label="Sign out"
            className="hover:bg-bg-secondary h-8 rounded-[12px] border-[var(--dashboard-shutdown-border)] bg-transparent px-3 text-[13px] text-[var(--dashboard-shutdown-text)] shadow-none"
            leadingIcon={<LogOut className="size-[15px]" />}
            onClick={signOut}
            tone="secondary"
          >
            Logout
          </Button>
        </div>
      </div>
    </header>
  );
}

function resolveTopbarMeta(pathname: string) {
  if (pathname.startsWith("/providers/")) {
    return topbarMeta["/providers/detail"];
  }

  return (
    topbarMeta[pathname as keyof typeof topbarMeta] ?? topbarMeta["/providers"]
  );
}

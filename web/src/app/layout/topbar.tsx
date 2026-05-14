import { Menu, MoonStar, SunMedium } from "lucide-react";
import { useLocation } from "react-router-dom";

import { useAuthStore } from "@/features/auth/auth-store";
import { useUIStore } from "@/shared/store/ui-store";
import { Button } from "@/shared/ui/button";

const topbarMeta = {
  "/providers": {
    eyebrow: "Provider Operations",
    title: "Monitor upstream readiness and credential posture",
  },
  "/settings": {
    eyebrow: "Runtime Control",
    title: "Review ingress, auth, and default routing behavior",
  },
} as const;

export function Topbar({ onOpenNavigation }: { onOpenNavigation: () => void }) {
  const location = useLocation();
  const signOut = useAuthStore((state) => state.signOut);
  const theme = useUIStore((state) => state.theme);
  const toggleTheme = useUIStore((state) => state.toggleTheme);

  const pageMeta =
    topbarMeta[location.pathname as keyof typeof topbarMeta] ??
    topbarMeta["/providers"];

  return (
    <header className="admin-topbar border-border/85 bg-bg-secondary/92 sticky top-0 z-20 border-b backdrop-blur-[10px]">
      <div className="flex min-h-[76px] items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <Button
            aria-label="Open navigation menu"
            className="lg:hidden"
            iconOnly
            leadingIcon={<Menu className="size-4" />}
            onClick={onOpenNavigation}
            ripple={false}
            tone="secondary"
          />
          <div className="min-w-0">
            <p className="text-fg-muted text-[11px] font-semibold tracking-[0.24em] uppercase">
              {pageMeta.eyebrow}
            </p>
            <p className="text-fg-primary truncate text-base font-semibold tracking-tight sm:text-lg">
              {pageMeta.title}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 sm:gap-3">
          <Button
            aria-label={
              theme === "light"
                ? "Switch to dark theme"
                : "Switch to light theme"
            }
            leadingIcon={
              theme === "light" ? (
                <MoonStar className="size-4" />
              ) : (
                <SunMedium className="size-4" />
              )
            }
            onClick={toggleTheme}
            tone="secondary"
          >
            <span className="hidden sm:inline">
              {theme === "light" ? "Dark mode" : "Light mode"}
            </span>
          </Button>
          <Button onClick={signOut} tone="secondary">
            <span className="hidden sm:inline">Sign out</span>
            <span className="sm:hidden">Exit</span>
          </Button>
        </div>
      </div>
    </header>
  );
}

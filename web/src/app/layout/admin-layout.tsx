import { motion, useAnimationControls } from "motion/react";
import { useLayoutEffect, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";

import { SidebarNav } from "@/app/layout/sidebar-nav";
import { Topbar } from "@/app/layout/topbar";
import { fadeInUp } from "@/shared/lib/motion";

export function AdminLayout() {
  const location = useLocation();
  const [isMobileNavOpen, setIsMobileNavOpen] = useState(false);
  const contentAnimation = useAnimationControls();

  useLayoutEffect(() => {
    contentAnimation.set("hidden");
    void contentAnimation.start("visible");
  }, [contentAnimation, location.pathname]);

  return (
    <main className="admin-dashboard h-dvh overflow-hidden">
      <SidebarNav
        isMobileOpen={isMobileNavOpen}
        onClose={() => setIsMobileNavOpen(false)}
      />

      <div className="flex h-dvh min-h-0 flex-col overflow-hidden lg:pl-[300px]">
        <Topbar onOpenNavigation={() => setIsMobileNavOpen(true)} />

        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4 sm:px-5 sm:py-5 lg:px-7 lg:py-5">
          <motion.div
            animate={contentAnimation}
            className="space-y-5"
            initial={false}
            variants={fadeInUp}
          >
            <Outlet />
          </motion.div>
        </div>
      </div>
    </main>
  );
}

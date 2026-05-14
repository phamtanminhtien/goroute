import { AnimatePresence, motion } from "motion/react";
import { useState } from "react";
import { Outlet, useLocation } from "react-router-dom";

import { SidebarNav } from "@/app/layout/sidebar-nav";
import { Topbar } from "@/app/layout/topbar";
import { fadeInUp } from "@/shared/lib/motion";

export function AdminLayout() {
  const location = useLocation();
  const [isMobileNavOpen, setIsMobileNavOpen] = useState(false);

  return (
    <main className="admin-dashboard h-dvh overflow-hidden">
      <SidebarNav
        isMobileOpen={isMobileNavOpen}
        onClose={() => setIsMobileNavOpen(false)}
      />

      <div className="flex h-dvh min-h-0 flex-col overflow-hidden lg:pl-[300px]">
        <Topbar onOpenNavigation={() => setIsMobileNavOpen(true)} />

        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4 sm:px-5 sm:py-5 lg:px-7 lg:py-5">
          <AnimatePresence mode="wait">
            <motion.div
              animate="visible"
              className="space-y-5"
              initial="hidden"
              key={location.pathname}
              variants={fadeInUp}
            >
              <Outlet />
            </motion.div>
          </AnimatePresence>
        </div>
      </div>
    </main>
  );
}

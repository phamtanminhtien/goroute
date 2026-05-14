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
    <main className="admin-shell min-h-screen">
      <SidebarNav
        isMobileOpen={isMobileNavOpen}
        onClose={() => setIsMobileNavOpen(false)}
      />

      <div className="min-h-screen lg:pl-[288px]">
        <Topbar onOpenNavigation={() => setIsMobileNavOpen(true)} />

        <div className="px-4 py-5 sm:px-6 sm:py-6 lg:px-8 lg:py-8">
          <AnimatePresence mode="wait">
            <motion.div
              animate="visible"
              className="space-y-6"
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

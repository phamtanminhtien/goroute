import { Navigate, useRoutes } from "react-router-dom";

import { AdminShell } from "@/app/layout/admin-shell";
import { ProvidersPage } from "@/pages/providers-page";
import { SettingsPage } from "@/pages/settings-page";

export function AppRoutes() {
  return useRoutes([
    {
      path: "/",
      element: <AdminShell />,
      children: [
        { index: true, element: <Navigate to="/providers" replace /> },
        { path: "providers", element: <ProvidersPage /> },
        { path: "settings", element: <SettingsPage /> },
      ],
    },
  ]);
}

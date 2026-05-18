import { Navigate, type RouteObject, useRoutes } from "react-router-dom";

import { AdminLayout } from "@/app/layout/admin-layout";
import { AuthGuard, PublicOnlyGuard } from "@/features/auth/auth-guard";
import { AIRequestLogDetailPage } from "@/pages/ai-request-log-detail-page";
import { AIRequestLogsPage } from "@/pages/ai-request-logs-page";
import { CodexUsagePage } from "@/pages/codex-usage-page";
import { CombosPage } from "@/pages/combos-page";
import { ConsoleLogPage } from "@/pages/console-log-page";
import { LoginPage } from "@/pages/login-page";
import { ProviderDetailPage } from "@/pages/provider-detail-page";
import { ProvidersPage } from "@/pages/providers-page";
import { SettingsPage } from "@/pages/settings-page";
import { UsageAnalyticsPage } from "@/pages/usage-analytics-page";

const appRoutes: RouteObject[] = [
  {
    path: "/login",
    element: (
      <PublicOnlyGuard>
        <LoginPage />
      </PublicOnlyGuard>
    ),
  },
  {
    path: "/",
    element: <AuthGuard />,
    children: [
      {
        element: <AdminLayout />,
        children: [
          { index: true, element: <Navigate to="/providers" replace /> },
          { path: "providers", element: <ProvidersPage /> },
          { path: "providers/:providerId", element: <ProviderDetailPage /> },
          { path: "combos", element: <CombosPage /> },
          { path: "usage", element: <UsageAnalyticsPage /> },
          { path: "ai-logs", element: <AIRequestLogsPage /> },
          { path: "ai-logs/:requestId", element: <AIRequestLogDetailPage /> },
          { path: "logs", element: <ConsoleLogPage /> },
          { path: "quota", element: <CodexUsagePage /> },
          { path: "settings", element: <SettingsPage /> },
        ],
      },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/providers" replace />,
  },
];

export function AppRoutes() {
  return useRoutes(appRoutes);
}

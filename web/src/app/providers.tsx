import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { PropsWithChildren } from "react";
import { useEffect } from "react";
import { BrowserRouter, useNavigate } from "react-router-dom";

import { authRedirectEvent } from "@/features/auth/auth-session";
import { useAuthStore } from "@/features/auth/auth-store";
import { useSyncTheme } from "@/shared/store/ui-store";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});

function ThemeBridge() {
  useSyncTheme();
  return null;
}

function AuthBridge() {
  const hydrate = useAuthStore((state) => state.hydrate);
  const navigate = useNavigate();

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  useEffect(() => {
    function handleAuthRedirect() {
      navigate("/login", { replace: true });
    }

    window.addEventListener(authRedirectEvent, handleAuthRedirect);
    return () =>
      window.removeEventListener(authRedirectEvent, handleAuthRedirect);
  }, [navigate]);

  return null;
}

export function AppProviders({ children }: PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ThemeBridge />
        <AuthBridge />
        {children}
      </BrowserRouter>
    </QueryClientProvider>
  );
}

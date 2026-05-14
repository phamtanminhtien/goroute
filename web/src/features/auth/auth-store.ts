import { create } from "zustand";

import {
  clearStoredAdminToken,
  persistAdminToken,
  readStoredAdminToken,
} from "@/features/auth/auth-session";

type AuthState = {
  hydrated: boolean;
  isAuthenticated: boolean;
  token: string | null;
  hydrate: () => void;
  signIn: (token: string) => void;
  signOut: () => void;
};

function readAuthSnapshot() {
  const token = readStoredAdminToken();

  return {
    hydrated: true,
    isAuthenticated: token !== null,
    token,
  };
}

export const useAuthStore = create<AuthState>((set) => ({
  ...readAuthSnapshot(),
  hydrate: () => set(readAuthSnapshot()),
  signIn: (token) => {
    const normalizedToken = token.trim();

    persistAdminToken(normalizedToken);
    set({
      hydrated: true,
      isAuthenticated: true,
      token: normalizedToken,
    });
  },
  signOut: () => {
    clearStoredAdminToken();
    set({
      hydrated: true,
      isAuthenticated: false,
      token: null,
    });
  },
}));

export function getAuthToken() {
  return useAuthStore.getState().token;
}

export function clearAuthSession() {
  useAuthStore.getState().signOut();
}

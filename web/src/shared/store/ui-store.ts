import { useEffect } from "react";
import { create } from "zustand";

export type ThemeMode = "dark" | "light";

type UIState = {
  theme: ThemeMode;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
};

const storageKey = "goroute.theme";

function readInitialTheme(): ThemeMode {
  if (typeof document === "undefined") {
    return "dark";
  }

  const storedTheme = localStorage.getItem(storageKey);
  if (storedTheme === "dark" || storedTheme === "light") {
    return storedTheme;
  }

  const datasetTheme = document.documentElement.dataset.theme;
  if (datasetTheme === "dark" || datasetTheme === "light") {
    return datasetTheme;
  }

  return "dark";
}

export const useUIStore = create<UIState>((set) => ({
  theme: readInitialTheme(),
  setTheme: (theme) => set({ theme }),
  toggleTheme: () =>
    set((state) => ({ theme: state.theme === "dark" ? "light" : "dark" })),
}));

export function useSyncTheme() {
  const theme = useUIStore((state) => state.theme);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem(storageKey, theme);
  }, [theme]);
}

import { beforeEach, describe, expect, it } from "vitest";

import { adminTokenStorageKey } from "@/features/auth/auth-session";
import { useAuthStore } from "@/features/auth/auth-store";

describe("auth store", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: false,
      token: null,
    });
  });

  it("persists token on sign in", () => {
    useAuthStore.getState().signIn("secret-token");

    expect(useAuthStore.getState().token).toBe("secret-token");
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(localStorage.getItem(adminTokenStorageKey)).toBe("secret-token");
  });

  it("clears state and storage on sign out", () => {
    useAuthStore.getState().signIn("secret-token");

    useAuthStore.getState().signOut();

    expect(useAuthStore.getState().token).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(localStorage.getItem(adminTokenStorageKey)).toBeNull();
  });

  it("hydrates token from localStorage", () => {
    localStorage.setItem(adminTokenStorageKey, "restored-token");

    useAuthStore.getState().hydrate();

    expect(useAuthStore.getState().token).toBe("restored-token");
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
  });
});

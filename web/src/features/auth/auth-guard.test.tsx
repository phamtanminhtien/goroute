import { screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, it } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import { renderWithQueryClient } from "@/test/test-utils";

describe("auth routes", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: false,
      token: null,
    });
  });

  it("redirects unauthenticated users from admin routes to login", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", {
      level: 2,
      name: /sign in with your admin token/i,
    });
  });

  it("redirects authenticated users from login to providers", async () => {
    useAuthStore.getState().signIn("secret-token");

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/login"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", {
      level: 1,
      name: /provider registry/i,
    });
  });
});

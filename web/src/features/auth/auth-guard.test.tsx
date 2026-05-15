import { screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import { listProviders } from "@/features/providers/api";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("@/features/providers/api", () => ({
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
}));

const providersListMock = vi.mocked(listProviders);

describe("auth routes", () => {
  beforeEach(() => {
    localStorage.clear();
    providersListMock.mockResolvedValue([
      {
        auth_type: "oauth",
        category: "oauth",
        connection_count: 0,
        connections: [],
        default_model: "cx/gpt-5.4",
        id: "cx",
        models: [{ description: "", id: "cx/gpt-5.4", name: "GPT-5.4" }],
        name: "Codex",
      },
    ]);
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

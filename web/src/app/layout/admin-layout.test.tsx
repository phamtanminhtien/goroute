import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import { listProviders } from "@/features/providers/api";
import { useUIStore } from "@/shared/store/ui-store";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("@/features/providers/api", () => ({
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
}));

const providersListMock = vi.mocked(listProviders);
const defaultProviders = [
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
];

describe("admin layout", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.dataset.theme = "light";
    providersListMock.mockResolvedValue(defaultProviders);
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    useUIStore.setState({ theme: "light" });
  });

  it("renders the classic full-width dashboard", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", {
      level: 1,
      name: /provider registry/i,
    });

    expect(
      screen.getByRole("navigation", { name: /primary admin navigation/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /open navigation menu/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /sign out/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { level: 1, name: /goroute proxy/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/operational control surface/i),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/admin token active/i)).not.toBeInTheDocument();
  });

  it("keeps sign-out accessible from the admin dashboard", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: /sign out/i }));

    await screen.findByRole("heading", {
      level: 2,
      name: /sign in with your admin token/i,
    });
  });

  it("switches between providers and runtime sections", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("link", { name: /runtime/i }));

    await screen.findByRole("heading", {
      level: 1,
      name: /system configuration/i,
    });
    expect(screen.getByText(/ingress and server binding/i)).toBeInTheDocument();
  });

  it("opens the mobile navigation drawer from the header", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await user.click(
      screen.getByRole("button", { name: /open navigation menu/i }),
    );

    expect(
      screen.getByRole("dialog", { name: /navigation menu/i }),
    ).toBeInTheDocument();
    expect(
      screen.getAllByRole("link", { name: /runtime/i }).length,
    ).toBeGreaterThan(0);
  });
});

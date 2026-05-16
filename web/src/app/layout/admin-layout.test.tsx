import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import {
  connectionUsageQueryKey,
  deleteConnection,
  getConnectionUsage,
  listProviders,
  updateConnection,
} from "@/features/providers/api";
import { useUIStore } from "@/shared/store/ui-store";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("@/features/providers/api", () => ({
  connectionUsageQueryKey: vi.fn((connectionID: string) => [
    "connections",
    connectionID,
    "usage",
  ]),
  deleteConnection: vi.fn(),
  getConnectionUsage: vi.fn(),
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
  updateConnection: vi.fn(),
}));

const connectionUsageQueryKeyMock = vi.mocked(connectionUsageQueryKey);
const deleteConnectionMock = vi.mocked(deleteConnection);
const getConnectionUsageMock = vi.mocked(getConnectionUsage);
const providersListMock = vi.mocked(listProviders);
const updateConnectionMock = vi.mocked(updateConnection);
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
    connectionUsageQueryKeyMock.mockImplementation((connectionID: string) => [
      "connections",
      connectionID,
      "usage",
    ]);
    deleteConnectionMock.mockResolvedValue(undefined);
    getConnectionUsageMock.mockResolvedValue({
      limitReached: false,
      plan: "plus",
      quotas: {},
      reviewLimitReached: false,
    });
    providersListMock.mockResolvedValue(defaultProviders);
    updateConnectionMock.mockResolvedValue({
      has_access_token: true,
      has_api_key: false,
      has_refresh_token: false,
      id: "codex-1",
      name: "codex-user",
      problems: [],
      provider_id: "cx",
      status: "ready",
    });
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

  it("opens the dedicated quota screen from navigation", async () => {
    const user = userEvent.setup();
    getConnectionUsageMock.mockResolvedValue({
      limitReached: false,
      plan: "plus",
      quotas: {},
      reviewLimitReached: false,
    });

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("link", { name: /quota/i }));

    await screen.findByRole("heading", {
      level: 1,
      name: /codex usage/i,
    });
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
      screen.getAllByRole("link", { name: /codex usage/i }).length,
    ).toBeGreaterThan(0);
  });
});

import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import {
  completeOAuthConnection,
  connectionUsageQueryKey,
  createConnection,
  deleteConnection,
  generateProviderOAuthURL,
  getConnectionUsage,
  listProviders,
  updateConnection,
} from "@/features/providers/api";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("@/features/providers/api", () => ({
  connectionUsageQueryKey: vi.fn((connectionID: string) => [
    "connections",
    connectionID,
    "usage",
  ]),
  completeOAuthConnection: vi.fn(),
  createConnection: vi.fn(),
  deleteConnection: vi.fn(),
  generateProviderOAuthURL: vi.fn(),
  getConnectionUsage: vi.fn(),
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
  updateConnection: vi.fn(),
}));

const connectionUsageQueryKeyMock = vi.mocked(connectionUsageQueryKey);
const completeOAuthConnectionMock = vi.mocked(completeOAuthConnection);
const createConnectionMock = vi.mocked(createConnection);
const deleteConnectionMock = vi.mocked(deleteConnection);
const generateProviderOAuthURLMock = vi.mocked(generateProviderOAuthURL);
const getConnectionUsageMock = vi.mocked(getConnectionUsage);
const listProvidersMock = vi.mocked(listProviders);
const updateConnectionMock = vi.mocked(updateConnection);

const baseProviders = [
  {
    auth_type: "oauth",
    category: "oauth",
    connection_count: 1,
    connections: [
      {
        has_access_token: true,
        has_api_key: false,
        has_refresh_token: true,
        id: "codex-1",
        name: "codex-user",
        problems: [],
        provider_id: "cx",
        status: "ready",
      },
    ],
    default_model: "cx/gpt-5.4",
    id: "cx",
    models: [{ description: "", id: "cx/gpt-5.4", name: "GPT-5.4" }],
    name: "Codex",
  },
  {
    auth_type: "api_key",
    category: "api_key",
    connection_count: 0,
    connections: [],
    default_model: "openai/gpt-4.1",
    id: "openai",
    models: [{ description: "", id: "openai/gpt-4.1", name: "GPT-4.1" }],
    name: "OpenAI",
  },
];

describe("providers pages", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    vi.restoreAllMocks();
    listProvidersMock.mockResolvedValue(baseProviders);
    completeOAuthConnectionMock.mockResolvedValue(
      baseProviders[0].connections[0],
    );
    connectionUsageQueryKeyMock.mockImplementation((connectionID: string) => [
      "connections",
      connectionID,
      "usage",
    ]);
    createConnectionMock.mockResolvedValue(baseProviders[0].connections[0]);
    updateConnectionMock.mockResolvedValue(baseProviders[0].connections[0]);
    deleteConnectionMock.mockResolvedValue(undefined);
    getConnectionUsageMock.mockResolvedValue({
      limitReached: false,
      plan: "plus",
      quotas: {
        session: {
          remaining: 58,
          resetAt: "2026-05-16T10:00:00.000Z",
          total: 100,
          unlimited: false,
          used: 42,
        },
      },
      reviewLimitReached: false,
    });
    generateProviderOAuthURLMock.mockResolvedValue({
      sessionID: "oauth-session-1",
      url: "https://auth.openai.com/oauth/authorize?response_type=code&client_id=app_EMoamEEZ73f0CkXaXp7hrann",
    });
  });

  it("groups providers by category and computes connection status text", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByText(/oauth providers/i);

    expect(screen.getByText(/api key providers/i)).toBeInTheDocument();
    expect(screen.getByText("1 Connected")).toBeInTheDocument();
    expect(screen.getByText("No connections")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /codex 1 connected/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /openai no connections/i }),
    ).toBeInTheDocument();
  });

  it("renders provider detail with connections and available models sections", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /connections/i });

    expect(
      screen.getByRole("heading", { level: 2, name: /available models/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/^default$/i)).toBeInTheDocument();
    expect(screen.getByText(/codex-user/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /test/i })).toBeInTheDocument();
  });

  it("creates a connection from provider detail and refetches providers", async () => {
    const user = userEvent.setup();
    listProvidersMock
      .mockResolvedValueOnce([
        {
          ...baseProviders[0],
          connection_count: 0,
          connections: [],
        },
        baseProviders[1],
      ])
      .mockResolvedValueOnce([
        {
          ...baseProviders[0],
          connection_count: 1,
          connections: [
            {
              has_access_token: true,
              has_api_key: false,
              has_refresh_token: false,
              id: "codex-2",
              name: "new-user",
              problems: [],
              provider_id: "cx",
              status: "ready",
            },
          ],
        },
        baseProviders[1],
      ]);

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /connections/i });
    await user.click(screen.getByRole("button", { name: /add connection/i }));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(
      screen.getByText(/waiting for popup authorization/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^copy$/i })).toBeInTheDocument();
    expect(
      screen.getByDisplayValue(
        "https://auth.openai.com/oauth/authorize?response_type=code&client_id=app_EMoamEEZ73f0CkXaXp7hrann",
      ),
    ).toBeInTheDocument();
    expect(generateProviderOAuthURLMock).toHaveBeenCalledWith("cx");
    expect(
      screen.getByPlaceholderText(/localhost:20128\/callback\?code=/i),
    ).toBeInTheDocument();
    expect(screen.queryByText(/^connection details$/i)).not.toBeInTheDocument();

    await user.type(
      screen.getByPlaceholderText(/localhost:20128\/callback\?code=/i),
      "http://localhost:20128/callback?code=codex-2",
    );
    await user.click(
      screen.getByRole("button", { name: /create connection/i }),
    );

    await waitFor(() => {
      expect(completeOAuthConnectionMock).toHaveBeenCalledWith(
        "oauth-session-1",
        "http://localhost:20128/callback?code=codex-2",
      );
    });
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
    await screen.findByText(/connection saved/i);
    expect(screen.getByText("1 Connected")).toBeInTheDocument();
  });

  it("updates a connection without sending blank secret fields", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByText(/codex-user/i);

    await user.click(screen.getByRole("button", { name: /edit connection/i }));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    const displayNameInput = screen.getByLabelText(/display name/i);
    await user.clear(displayNameInput);
    await user.type(displayNameInput, "renamed-user");
    await user.click(screen.getByRole("button", { name: /save changes/i }));

    await waitFor(() => {
      expect(updateConnectionMock).toHaveBeenCalledWith("codex-1", {
        id: "codex-1",
        name: "renamed-user",
        provider_id: "cx",
      });
    });
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
  });

  it("shows callback-only fields for cx create form", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /connections/i });
    await user.click(screen.getByRole("button", { name: /add connection/i }));

    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(/localhost:20128\/callback\?code=/i),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText(/connection id/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/display name/i)).not.toBeInTheDocument();
    expect(
      screen.queryByPlaceholderText(/enter a new access token/i),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByPlaceholderText(/enter a new refresh token/i),
    ).not.toBeInTheDocument();
  });

  it("shows only api key field for openai forms", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/openai"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /connections/i });
    await user.click(screen.getByRole("button", { name: /add connection/i }));

    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(/enter a new api key/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByPlaceholderText(/enter a new access token/i),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByPlaceholderText(/enter a new refresh token/i),
    ).not.toBeInTheDocument();
  });

  it("deletes a connection and refetches providers", async () => {
    const user = userEvent.setup();
    listProvidersMock
      .mockResolvedValueOnce(baseProviders)
      .mockResolvedValueOnce([
        {
          ...baseProviders[0],
          connection_count: 0,
          connections: [],
        },
        baseProviders[1],
      ]);

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByText(/codex-user/i);
    await user.click(screen.getByRole("button", { name: /^delete$/i }));
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /confirm delete/i }));

    await waitFor(() => {
      expect(deleteConnectionMock).toHaveBeenNthCalledWith(
        1,
        "codex-1",
        expect.any(Object),
      );
    });
    await screen.findByText(/connection deleted/i);
  });
});

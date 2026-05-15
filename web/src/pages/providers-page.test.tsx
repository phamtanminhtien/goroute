import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import {
  createConnection,
  deleteConnection,
  listProviders,
  updateConnection,
} from "@/features/providers/api";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("@/features/providers/api", () => ({
  createConnection: vi.fn(),
  deleteConnection: vi.fn(),
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
  updateConnection: vi.fn(),
}));

const createConnectionMock = vi.mocked(createConnection);
const deleteConnectionMock = vi.mocked(deleteConnection);
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
    createConnectionMock.mockResolvedValue(baseProviders[0].connections[0]);
    updateConnectionMock.mockResolvedValue(baseProviders[0].connections[0]);
    deleteConnectionMock.mockResolvedValue(undefined);
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
      <MemoryRouter initialEntries={["/providers/cx?mode=create"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /connections/i });

    await user.type(screen.getByLabelText(/connection id/i), "codex-2");
    await user.type(screen.getByLabelText(/display name/i), "new-user");
    await user.type(screen.getByLabelText(/access token/i), "new-token");
    await user.click(
      screen.getByRole("button", { name: /create connection/i }),
    );

    await waitFor(() => {
      expect(createConnectionMock).toHaveBeenNthCalledWith(
        1,
        {
          access_token: "new-token",
          id: "codex-2",
          name: "new-user",
          provider_id: "cx",
        },
        expect.any(Object),
      );
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

import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import {
  connectionUsageQueryKey,
  deleteConnection,
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
  deleteConnection: vi.fn(),
  getConnectionUsage: vi.fn(),
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
  updateConnection: vi.fn(),
}));

const connectionUsageQueryKeyMock = vi.mocked(connectionUsageQueryKey);
const deleteConnectionMock = vi.mocked(deleteConnection);
const getConnectionUsageMock = vi.mocked(getConnectionUsage);
const listProvidersMock = vi.mocked(listProviders);
const updateConnectionMock = vi.mocked(updateConnection);

type MockUsageResponse = {
  limitReached: boolean;
  message?: string;
  plan?: string;
  quotas?: {
    session?: {
      remaining: number;
      resetAt?: string;
      total: number;
      unlimited: boolean;
      used: number;
    };
  };
  reviewLimitReached: boolean;
};

describe("codex usage page", () => {
  beforeEach(() => {
    vi.spyOn(Date, "now").mockReturnValue(
      new Date("2026-05-11T16:05:00.000Z").getTime(),
    );
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    connectionUsageQueryKeyMock.mockImplementation((connectionID: string) => [
      "connections",
      connectionID,
      "usage",
    ]);
    deleteConnectionMock.mockResolvedValue(undefined);
    listProvidersMock.mockResolvedValue([
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
    ]);
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
    updateConnectionMock.mockResolvedValue({
      has_access_token: true,
      has_api_key: false,
      has_refresh_token: true,
      id: "codex-1",
      name: "codex-user",
      problems: [],
      provider_id: "cx",
      status: "ready",
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads Codex quota on the dedicated screen", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/quota/codex"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 1, name: /codex usage/i });

    expect(await screen.findByText(/normal ready/i)).toBeInTheDocument();
    expect(screen.getByText(/plan plus/i)).toBeInTheDocument();
    expect(screen.getByText(/^session$/i)).toBeInTheDocument();
    expect(screen.getByText(/^42\/100$/i)).toBeInTheDocument();
    expect(screen.getByText(/used quota/i)).toBeInTheDocument();
    expect(screen.getByText(/4d 17h 55m/i)).toBeInTheDocument();
    expect(getConnectionUsageMock).toHaveBeenCalledWith("codex-1");
    expect(
      screen.getByRole("button", { name: /refresh codex-user usage/i }),
    ).toBeInTheDocument();
  });

  it("shows the upstream unavailable message when usage cannot be fetched", async () => {
    getConnectionUsageMock.mockResolvedValue({
      limitReached: false,
      message: "Codex connected. Usage API temporarily unavailable (503).",
      reviewLimitReached: false,
    });

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/quota/codex"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    expect(
      await screen.findByText(/usage api temporarily unavailable \(503\)/i),
    ).toBeInTheDocument();
  });

  it("shows a visible refreshing state when the refresh action runs", async () => {
    const user = userEvent.setup();
    let resolveRefresh!: (value: MockUsageResponse) => void;

    getConnectionUsageMock
      .mockResolvedValueOnce({
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
      })
      .mockImplementationOnce(
        () =>
          new Promise<MockUsageResponse>((resolve) => {
            resolveRefresh = resolve;
          }),
      );

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/quota/codex"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByText(/^session$/i);

    await user.click(
      screen.getByRole("button", { name: /refresh codex-user usage/i }),
    );

    expect(
      await screen.findByRole("button", { name: /refresh codex-user usage/i }),
    ).toHaveTextContent(/refreshing/i);
    expect(
      screen.getByRole("button", { name: /refresh codex-user usage/i }),
    ).toBeDisabled();

    resolveRefresh({
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

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /refresh codex-user usage/i }),
      ).toHaveTextContent(/^refresh$/i);
    });
  });
});

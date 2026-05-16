import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import { listProviders } from "@/features/providers/api";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("echarts", () => ({
  init: vi.fn(() => ({
    dispose: vi.fn(),
    resize: vi.fn(),
    setOption: vi.fn(),
  })),
}));

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

const listProvidersMock = vi.mocked(listProviders);

describe("usage analytics page", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    listProvidersMock.mockReset();
  });

  it("renders the usage analytics dashboard on /usage", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/usage"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: /usage & analytics/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/total requests/i)).toBeInTheDocument();
    expect(screen.getByText(/total input tokens/i)).toBeInTheDocument();
    expect(screen.getByText(/^output tokens$/i)).toBeInTheDocument();
    expect(screen.getAllByText(/est. cost/i)).toHaveLength(2);
    expect(screen.getByText(/recent requests/i)).toBeInTheDocument();
    expect(screen.getByText(/^request volume$/i)).toBeInTheDocument();
    expect(screen.getByText(/^token flow$/i)).toBeInTheDocument();
    expect(listProvidersMock).not.toHaveBeenCalled();
  });

  it("switches mocked datasets when the range filter changes", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/usage"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", {
      level: 1,
      name: /usage & analytics/i,
    });

    expect(screen.getByText("12,480")).toBeInTheDocument();
    expect(screen.getByText("$174.82")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "7D" }));

    expect(screen.getByText("78,930")).toBeInTheDocument();
    expect(screen.getByText("$1,024.18")).toBeInTheDocument();
    expect(screen.getByText(/today, 09:42/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "30D" }));

    expect(screen.getByText("341,280")).toBeInTheDocument();
    expect(screen.getByText("$4,296.77")).toBeInTheDocument();
    expect(screen.getByText(/may 16, 09:42/i)).toBeInTheDocument();
  });

  it("exposes usage analytics in the sidebar", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/usage"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("link", { name: /usage & analytics/i }),
    ).toHaveAttribute("href", "/usage");
  });
});

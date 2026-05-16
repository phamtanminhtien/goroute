import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import {
  getUsageProviderBreakdown,
  getUsageRecentRequests,
  getUsageSummary,
  getUsageTimeseries,
} from "@/features/analytics/api";
import { useAuthStore } from "@/features/auth/auth-store";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("echarts", () => ({
  init: vi.fn(() => ({
    dispose: vi.fn(),
    resize: vi.fn(),
    setOption: vi.fn(),
  })),
}));

vi.mock("@/features/analytics/api", () => ({
  getUsageProviderBreakdown: vi.fn(),
  getUsageRecentRequests: vi.fn(),
  getUsageSummary: vi.fn(),
  getUsageTimeseries: vi.fn(),
  usageProviderBreakdownQueryKey: vi.fn((range: string) => [
    "analytics",
    "usage",
    "provider-breakdown",
    range,
  ]),
  usageRangeOptions: [
    { label: "24H", value: "24h" },
    { label: "7D", value: "7d" },
    { label: "30D", value: "30d" },
  ],
  usageRecentRequestsQueryKey: vi.fn((range: string) => [
    "analytics",
    "usage",
    "recent-requests",
    range,
  ]),
  usageSummaryQueryKey: vi.fn((range: string) => [
    "analytics",
    "usage",
    "summary",
    range,
  ]),
  usageTimeseriesQueryKey: vi.fn((range: string) => [
    "analytics",
    "usage",
    "timeseries",
    range,
  ]),
}));

const getUsageSummaryMock = vi.mocked(getUsageSummary);
const getUsageTimeseriesMock = vi.mocked(getUsageTimeseries);
const getUsageProviderBreakdownMock = vi.mocked(getUsageProviderBreakdown);
const getUsageRecentRequestsMock = vi.mocked(getUsageRecentRequests);

describe("usage analytics page", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    getUsageSummaryMock.mockReset();
    getUsageTimeseriesMock.mockReset();
    getUsageProviderBreakdownMock.mockReset();
    getUsageRecentRequestsMock.mockReset();

    getUsageSummaryMock
      .mockResolvedValueOnce(
        buildSummaryResponse(12480, 18600000, 7900000, 174.82, 0.014),
      )
      .mockResolvedValueOnce(
        buildSummaryResponse(78930, 112400000, 49700000, 1024.18, 0.013),
      )
      .mockResolvedValueOnce(
        buildSummaryResponse(341280, 486900000, 201300000, 4296.77, 0.013),
      );
    getUsageTimeseriesMock.mockResolvedValue({
      bucket: "hour",
      filters: {
        connection_id: null,
        model: null,
        path: null,
        provider_id: null,
      },
      from: "2026-05-15T10:00:00Z",
      generated_at: "2026-05-16T10:00:05Z",
      points: [
        {
          bucket_end: "2026-05-16T00:59:59Z",
          bucket_start: "2026-05-16T00:00:00Z",
          estimated_cost_usd: 6.1,
          input_tokens: 680000,
          output_tokens: 240000,
          requests: 420,
        },
      ],
      to: "2026-05-16T10:00:00Z",
    });
    getUsageProviderBreakdownMock.mockResolvedValue({
      filters: {
        connection_id: null,
        model: null,
        path: null,
        provider_id: null,
      },
      from: "2026-05-15T10:00:00Z",
      generated_at: "2026-05-16T10:00:05Z",
      items: [
        {
          estimated_cost_usd: 82.1,
          input_tokens: 9000000,
          output_tokens: 3200000,
          provider_id: "openai",
          provider_name: "OpenAI",
          requests: 5990,
          requests_pct: 48,
        },
      ],
      to: "2026-05-16T10:00:00Z",
    });
    getUsageRecentRequestsMock.mockResolvedValue({
      filters: {
        connection_id: null,
        model: null,
        path: null,
        provider_id: null,
      },
      from: "2026-05-15T10:00:00Z",
      generated_at: "2026-05-16T10:00:05Z",
      items: [
        {
          connection_id: "codex-user",
          connection_name: "codex-user",
          estimated_cost_usd: 0.036,
          input_tokens: 11200,
          latency_ms: 1200,
          model: "gpt-5.4",
          output_tokens: 2100,
          path: "/v1/chat/completions",
          provider_id: "cx",
          provider_name: "Codex",
          request_id: "req_01",
          status: "completed",
          status_code: 200,
          timestamp: "2026-05-16T09:40:00Z",
        },
      ],
      page: {
        has_more: false,
        limit: 20,
        returned: 1,
      },
      to: "2026-05-16T10:00:00Z",
    });
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
    expect(getUsageSummaryMock).toHaveBeenCalledWith("24h");
  });

  it("switches analytics queries when the range filter changes", async () => {
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

    expect(await screen.findByText("12,480")).toBeInTheDocument();
    expect(await screen.findByText("$174.82")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "7D" }));

    expect(await screen.findByText("78,930")).toBeInTheDocument();
    expect(await screen.findByText("$1,024.18")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "30D" }));

    expect(await screen.findByText("341,280")).toBeInTheDocument();
    expect(await screen.findByText("$4,296.77")).toBeInTheDocument();
    expect(getUsageSummaryMock).toHaveBeenCalledTimes(3);
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

function buildSummaryResponse(
  requests: number,
  inputTokens: number,
  outputTokens: number,
  totalCost: number,
  averageCost: number,
) {
  return {
    estimated_cost_usd: {
      avg_per_request: averageCost,
      delta_label: "$0.014 avg / req",
      delta_tone: "critical" as const,
      value: totalCost,
    },
    filters: {
      connection_id: null,
      model: null,
      path: null,
      provider_id: null,
    },
    from: "2026-05-15T10:00:00Z",
    generated_at: "2026-05-16T10:00:05Z",
    input_tokens: {
      delta_label: "+9.4% context growth",
      delta_tone: "warning" as const,
      value: inputTokens,
    },
    output_tokens: {
      delta_label: "+13.1% generation load",
      delta_tone: "positive" as const,
      value: outputTokens,
    },
    requests: {
      delta_label: "+18.2% vs prev day",
      delta_tone: "positive" as const,
      value: requests,
    },
    to: "2026-05-16T10:00:00Z",
  };
}

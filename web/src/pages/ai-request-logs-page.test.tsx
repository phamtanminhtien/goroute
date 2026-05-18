import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import {
  getAIRequestLogDetail,
  getAIRequestLogs,
} from "@/features/analytics/api";
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

vi.mock("@/features/analytics/api", () => ({
  aiRequestLogDetailQueryKey: vi.fn((requestID: string) => [
    "analytics",
    "usage",
    "requests",
    requestID,
  ]),
  aiRequestLogsQueryKey: vi.fn((filters: unknown, page: number) => [
    "analytics",
    "usage",
    "requests",
    filters,
    page,
  ]),
  getAIRequestLogDetail: vi.fn(),
  getAIRequestLogs: vi.fn(),
  getUsageProviderBreakdown: vi.fn(),
  getUsageRecentRequests: vi.fn(),
  getUsageSummary: vi.fn(),
  getUsageTimeseries: vi.fn(),
  usageProviderBreakdownQueryKey: vi.fn(),
  usageRangeOptions: [
    { label: "24H", value: "24h" },
    { label: "7D", value: "7d" },
    { label: "30D", value: "30d" },
  ],
  usageRecentRequestsQueryKey: vi.fn(),
  usageSummaryQueryKey: vi.fn(),
  usageTimeseriesQueryKey: vi.fn(),
}));

vi.mock("@/features/providers/api", () => ({
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
}));

const getAIRequestLogsMock = vi.mocked(getAIRequestLogs);
const getAIRequestLogDetailMock = vi.mocked(getAIRequestLogDetail);
const listProvidersMock = vi.mocked(listProviders);

describe("ai request logs pages", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    getAIRequestLogsMock.mockReset();
    getAIRequestLogDetailMock.mockReset();
    listProvidersMock.mockReset();
    listProvidersMock.mockResolvedValue([
      {
        auth_type: "oauth",
        category: "oauth",
        connection_count: 1,
        connections: [
          {
            enabled: true,
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
    getAIRequestLogsMock.mockResolvedValue({
      filters: {
        connection_id: null,
        model: null,
        path: null,
        provider_id: null,
      },
      generated_at: "2026-05-16T10:00:05Z",
      items: [
        {
          connection_id: "codex-1",
          connection_name: "codex-user",
          estimated_cost_usd: 0.625,
          input_tokens: 100000,
          latency_ms: 1200,
          model: "cx/gpt-5.4",
          output_tokens: 50000,
          path: "/v1/chat/completions",
          provider_id: "cx",
          provider_name: "Codex",
          request_id: "req-detail",
          status: "completed",
          status_code: 200,
          timestamp: "2026-05-16T09:40:00Z",
        },
      ],
      page: {
        has_more: false,
        has_prev: false,
        limit: 50,
        page: 1,
        returned: 1,
      },
    });
    getAIRequestLogDetailMock.mockResolvedValue(buildDetailResponse());
  });

  it("renders a dedicated AI Logs list and links to detail pages", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/ai-logs"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("heading", { level: 1, name: /ai logs/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/persisted ai request runs/i)).toBeInTheDocument();
    expect(
      await screen.findByRole("link", { name: "cx/gpt-5.4" }),
    ).toHaveAttribute("href", "/ai-logs/req-detail");

    await user.click(screen.getByLabelText(/provider/i));
    await user.click(await screen.findByRole("option", { name: "Codex" }));

    expect(getAIRequestLogsMock).toHaveBeenLastCalledWith(
      { provider_id: "cx" },
      1,
    );
  });

  it("renders full persisted detail data on the detail page", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/ai-logs/req-detail"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("heading", { level: 1, name: /req-detail/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/0.625/i)).toBeInTheDocument();
    expect(screen.getByText(/codex-user/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: /flow/i }));
    expect(screen.getByText(/translated request body/i)).toBeInTheDocument();
    expect(screen.getAllByText(/gpt-5.4/i).length).toBeGreaterThan(0);

    await user.click(screen.getByRole("tab", { name: /upstream attempts/i }));
    expect(screen.getByText(/attempt 0/i)).toBeInTheDocument();
    expect(screen.getByText(/rate limited/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: /^rtk$/i }));
    expect(screen.getByText(/dedup-log/i)).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: /raw/i }));
    const rawPanel = screen.getByRole("tabpanel");
    expect(within(rawPanel).getByText(/third_party_logs/i)).toBeInTheDocument();
  });

  it("exposes AI Logs in the sidebar", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/ai-logs"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("link", { name: /ai logs/i }),
    ).toHaveAttribute("href", "/ai-logs");
  });
});

function buildDetailResponse() {
  return {
    estimated_cost_usd: 0.625,
    flow: {
      attempt_trace: `[{"connection_id":"codex-1"}]`,
      completed_at: 1778895601200,
      created_at: 1778895600000,
      duration_ms: 1200,
      error_message: "",
      error_type: "",
      id: 2,
      method: "POST",
      path: "/v1/chat/completions",
      provider_id: "cx",
      provider_name: "Codex",
      provider_request_mode: "stream",
      query: "",
      remote_addr: "127.0.0.1",
      request_body: `{"model":"cx/gpt-5.4"}`,
      request_headers: `{"Authorization":["REDACTED"]}`,
      request_id: "req-detail",
      request_mode: "stream",
      response_body: `{"id":"chatcmpl-1"}`,
      response_headers: `{"Content-Type":["application/json"]}`,
      response_status_code: 200,
      run_id: 1,
      started_at: 1778895600000,
      translated_request_body: `{"model":"gpt-5.4"}`,
      translated_response_body: "",
      type: "completions",
      updated_at: 1778895601200,
      user_agent: "vitest",
    },
    model: "cx/gpt-5.4",
    rtk: {
      applied: true,
      bytes_after: 700,
      bytes_before: 1000,
      created_at: 1778895600000,
      field_count: 1,
      filter_chain: "dedup-log",
      hit_count: 2,
      id: 4,
      request_id: "req-detail",
      run_id: 1,
      saved_bytes: 300,
      saved_percent: 30,
      updated_at: 1778895601200,
    },
    run: {
      attempt_count: 2,
      completed_at: 1778895601200,
      completion_tokens: 50000,
      created_at: 1778895600000,
      duration_ms: 1200,
      error_message: "",
      error_type: "",
      final_connection_id: "codex-1",
      final_connection_name: "codex-user",
      final_error_category: "",
      id: 1,
      method: "POST",
      path: "/v1/chat/completions",
      prompt_tokens: 100000,
      provider_id: "cx",
      provider_name: "Codex",
      provider_request_mode: "stream",
      request_id: "req-detail",
      request_mode: "stream",
      requested_model: "cx/gpt-5.4",
      resolved_model: "cx/gpt-5.4",
      started_at: 1778895600000,
      status_code: 200,
      total_tokens: 150000,
      type: "completions",
      updated_at: 1778895601200,
    },
    status: "completed" as const,
    third_party_logs: [
      {
        attempt_index: 0,
        completed_at: 1778895600500,
        connection_id: "codex-1",
        connection_name: "primary",
        created_at: 1778895600000,
        duration_ms: 500,
        error_message: "rate limited",
        error_type: "upstream_retryable_error",
        id: 3,
        provider_id: "cx",
        provider_name: "Codex",
        provider_request_mode: "stream",
        request_body: `{"model":"gpt-5.4"}`,
        request_headers: `{"Authorization":["REDACTED"]}`,
        request_id: "req-detail",
        request_method: "POST",
        request_mode: "stream",
        request_url: "https://chatgpt.com/backend-api/codex",
        response_body: "",
        response_headers: "",
        response_status_code: 429,
        run_id: 1,
        started_at: 1778895600000,
        type: "completions",
        updated_at: 1778895600500,
      },
    ],
  };
}

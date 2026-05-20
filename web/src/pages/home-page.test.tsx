import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import { getSettings, updateSettings } from "@/features/settings/api";
import {
  createSystemAPIKey,
  listSystemAPIKeys,
  updateSystemAPIKey,
} from "@/features/system-api-keys/api";
import { resolveOpenAIBaseURL } from "@/pages/home-page";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("@/features/settings/api", () => ({
  getSettings: vi.fn(),
  settingsQueryKey: ["settings"],
  updateSettings: vi.fn(),
}));

vi.mock("@/features/system-api-keys/api", () => ({
  createSystemAPIKey: vi.fn(),
  listSystemAPIKeys: vi.fn(),
  systemAPIKeysQueryKey: ["system-api-keys"],
  updateSystemAPIKey: vi.fn(),
}));

const getSettingsMock = vi.mocked(getSettings);
const updateSettingsMock = vi.mocked(updateSettings);
const createSystemAPIKeyMock = vi.mocked(createSystemAPIKey);
const listSystemAPIKeysMock = vi.mocked(listSystemAPIKeys);
const updateSystemAPIKeyMock = vi.mocked(updateSystemAPIKey);

describe("home page", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    vi.restoreAllMocks();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
    getSettingsMock.mockResolvedValue({
      llmLogging: { enabled: { flow: true, thirdParty: true } },
      openAICompatibleAuth: { enabled: false },
      rtk: { enabled: true },
      server: {
        listen: ":2232",
        web_ui_dir: "web/dist",
      },
    });
    updateSettingsMock.mockResolvedValue({
      llmLogging: { enabled: { flow: true, thirdParty: true } },
      openAICompatibleAuth: { enabled: true },
      rtk: { enabled: true },
      server: {
        listen: ":2232",
        web_ui_dir: "web/dist",
      },
    });
    listSystemAPIKeysMock.mockResolvedValue([]);
    createSystemAPIKeyMock.mockResolvedValue({
      created_at: 1760000000,
      enabled: true,
      id: "sak_test",
      key: "sk-goroute-testkey123456",
      last_used_at: 0,
      name: "Production app",
      updated_at: 1760000000,
    });
    updateSystemAPIKeyMock.mockResolvedValue({
      created_at: 1760000000,
      enabled: false,
      id: "sak_test",
      key: "sk-goroute-testkey123456",
      last_used_at: 0,
      name: "Production app",
      updated_at: 1760000001,
    });
  });

  it("renders OpenAI-compatible and Anthropic-compatible connection details from the runtime listen port", async () => {
    renderHome();

    expect(await screen.findByText("AI connection hub")).toBeInTheDocument();
    expect(
      screen.getAllByText(/OpenAI-compatible and Anthropic-compatible/i).length,
    ).toBeGreaterThan(0);
    expect(await screen.findByText("GET /v1/models")).toBeInTheDocument();
    expect(screen.getByText("POST /v1/chat/completions")).toBeInTheDocument();
    expect(screen.getByText("POST /v1/responses")).toBeInTheDocument();
    expect(screen.getByText("POST /v1/messages")).toBeInTheDocument();
    expect(
      screen.getByText(/anthropic\/claude-sonnet-4-5/),
    ).toBeInTheDocument();
    expect(screen.getByText("System API keys")).toBeInTheDocument();
  });

  it("toggles OpenAI-compatible auth through settings", async () => {
    const user = userEvent.setup();
    renderHome();

    const authSwitch = await screen.findByRole("switch", {
      name: /require system api key for api-compatible clients/i,
    });
    await user.click(authSwitch);

    await waitFor(() => {
      expect(updateSettingsMock).toHaveBeenCalledWith({
        llmLogging: { enabled: { flow: true, thirdParty: true } },
        openAICompatibleAuth: { enabled: true },
        rtk: { enabled: true },
      });
    });
  });

  it("resolves a custom localhost listen address without hard-coding the default port", async () => {
    expect(resolveOpenAIBaseURL(":2232")).toMatch(/:2232\/v1$/);
    expect(resolveOpenAIBaseURL("127.0.0.1:3000")).toBe(
      "http://127.0.0.1:3000/v1",
    );
  });

  it("adds a server-generated API key and toggles its enabled state", async () => {
    const user = userEvent.setup();
    renderHome();

    await screen.findByText("No API keys yet");
    await user.click(
      screen.getAllByRole("button", { name: /add api key/i })[0],
    );
    await user.type(screen.getByLabelText("Name"), "Production app");
    expect(screen.queryByLabelText("Key")).not.toBeInTheDocument();
    expect(
      screen.getByText(/api key value will be generated/i),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /save api key/i }));

    const row = await screen.findByText("Production app");
    const rowContainer = row.closest("[class]")?.parentElement?.parentElement;
    expect(rowContainer).toBeTruthy();
    expect(screen.getByText(/sk-goroute.*\.\.\./)).toBeInTheDocument();
    expect(screen.getByText("Enabled")).toBeInTheDocument();

    const switchControl = screen.getByRole("switch", {
      name: /production app api key enabled/i,
    });
    await user.click(switchControl);

    await waitFor(() => {
      expect(
        within(rowContainer as HTMLElement).getByText("Disabled"),
      ).toBeInTheDocument();
    });
  });
});

function renderHome() {
  renderWithQueryClient(
    <MemoryRouter initialEntries={["/"]}>
      <AppRoutes />
    </MemoryRouter>,
  );
}

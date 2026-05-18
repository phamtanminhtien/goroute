import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import {
  completeOAuthConnection,
  connectionUsageQueryKey,
  createConnection,
  createModelCombo,
  createProviderModel,
  deleteConnection,
  deleteModelCombo,
  deleteProviderModel,
  generateProviderOAuthURL,
  getConnectionUsage,
  listModelCombos,
  listProviders,
  testProviderModel,
  updateConnection,
  updateProviderModel,
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
  createModelCombo: vi.fn(),
  createProviderModel: vi.fn(),
  deleteConnection: vi.fn(),
  deleteProviderModel: vi.fn(),
  deleteModelCombo: vi.fn(),
  generateProviderOAuthURL: vi.fn(),
  getConnectionUsage: vi.fn(),
  listModelCombos: vi.fn(),
  listProviders: vi.fn(),
  providersQueryKey: ["providers"],
  testProviderModel: vi.fn(),
  updateConnection: vi.fn(),
  updateProviderModel: vi.fn(),
  updateModelCombo: vi.fn(),
}));

const connectionUsageQueryKeyMock = vi.mocked(connectionUsageQueryKey);
const completeOAuthConnectionMock = vi.mocked(completeOAuthConnection);
const createConnectionMock = vi.mocked(createConnection);
const createModelComboMock = vi.mocked(createModelCombo);
const createProviderModelMock = vi.mocked(createProviderModel);
const deleteConnectionMock = vi.mocked(deleteConnection);
const deleteProviderModelMock = vi.mocked(deleteProviderModel);
const deleteModelComboMock = vi.mocked(deleteModelCombo);
const generateProviderOAuthURLMock = vi.mocked(generateProviderOAuthURL);
const getConnectionUsageMock = vi.mocked(getConnectionUsage);
const listModelCombosMock = vi.mocked(listModelCombos);
const listProvidersMock = vi.mocked(listProviders);
const testProviderModelMock = vi.mocked(testProviderModel);
const updateConnectionMock = vi.mocked(updateConnection);
const updateProviderModelMock = vi.mocked(updateProviderModel);

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
    listModelCombosMock.mockResolvedValue([]);
    completeOAuthConnectionMock.mockResolvedValue(
      baseProviders[0].connections[0],
    );
    createModelComboMock.mockResolvedValue({
      alias: "combo/fast",
      description: "",
      name: "Fast Combo",
      targets: [
        {
          enabled: true,
          model_id: "cx/gpt-5.4",
          priority: 0,
          provider_id: "cx",
        },
      ],
    });
    deleteModelComboMock.mockResolvedValue(undefined);
    connectionUsageQueryKeyMock.mockImplementation((connectionID: string) => [
      "connections",
      connectionID,
      "usage",
    ]);
    createConnectionMock.mockResolvedValue(baseProviders[0].connections[0]);
    createProviderModelMock.mockResolvedValue({
      description: "",
      id: "cx/gpt-5.5",
      name: "GPT-5.5",
      source: "custom",
    });
    updateProviderModelMock.mockResolvedValue({
      description: "Updated custom model",
      id: "cx/gpt-5.5",
      name: "GPT-5.5 Turbo",
      source: "custom",
    });
    updateConnectionMock.mockResolvedValue(baseProviders[0].connections[0]);
    deleteProviderModelMock.mockResolvedValue(undefined);
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
    testProviderModelMock.mockResolvedValue({
      message: "Model test succeeded.",
      model: "cx/gpt-5.4",
      output_text: "OK",
      provider_id: "cx",
      status: "success",
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

  it("disables model testing and shows a tooltip when the provider has no connections", async () => {
    const user = userEvent.setup();
    listProvidersMock.mockResolvedValueOnce([
      baseProviders[0],
      {
        ...baseProviders[1],
        connection_count: 0,
        connections: [],
      },
    ]);

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/openai"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /available models/i });

    const testButton = screen.getByRole("button", { name: /^test$/i });
    expect(testButton).toBeDisabled();

    await user.hover(testButton.parentElement as HTMLElement);

    expect(await screen.findByRole("tooltip")).toHaveTextContent(
      /add at least one connection to test this model\./i,
    );
    expect(testProviderModelMock).not.toHaveBeenCalled();
  });

  it("runs a model test and renders inline success on the selected model card", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /available models/i });

    await user.click(screen.getByRole("button", { name: /^test$/i }));

    await waitFor(() => {
      expect(testProviderModelMock).toHaveBeenCalledWith("cx", {
        model: "cx/gpt-5.4",
      });
    });

    expect(
      await screen.findByText(/model test succeeded\./i),
    ).toBeInTheDocument();
    expect(screen.getByText(/^ok$/i)).toBeInTheDocument();
  });

  it("adds a provider model when the model id uses the provider prefix", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /available models/i });

    await user.click(screen.getByRole("button", { name: /add model/i }));
    const dialog = await screen.findByRole("dialog");

    await user.type(within(dialog).getByLabelText(/model id/i), "cx/gpt-5.5");
    await user.type(within(dialog).getByLabelText(/^name/i), "GPT-5.5");
    await user.click(
      within(dialog).getByRole("button", { name: /add model/i }),
    );

    await waitFor(() => {
      expect(createProviderModelMock).toHaveBeenCalledWith("cx", {
        description: "",
        id: "cx/gpt-5.5",
        name: "GPT-5.5",
      });
    });
  });

  it("labels system and custom models, and only custom models can be edited or deleted", async () => {
    const user = userEvent.setup();
    listProvidersMock.mockResolvedValueOnce([
      {
        ...baseProviders[0],
        models: [
          {
            description: "",
            id: "cx/gpt-5.4",
            name: "GPT-5.4",
            source: "system",
          },
          {
            description: "Custom model",
            id: "cx/gpt-5.5",
            name: "GPT-5.5",
            source: "custom",
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

    await screen.findByRole("heading", { level: 2, name: /available models/i });

    expect(screen.getAllByText(/^system$/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/^custom$/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^edit$/i })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: /^delete$/i })).toHaveLength(
      2,
    );
    expect(screen.getAllByRole("button", { name: /^edit$/i })).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: /^edit$/i }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByLabelText(/model id/i)).toHaveValue(
      "cx/gpt-5.5",
    );

    await user.clear(within(dialog).getByLabelText(/^name/i));
    await user.type(within(dialog).getByLabelText(/^name/i), "GPT-5.5 Turbo");
    await user.click(
      within(dialog).getByRole("button", { name: /save model/i }),
    );

    await waitFor(() => {
      expect(updateProviderModelMock).toHaveBeenCalledWith("cx", "cx/gpt-5.5", {
        description: "Custom model",
        id: "cx/gpt-5.5",
        name: "GPT-5.5 Turbo",
      });
    });
  });

  it("rejects a provider model before submit when the prefix does not match", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /available models/i });

    await user.click(screen.getByRole("button", { name: /add model/i }));
    const dialog = await screen.findByRole("dialog");

    await user.type(
      within(dialog).getByLabelText(/model id/i),
      "openai/gpt-5.5",
    );
    await user.click(
      within(dialog).getByRole("button", { name: /add model/i }),
    );

    expect(
      await screen.findByText(/model id must start with cx\//i),
    ).toBeInTheDocument();
    expect(createProviderModelMock).not.toHaveBeenCalled();
  });

  it("renders an inline error when the model test fails", async () => {
    const user = userEvent.setup();
    testProviderModelMock.mockRejectedValueOnce({
      message: "Upstream timeout",
    });

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/providers/cx"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /available models/i });

    await user.click(screen.getByRole("button", { name: /^test$/i }));

    expect(await screen.findByText(/upstream timeout/i)).toBeInTheDocument();
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

  it("validates and creates a combo alias", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/combos"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("heading", { level: 2, name: /combo aliases/i });
    await user.click(screen.getByRole("button", { name: /add combo/i }));
    await user.click(screen.getByRole("button", { name: /save combo/i }));

    expect(await screen.findByText(/alias is required/i)).toBeInTheDocument();
    expect(createModelComboMock).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText(/alias/i), "combo/fast");
    await user.type(screen.getByLabelText(/name/i), "Fast Combo");
    await chooseSelectOption(user, 0, "Codex");
    await chooseSelectOption(user, 1, "GPT-5.4 (cx/gpt-5.4)");
    await user.click(screen.getByRole("button", { name: /save combo/i }));

    await waitFor(() => {
      expect(createModelComboMock).toHaveBeenCalledWith({
        alias: "combo/fast",
        description: "",
        name: "Fast Combo",
        targets: [{ enabled: true, model_id: "cx/gpt-5.4", provider_id: "cx" }],
      });
    });
  });

  it("renders combos and reorders target priority", async () => {
    const user = userEvent.setup();
    listModelCombosMock.mockResolvedValueOnce([
      {
        alias: "combo/fast",
        description: "Primary fallback",
        name: "Fast Combo",
        targets: [
          {
            enabled: true,
            model_id: "cx/gpt-5.4",
            priority: 0,
            provider_id: "cx",
          },
          {
            enabled: true,
            model_id: "openai/gpt-4.1",
            priority: 1,
            provider_id: "openai",
          },
        ],
      },
    ]);

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/combos"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByText(/fast combo · combo\/fast/i);
    expect(screen.getByText(/2\/2 enabled/i)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /add combo/i }));
    await user.type(screen.getByLabelText(/alias/i), "combo/balanced");
    await chooseSelectOption(user, 0, "Codex");
    await chooseSelectOption(user, 1, "GPT-5.4 (cx/gpt-5.4)");
    await user.click(screen.getByRole("button", { name: /add target/i }));
    await chooseSelectOption(user, 3, "OpenAI");
    await chooseSelectOption(user, 4, "GPT-4.1 (openai/gpt-4.1)");
    await user.click(screen.getByRole("button", { name: /move target 2 up/i }));

    const dialog = screen.getByRole("dialog");
    const reorderedComboboxes = within(dialog).getAllByRole("combobox");
    expect(reorderedComboboxes[0]).toHaveTextContent("OpenAI");
    expect(reorderedComboboxes[3]).toHaveTextContent("Codex");
  });
});

async function chooseSelectOption(
  user: ReturnType<typeof userEvent.setup>,
  index: number,
  optionName: string,
) {
  const comboboxes = screen.getAllByRole("combobox");
  await user.click(comboboxes[index]);
  await user.click(await screen.findByRole("option", { name: optionName }));
}

import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoutes } from "@/app/routes";
import { useAuthStore } from "@/features/auth/auth-store";
import {
  getSettings,
  settingsQueryKey,
  updateSettings,
} from "@/features/settings/api";
import { renderWithQueryClient } from "@/test/test-utils";

vi.mock("@/features/settings/api", () => ({
  getSettings: vi.fn(),
  settingsQueryKey: ["settings"],
  updateSettings: vi.fn(),
}));

const getSettingsMock = vi.mocked(getSettings);
const updateSettingsMock = vi.mocked(updateSettings);

describe("settings page", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: true,
      token: "secret-token",
    });
    vi.restoreAllMocks();

    getSettingsMock.mockResolvedValue({
      llmLogging: {
        enabled: {
          flow: true,
          thirdParty: false,
        },
      },
      server: {
        listen: ":2232",
        web_ui_dir: "web/dist",
      },
    });
    updateSettingsMock.mockResolvedValue({
      llmLogging: {
        enabled: {
          flow: false,
          thirdParty: true,
        },
      },
      server: {
        listen: ":2232",
        web_ui_dir: "web/dist",
      },
    });
  });

  it("loads current settings and renders server runtime details", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/settings"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByText(":2232");

    expect(screen.getByText("web/dist")).toBeInTheDocument();
    expect(
      screen.getByRole("switch", { name: /flow log toggle/i }),
    ).toHaveAttribute("data-state", "checked");
    expect(
      screen.getByRole("switch", { name: /third-party log toggle/i }),
    ).toHaveAttribute("data-state", "unchecked");
  });

  it("submits updated logging settings and shows success feedback", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/settings"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    const flowSwitch = await screen.findByRole("switch", {
      name: /flow log toggle/i,
    });
    const thirdPartySwitch = screen.getByRole("switch", {
      name: /third-party log toggle/i,
    });

    await user.click(flowSwitch);
    await user.click(thirdPartySwitch);
    await user.click(screen.getByRole("button", { name: /save settings/i }));

    await waitFor(() => {
      expect(updateSettingsMock).toHaveBeenCalledTimes(1);
      expect(updateSettingsMock.mock.calls[0]?.[0]).toEqual({
        llmLogging: {
          enabled: {
            flow: false,
            thirdParty: true,
          },
        },
      });
    });

    expect(
      await screen.findByText(/logging settings saved/i),
    ).toBeInTheDocument();
  });

  it("shows mutation errors from the admin API", async () => {
    const user = userEvent.setup();
    updateSettingsMock.mockRejectedValueOnce(new Error("save failed"));

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/settings"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    await screen.findByRole("switch", { name: /flow log toggle/i });
    await user.click(
      screen.getByRole("switch", { name: /third-party log toggle/i }),
    );
    await user.click(screen.getByRole("button", { name: /save settings/i }));

    expect(await screen.findByText(/save failed/i)).toBeInTheDocument();
  });

  it("exports the stable settings query key", () => {
    expect(settingsQueryKey).toEqual(["settings"]);
  });
});

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
      rtk: {
        enabled: false,
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
      rtk: {
        enabled: true,
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

    await screen.findByText("web/dist");

    expect(screen.getByText(/2232/)).toBeInTheDocument();
    expect(screen.getByText("web/dist")).toBeInTheDocument();
    expect(
      screen.getByRole("switch", { name: /request history details toggle/i }),
    ).toHaveAttribute("data-state", "checked");
    expect(
      screen.getByRole("switch", { name: /provider exchange details toggle/i }),
    ).toHaveAttribute("data-state", "unchecked");
    expect(
      screen.getByRole("switch", { name: /rtk compression toggle/i }),
    ).toHaveAttribute("data-state", "unchecked");
    expect(
      screen.getByRole("link", { name: /learn more about rtk/i }),
    ).toHaveAttribute("href", "https://github.com/rtk-ai/rtk");
    expect(screen.queryByText(/ai_request_flows/i)).not.toBeInTheDocument();
    expect(
      screen.queryByText(/third_party_request_logs/i),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/ai_request_runs/i)).not.toBeInTheDocument();
  });

  it("submits updated logging settings and shows success feedback", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/settings"]}>
        <AppRoutes />
      </MemoryRouter>,
    );

    const flowSwitch = await screen.findByRole("switch", {
      name: /request history details toggle/i,
    });
    const thirdPartySwitch = screen.getByRole("switch", {
      name: /provider exchange details toggle/i,
    });
    const rtkSwitch = screen.getByRole("switch", {
      name: /rtk compression toggle/i,
    });

    await user.click(flowSwitch);
    await user.click(thirdPartySwitch);
    await user.click(rtkSwitch);
    await user.click(screen.getByRole("button", { name: /save changes/i }));

    await waitFor(() => {
      expect(updateSettingsMock).toHaveBeenCalledTimes(1);
      expect(updateSettingsMock.mock.calls[0]?.[0]).toEqual({
        llmLogging: {
          enabled: {
            flow: false,
            thirdParty: true,
          },
        },
        rtk: {
          enabled: true,
        },
      });
    });

    expect(
      await screen.findByText(
        /settings saved\. new requests will use them right away\./i,
      ),
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

    await screen.findByRole("switch", {
      name: /request history details toggle/i,
    });
    await user.click(
      screen.getByRole("switch", {
        name: /provider exchange details toggle/i,
      }),
    );
    await user.click(screen.getByRole("button", { name: /save changes/i }));

    expect(await screen.findByText(/save failed/i)).toBeInTheDocument();
  });

  it("exports the stable settings query key", () => {
    expect(settingsQueryKey).toEqual(["settings"]);
  });
});

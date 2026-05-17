import { apiClient } from "@/shared/api/client";

export const settingsQueryKey = ["settings"] as const;

export type SettingsResponse = {
  llmLogging: {
    enabled: {
      flow: boolean;
      thirdParty: boolean;
    };
  };
  server: {
    listen: string;
    web_ui_dir: string;
  };
};

export type UpdateSettingsPayload = {
  llmLogging: {
    enabled: {
      flow: boolean;
      thirdParty: boolean;
    };
  };
};

export async function getSettings() {
  const response = await apiClient.get<SettingsResponse>("/settings");
  return response.data;
}

export async function updateSettings(payload: UpdateSettingsPayload) {
  const response = await apiClient.put<SettingsResponse>("/settings", payload);
  return response.data;
}

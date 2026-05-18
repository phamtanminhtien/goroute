import { apiClient } from "@/shared/api/client";

export const providersQueryKey = ["providers"] as const;
export const connectionUsageQueryKey = (connectionID: string) =>
  ["connections", connectionID, "usage"] as const;

export type ProviderModel = {
  description: string;
  id: string;
  input_price_per_million_usd?: number;
  name: string;
  output_price_per_million_usd?: number;
  source?: "custom" | "system";
};

export type ProviderConnection = {
  enabled?: boolean;
  expires_in?: number;
  has_access_token: boolean;
  has_api_key: boolean;
  has_refresh_token: boolean;
  id: string;
  last_error_at?: number;
  last_error_category?: string;
  last_error_message?: string;
  name: string;
  problems: string[];
  provider_id: string;
  retry_after?: number;
  status: string;
  token_type?: string;
};

export type ProviderItem = {
  auth_type: string;
  category: string;
  connection_count: number;
  connections: ProviderConnection[];
  default_model: string;
  enabled_connection_count?: number;
  id: string;
  models: ProviderModel[];
  name: string;
};

export type ModelComboTarget = {
  combo_alias?: string;
  connection_id?: string;
  enabled: boolean;
  id?: number;
  model_id: string;
  priority: number;
  provider_id: string;
};

export type ModelCombo = {
  alias: string;
  created_at?: number;
  description: string;
  enabled?: boolean;
  name: string;
  targets: ModelComboTarget[];
  updated_at?: number;
};

export type ProviderUsageQuotaWindow = {
  remaining: number;
  resetAt?: string;
  total: number;
  unlimited: boolean;
  used: number;
};

export type ProviderUsage = {
  limitReached: boolean;
  message?: string;
  plan?: string;
  quotas?: Record<string, ProviderUsageQuotaWindow>;
  reviewLimitReached: boolean;
};

export type ProviderModelTestPayload = {
  model: string;
};

export type ProviderModelPayload = {
  description?: string;
  id: string;
  input_price_per_million_usd?: number;
  name?: string;
  output_price_per_million_usd?: number;
};

export type ModelComboPayload = {
  alias: string;
  description?: string;
  enabled: boolean;
  name?: string;
  targets: Array<{
    connection_id?: string;
    enabled: boolean;
    model_id: string;
    provider_id: string;
  }>;
};

export type ProviderModelTestResult = {
  message: string;
  model: string;
  output_text?: string;
  provider_id: string;
  status: "success" | "error";
};

type ProviderOAuthURLResponse = {
  provider_id: string;
  session_id: string;
  url: string;
};

type ProviderOAuthStart = {
  sessionID: string;
  url: string;
};

type RawProviderItem = Omit<ProviderItem, "connections" | "models"> & {
  connections: ProviderConnection[] | null;
  models: ProviderModel[] | null;
};

export type ConnectionPayload = {
  access_token?: string;
  api_key?: string;
  enabled?: boolean;
  id: string;
  name: string;
  provider_id: string;
  refresh_token?: string;
};

export type ProviderConnectionsEnabledPayload = {
  enabled: boolean;
};

type ListResponse<T> = {
  data: T[];
  object: string;
};

export async function listProviders() {
  const response =
    await apiClient.get<ListResponse<RawProviderItem>>("/providers");
  return response.data.data.map(normalizeProvider);
}

export async function listModelCombos() {
  const response =
    await apiClient.get<ListResponse<ModelCombo>>("/model-combos");
  return response.data.data.map(normalizeModelCombo);
}

export async function createModelCombo(payload: ModelComboPayload) {
  const response = await apiClient.post<ModelCombo>("/model-combos", payload);
  return normalizeModelCombo(response.data);
}

export async function updateModelCombo(
  alias: string,
  payload: ModelComboPayload,
) {
  const response = await apiClient.put<ModelCombo>(
    `/model-combos/${alias}`,
    payload,
  );
  return normalizeModelCombo(response.data);
}

export async function deleteModelCombo(alias: string) {
  await apiClient.delete(`/model-combos/${alias}`);
}

export async function createConnection(payload: ConnectionPayload) {
  const response = await apiClient.post<ProviderConnection>(
    "/connections",
    payload,
  );
  return response.data;
}

export async function updateConnection(id: string, payload: ConnectionPayload) {
  const response = await apiClient.put<ProviderConnection>(
    `/connections/${id}`,
    payload,
  );
  return response.data;
}

export async function updateProviderConnectionsEnabled(
  providerID: string,
  payload: ProviderConnectionsEnabledPayload,
) {
  const response = await apiClient.put<ListResponse<ProviderConnection>>(
    `/providers/${providerID}/connections/enabled`,
    payload,
  );
  return response.data.data;
}

export async function deleteConnection(id: string) {
  await apiClient.delete(`/connections/${id}`);
}

export async function generateProviderOAuthURL(providerID: string) {
  const response = await apiClient.post<ProviderOAuthURLResponse>(
    `/providers/${providerID}/oauth-url`,
  );
  return {
    sessionID: response.data.session_id,
    url: response.data.url,
  } satisfies ProviderOAuthStart;
}

export async function completeOAuthConnection(
  sessionID: string,
  callbackURL: string,
) {
  const response = await apiClient.post<ProviderConnection>(
    "/connections/oauth",
    {
      callback_url: callbackURL,
      session_id: sessionID,
    },
  );
  return response.data;
}

export async function getConnectionUsage(connectionID: string) {
  const response = await apiClient.get<ProviderUsage>(
    `/connections/${connectionID}/usage`,
  );
  return response.data;
}

export async function testProviderModel(
  providerID: string,
  payload: ProviderModelTestPayload,
) {
  const response = await apiClient.post<ProviderModelTestResult>(
    `/providers/${providerID}/test`,
    payload,
  );
  return response.data;
}

export async function createProviderModel(
  providerID: string,
  payload: ProviderModelPayload,
) {
  const response = await apiClient.post<ProviderModel>(
    `/providers/${providerID}/models`,
    payload,
  );
  return response.data;
}

export async function updateProviderModel(
  providerID: string,
  modelID: string,
  payload: ProviderModelPayload,
) {
  const response = await apiClient.put<ProviderModel>(
    `/providers/${providerID}/models/${modelID}`,
    payload,
  );
  return response.data;
}

export async function deleteProviderModel(providerID: string, modelID: string) {
  await apiClient.delete(`/providers/${providerID}/models/${modelID}`);
}

function normalizeProvider(provider: RawProviderItem): ProviderItem {
  return {
    ...provider,
    connection_count: provider.connection_count ?? 0,
    enabled_connection_count:
      provider.enabled_connection_count ??
      (provider.connections ?? []).filter(
        (connection) => connection.enabled ?? true,
      ).length,
    connections: (provider.connections ?? []).map((connection) => ({
      ...connection,
      enabled: connection.enabled ?? true,
    })),
    models: (provider.models ?? []).map((model) => ({
      ...model,
      source: model.source ?? "system",
    })),
  };
}

function normalizeModelCombo(combo: ModelCombo): ModelCombo {
  return {
    ...combo,
    description: combo.description ?? "",
    enabled: combo.enabled ?? true,
    name: combo.name || combo.alias,
    targets: [...(combo.targets ?? [])]
      .sort((first, second) => first.priority - second.priority)
      .map((target) => ({ ...target, enabled: target.enabled ?? true })),
  };
}

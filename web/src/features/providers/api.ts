import { apiClient } from "@/shared/api/client";

export const providersQueryKey = ["providers"] as const;
export const connectionUsageQueryKey = (connectionID: string) =>
  ["connections", connectionID, "usage"] as const;

export type ProviderModel = {
  description: string;
  id: string;
  name: string;
};

export type ProviderConnection = {
  expires_in?: number;
  has_access_token: boolean;
  has_api_key: boolean;
  has_refresh_token: boolean;
  id: string;
  name: string;
  problems: string[];
  provider_id: string;
  status: string;
  token_type?: string;
};

export type ProviderItem = {
  auth_type: string;
  category: string;
  connection_count: number;
  connections: ProviderConnection[];
  default_model: string;
  id: string;
  models: ProviderModel[];
  name: string;
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
  id: string;
  name: string;
  provider_id: string;
  refresh_token?: string;
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

function normalizeProvider(provider: RawProviderItem): ProviderItem {
  return {
    ...provider,
    connection_count: provider.connection_count ?? 0,
    connections: provider.connections ?? [],
    models: provider.models ?? [],
  };
}

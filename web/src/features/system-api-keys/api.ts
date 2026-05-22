import { apiClient } from "@/shared/api/client";

export const systemAPIKeysQueryKey = ["system-api-keys"] as const;

export type SystemAPIKey = {
  created_at: number;
  daily_token_limit: number | null;
  enabled: boolean;
  id: string;
  key: string;
  last_used_at: number;
  monthly_token_limit: number | null;
  name: string;
  requests_per_minute_limit: number | null;
  updated_at: number;
  usage?: SystemAPIKeyUsage;
};

export type SystemAPIKeyUsage = {
  current_minute_requests: number;
  daily_limit_reached: boolean;
  daily_tokens: SystemAPIKeyTokenUsage;
  monthly_limit_reached: boolean;
  monthly_tokens: SystemAPIKeyTokenUsage;
  rate_limit_reached: boolean;
};

export type SystemAPIKeyTokenUsage = {
  limit: number | null;
  remaining: number | null;
  used: number;
};

export type CreateSystemAPIKeyPayload = {
  daily_token_limit?: number | null;
  monthly_token_limit?: number | null;
  name: string;
  requests_per_minute_limit?: number | null;
};

export type UpdateSystemAPIKeyPayload = {
  daily_token_limit?: number | null;
  enabled?: boolean;
  monthly_token_limit?: number | null;
  name?: string;
  requests_per_minute_limit?: number | null;
};

type ListResponse<T> = {
  data: T[];
  object: string;
};

export async function listSystemAPIKeys() {
  const response =
    await apiClient.get<ListResponse<SystemAPIKey>>("/system-api-keys");
  return response.data.data;
}

export async function createSystemAPIKey(payload: CreateSystemAPIKeyPayload) {
  const response = await apiClient.post<SystemAPIKey>(
    "/system-api-keys",
    payload,
  );
  return response.data;
}

export async function updateSystemAPIKey(
  id: string,
  payload: UpdateSystemAPIKeyPayload,
) {
  const response = await apiClient.put<SystemAPIKey>(
    `/system-api-keys/${id}`,
    payload,
  );
  return response.data;
}

export async function deleteSystemAPIKey(id: string) {
  await apiClient.delete(`/system-api-keys/${id}`);
}

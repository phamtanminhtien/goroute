import { apiClient } from "@/shared/api/client";

export const systemAPIKeysQueryKey = ["system-api-keys"] as const;

export type SystemAPIKey = {
  created_at: number;
  enabled: boolean;
  id: string;
  key: string;
  last_used_at: number;
  name: string;
  updated_at: number;
};

export type CreateSystemAPIKeyPayload = {
  name: string;
};

export type UpdateSystemAPIKeyPayload = {
  enabled?: boolean;
  name?: string;
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

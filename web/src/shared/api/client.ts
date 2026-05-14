import axios from "axios";

import { redirectToLogin } from "@/features/auth/auth-session";
import { clearAuthSession, getAuthToken } from "@/features/auth/auth-store";
import { adminAPIBaseURL } from "@/shared/lib/env";

export const apiClient = axios.create({
  baseURL: adminAPIBaseURL,
  headers: {
    Accept: "application/json",
  },
});

apiClient.interceptors.request.use((config) => {
  const token = getAuthToken();

  if (token) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }

  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status as number | undefined;
    const message =
      error.response?.data?.error?.message ??
      error.response?.data?.message ??
      error.message ??
      "Request failed";

    if (status === 401) {
      clearAuthSession();
      redirectToLogin();
    }

    return Promise.reject({
      ...error,
      status,
      message,
    });
  },
);

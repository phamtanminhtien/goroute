import axios from "axios";

import { adminAPIBaseURL } from "@/shared/lib/env";

export const apiClient = axios.create({
  baseURL: adminAPIBaseURL,
  headers: {
    Accept: "application/json",
  },
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

    return Promise.reject({
      ...error,
      status,
      message,
    });
  },
);

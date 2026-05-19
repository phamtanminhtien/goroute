import { AxiosHeaders } from "axios";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { authRedirectEvent } from "@/features/auth/auth-session";
import { useAuthStore } from "@/features/auth/auth-store";
import { apiClient } from "@/shared/api/client";

describe("api client auth behavior", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: false,
      token: null,
    });
  });

  it("attaches bearer token to request headers", async () => {
    useAuthStore.getState().signIn("secret-token");

    const interceptor = apiClient.interceptors.request.handlers?.[0];
    if (!interceptor?.fulfilled) {
      throw new Error("Request interceptor is not registered.");
    }

    const config = await interceptor.fulfilled({
      headers: new AxiosHeaders(),
      url: "/providers",
    });

    expect(config?.headers.get("Authorization")).toBe("Bearer secret-token");
  });

  it("clears session and emits redirect event on 401", async () => {
    useAuthStore.getState().signIn("secret-token");
    const redirectSpy = vi.fn();

    window.addEventListener(authRedirectEvent, redirectSpy);

    const rejected = apiClient.interceptors.response.handlers?.[0];
    if (!rejected?.rejected) {
      throw new Error("Response interceptor is not registered.");
    }

    await expect(
      rejected.rejected({
        message: "Unauthorized",
        response: {
          status: 401,
          data: {
            message: "Unauthorized",
          },
        },
      }),
    ).rejects.toMatchObject({
      status: 401,
      message: "Unauthorized",
    });

    expect(useAuthStore.getState().token).toBeNull();
    expect(redirectSpy).toHaveBeenCalledTimes(1);

    window.removeEventListener(authRedirectEvent, redirectSpy);
  });

  it("rejects response errors as Error instances with server messages", async () => {
    const rejected = apiClient.interceptors.response.handlers?.[0];
    if (!rejected?.rejected) {
      throw new Error("Response interceptor is not registered.");
    }

    await expect(
      rejected.rejected({
        message: "Request failed with status code 400",
        response: {
          status: 400,
          data: {
            error: {
              message:
                'default_model must start with provider prefix "openrouter/"',
            },
          },
        },
      }),
    ).rejects.toMatchObject({
      message: 'default_model must start with provider prefix "openrouter/"',
      status: 400,
    });

    await rejected
      .rejected({
        response: {
          status: 400,
          data: {
            error: {
              message: "base_url must be an absolute URL",
            },
          },
        },
      })
      .catch((error: unknown) => {
        expect(error).toBeInstanceOf(Error);
      });
  });
});

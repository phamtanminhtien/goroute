import { beforeEach, describe, expect, it, vi } from "vitest";

import { authRedirectEvent } from "@/features/auth/auth-session";
import { useAuthStore } from "@/features/auth/auth-store";
import {
  consumeConsoleLogStream,
  parseConsoleLogLine,
  parseSSEFrames,
} from "@/features/logs/stream";

describe("console log stream utilities", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: false,
      token: null,
    });
  });

  it("parses SSE log frames and ignores heartbeats", () => {
    const parsed = parseSSEFrames(
      ": keepalive\n\nevent: log\ndata: first line\n\nevent: log\ndata: second line\n\npartial",
    );

    expect(parsed.events).toEqual([
      { event: "log", data: "first line" },
      { event: "log", data: "second line" },
    ]);
    expect(parsed.rest).toBe("partial");
  });

  it("clears auth state and dispatches redirect event on 401", async () => {
    useAuthStore.getState().signIn("secret-token");
    const redirectSpy = vi.fn();

    window.addEventListener(authRedirectEvent, redirectSpy);
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("", { status: 401 }),
    );

    const states: string[] = [];
    const controller = new AbortController();

    await consumeConsoleLogStream(
      controller.signal,
      () => undefined,
      (state) => states.push(state),
    );

    expect(states).toEqual(["connecting", "unauthorized"]);
    expect(useAuthStore.getState().token).toBeNull();
    expect(redirectSpy).toHaveBeenCalledTimes(1);

    window.removeEventListener(authRedirectEvent, redirectSpy);
  });

  it("parses json console log lines into structured fields", () => {
    const entry = parseConsoleLogLine(
      JSON.stringify({
        level: "info",
        message: "request complete",
        path: "/v1/chat/completions",
        request_id: "req-123",
        time: "2026-05-17T10:00:00Z",
      }),
    );

    expect(entry.level).toBe("info");
    expect(entry.message).toBe("request complete");
    expect(entry.path).toBe("/v1/chat/completions");
    expect(entry.requestID).toBe("req-123");
    expect(entry.timestamp).toBe("2026-05-17T10:00:00Z");
  });
});

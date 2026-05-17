import { redirectToLogin } from "@/features/auth/auth-session";
import { clearAuthSession, getAuthToken } from "@/features/auth/auth-store";
import { adminAPIBaseURL } from "@/shared/lib/env";

export type ConsoleLogStreamEvent = {
  data: string;
  event: string;
};

export type ConsoleLogConnectionState =
  | "connecting"
  | "live"
  | "reconnecting"
  | "disconnected"
  | "unauthorized";

export type ParsedConsoleLogEntry = {
  fields?: Record<string, unknown>;
  level?: string;
  message?: string;
  path?: string;
  raw: string;
  requestID?: string;
  timestamp?: string;
};

export function parseSSEFrames(buffer: string) {
  const normalized = buffer.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  const separatorIndex = normalized.lastIndexOf("\n\n");
  if (separatorIndex < 0) {
    return {
      events: [] as ConsoleLogStreamEvent[],
      rest: normalized,
    };
  }

  const complete = normalized.slice(0, separatorIndex);
  const rest = normalized.slice(separatorIndex + 2);
  const frames = complete.split("\n\n");
  const events: ConsoleLogStreamEvent[] = [];

  for (const frame of frames) {
    const event = parseSSEFrame(frame);
    if (event) {
      events.push(event);
    }
  }

  return { events, rest };
}

function parseSSEFrame(frame: string) {
  const lines = frame.split("\n");
  let event = "message";
  const dataLines: string[] = [];

  for (const line of lines) {
    if (line.startsWith(":")) {
      continue;
    }

    if (line.startsWith("event:")) {
      event = line.slice("event:".length).trim() || "message";
      continue;
    }

    if (line.startsWith("data:")) {
      dataLines.push(line.slice("data:".length).trimStart());
    }
  }

  if (dataLines.length === 0) {
    return null;
  }

  return {
    data: dataLines.join("\n"),
    event,
  };
}

export async function consumeConsoleLogStream(
  signal: AbortSignal,
  onEvent: (event: ConsoleLogStreamEvent) => void,
  onStateChange: (state: ConsoleLogConnectionState) => void,
) {
  const token = getAuthToken();
  if (!token) {
    onStateChange("disconnected");
    return;
  }

  let attempt = 0;

  while (!signal.aborted) {
    onStateChange(attempt === 0 ? "connecting" : "reconnecting");

    try {
      const response = await fetch(`${adminAPIBaseURL}/logs/stream`, {
        headers: {
          Accept: "text/event-stream",
          Authorization: `Bearer ${token}`,
        },
        signal,
      });

      if (response.status === 401) {
        onStateChange("unauthorized");
        clearAuthSession();
        redirectToLogin();
        return;
      }

      if (!response.ok) {
        throw new Error(`Stream request failed with status ${response.status}`);
      }

      if (!response.body) {
        throw new Error("Stream response body unavailable");
      }

      onStateChange("live");
      await readStream(response.body, signal, onEvent);

      if (signal.aborted) {
        return;
      }

      attempt++;
      onStateChange("reconnecting");
      await waitForReconnect(signal);
    } catch (error) {
      if (signal.aborted) {
        return;
      }

      attempt++;
      onStateChange("reconnecting");
      await waitForReconnect(signal);
      if (error instanceof Error && error.name === "AbortError") {
        return;
      }
    }
  }
}

async function readStream(
  body: ReadableStream<Uint8Array>,
  signal: AbortSignal,
  onEvent: (event: ConsoleLogStreamEvent) => void,
) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let pending = "";

  try {
    while (!signal.aborted) {
      const { done, value } = await reader.read();
      if (done) {
        return;
      }

      pending += decoder.decode(value, { stream: true });
      const parsed = parseSSEFrames(pending);
      pending = parsed.rest;

      for (const event of parsed.events) {
        onEvent(event);
      }
    }
  } finally {
    reader.releaseLock();
  }
}

function waitForReconnect(signal: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    const timeoutID = window.setTimeout(() => {
      signal.removeEventListener("abort", handleAbort);
      resolve();
    }, 1000);

    function handleAbort() {
      window.clearTimeout(timeoutID);
      signal.removeEventListener("abort", handleAbort);
      reject(new DOMException("The operation was aborted.", "AbortError"));
    }

    signal.addEventListener("abort", handleAbort);
  });
}

export function parseConsoleLogLine(line: string): ParsedConsoleLogEntry {
  const trimmed = line.trim();
  if (trimmed === "") {
    return { raw: line };
  }

  try {
    const parsed = JSON.parse(trimmed);
    if (!isJSONObject(parsed)) {
      return { raw: line };
    }

    return {
      fields: parsed,
      level: readStringField(parsed.level),
      message: readStringField(parsed.message),
      path: readStringField(parsed.path),
      raw: line,
      requestID: readStringField(parsed.request_id),
      timestamp: readStringField(parsed.time),
    };
  } catch {
    return { raw: line };
  }
}

function isJSONObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readStringField(value: unknown) {
  return typeof value === "string" && value.trim() !== "" ? value : undefined;
}

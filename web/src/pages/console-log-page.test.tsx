import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  appendConsoleLogEntry,
  ConsoleLogPage,
  maxConsoleLogLines,
} from "@/pages/console-log-page";
import { renderWithQueryClient } from "@/test/test-utils";

const consumeConsoleLogStreamMock = vi.fn();

vi.mock("@/features/logs/stream", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@/features/logs/stream")>();
  return {
    ...actual,
    consumeConsoleLogStream: (...args: unknown[]) =>
      consumeConsoleLogStreamMock(...args),
  };
});

describe("console log page", () => {
  beforeEach(() => {
    consumeConsoleLogStreamMock.mockReset();
  });

  it("renders streamed log lines and clears the local buffer", async () => {
    const user = userEvent.setup();
    consumeConsoleLogStreamMock.mockImplementation(
      async (
        _signal: AbortSignal,
        onEvent: (event: { data: string; event: string }) => void,
        onStateChange: (state: string) => void,
      ) => {
        onStateChange("live");
        onEvent({
          event: "log",
          data: JSON.stringify({
            level: "info",
            message: "first line",
            path: "/admin/api/logs/stream",
            request_id: "req-1",
            time: "2026-05-17T15:30:00Z",
          }),
        });
        onEvent({ event: "log", data: "second line" });
      },
    );

    renderWithQueryClient(
      <MemoryRouter>
        <ConsoleLogPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("first line")).toBeInTheDocument();
    expect(screen.getByText("INFO")).toBeInTheDocument();
    expect(screen.getByText("req=req-1")).toBeInTheDocument();
    expect(screen.getByText("second line")).toBeInTheDocument();
    expect(screen.getByText(/^Live$/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /clear/i }));

    expect(screen.getByText(/waiting for backend logs/i)).toBeInTheDocument();
  });

  it("caps the in-memory console line buffer", () => {
    let entries: Array<{ raw: string }> = [];

    for (let index = 0; index < maxConsoleLogLines + 25; index++) {
      entries = appendConsoleLogEntry(entries, { raw: `line ${index}` });
    }

    expect(entries).toHaveLength(maxConsoleLogLines);
    expect(entries[0]?.raw).toBe("line 25");
    expect(entries.at(-1)?.raw).toBe(`line ${maxConsoleLogLines + 24}`);
  });
});

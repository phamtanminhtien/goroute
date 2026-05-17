import { Trash2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  type ConsoleLogConnectionState,
  consumeConsoleLogStream,
  parseConsoleLogLine,
  type ParsedConsoleLogEntry,
} from "@/features/logs/stream";
import { Button } from "@/shared/ui/button";
import { PageHeader } from "@/shared/ui/page-header";
import { SurfaceCard } from "@/shared/ui/surface-card";

export const maxConsoleLogLines = 750;

const statusLabels: Record<ConsoleLogConnectionState, string> = {
  connecting: "Connecting",
  disconnected: "Disconnected",
  live: "Live",
  reconnecting: "Reconnecting",
  unauthorized: "Unauthorized",
};

const statusToneClasses: Record<ConsoleLogConnectionState, string> = {
  connecting: "text-fg-secondary",
  disconnected: "text-fg-secondary",
  live: "text-[var(--success)]",
  reconnecting: "text-[var(--warning)]",
  unauthorized: "text-[var(--error)]",
};

export function ConsoleLogPage() {
  const [entries, setEntries] = useState<ParsedConsoleLogEntry[]>([]);
  const [status, setStatus] = useState<ConsoleLogConnectionState>("connecting");
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const shouldStickToBottomRef = useRef(true);

  useEffect(() => {
    const controller = new AbortController();

    void consumeConsoleLogStream(
      controller.signal,
      (event) => {
        if (event.event !== "log") {
          return;
        }

        setEntries((currentEntries) =>
          appendConsoleLogEntry(
            currentEntries,
            parseConsoleLogLine(event.data),
          ),
        );
      },
      setStatus,
    );

    return () => controller.abort();
  }, []);

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport || !shouldStickToBottomRef.current) {
      return;
    }

    viewport.scrollTop = viewport.scrollHeight;
  }, [entries]);

  const statusClassName = useMemo(() => statusToneClasses[status], [status]);

  return (
    <section className="space-y-4 pb-6">
      <PageHeader
        description="Live backend application logs from the admin SSE stream."
        eyebrow="Observability"
        title="Console Log"
      >
        <div className="flex items-center gap-2">
          <span
            className={`border-border bg-bg-secondary rounded-md border px-2.5 py-1 text-xs font-medium ${statusClassName}`}
          >
            {statusLabels[status]}
          </span>
          <Button
            leadingIcon={<Trash2 className="size-4" />}
            onClick={() => setEntries([])}
            tone="secondary"
          >
            Clear
          </Button>
        </div>
      </PageHeader>

      <SurfaceCard className="overflow-hidden" tone="glass">
        <div className="border-border/70 flex items-center justify-between gap-3 border-b px-4 py-3">
          <div className="text-fg-secondary text-sm">
            {entries.length} buffered
          </div>
          <div className="text-fg-muted text-xs">
            JSON lines are formatted when possible
          </div>
        </div>

        <div
          aria-label="Console log stream"
          className="max-h-[70vh] overflow-y-auto bg-[#111111]"
          onScroll={(event) => {
            const viewport = event.currentTarget;
            const distanceToBottom =
              viewport.scrollHeight -
              (viewport.scrollTop + viewport.clientHeight);
            shouldStickToBottomRef.current = distanceToBottom < 48;
          }}
          ref={viewportRef}
        >
          {entries.length === 0 ? (
            <div className="text-fg-muted px-4 py-6 text-sm">
              Waiting for backend logs.
            </div>
          ) : (
            <div>
              {entries.map((entry, index) => (
                <ConsoleLogRow
                  entry={entry}
                  key={`${entry.timestamp ?? "line"}-${index}`}
                />
              ))}
            </div>
          )}
        </div>
      </SurfaceCard>
    </section>
  );
}

function ConsoleLogRow({ entry }: { entry: ParsedConsoleLogEntry }) {
  const level = entry.level?.toUpperCase() ?? "RAW";
  const levelClassName = levelToneClassName(level);
  const message = entry.message ?? entry.raw;
  const meta = [
    entry.timestamp ? formatTimestamp(entry.timestamp) : null,
    entry.requestID ? `req=${entry.requestID}` : null,
    entry.path ? entry.path : null,
  ].filter(Boolean);

  return (
    <div className="border-b border-white/6 px-4 py-2 font-mono text-xs leading-5 text-slate-200">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className={`min-w-14 font-semibold ${levelClassName}`}>
          {level}
        </span>
        <span className="flex-1 break-words text-slate-100">{message}</span>
      </div>
      {meta.length > 0 ? (
        <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-slate-400">
          {meta.map((item) => (
            <span key={item}>{item}</span>
          ))}
        </div>
      ) : null}
      {entry.fields ? (
        <details className="mt-1 text-[11px] text-slate-400">
          <summary className="cursor-pointer select-none">raw json</summary>
          <pre className="mt-1 overflow-x-auto break-words whitespace-pre-wrap text-slate-500">
            {JSON.stringify(entry.fields, null, 2)}
          </pre>
        </details>
      ) : null}
    </div>
  );
}

function levelToneClassName(level: string) {
  switch (level) {
    case "ERROR":
    case "FATAL":
      return "text-[var(--error)]";
    case "WARN":
    case "WARNING":
      return "text-[var(--warning)]";
    case "DEBUG":
      return "text-slate-400";
    default:
      return "text-[var(--success)]";
  }
}

function formatTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function appendConsoleLogEntry(
  entries: ParsedConsoleLogEntry[],
  entry: ParsedConsoleLogEntry,
) {
  const nextEntries = [...entries, entry];
  if (nextEntries.length <= maxConsoleLogLines) {
    return nextEntries;
  }

  return nextEntries.slice(nextEntries.length - maxConsoleLogLines);
}

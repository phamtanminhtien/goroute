import { type ProviderConnection } from "@/features/providers/api";
import { StatusBadge } from "@/shared/ui/status-badge";

export function ConnectionRuntimeStatus({
  connection,
}: {
  connection: ProviderConnection;
}) {
  const latestError = connection.last_error_message?.trim();
  const category = connection.last_error_category?.trim();
  if (!latestError && !category) {
    return null;
  }

  const retryLabel = formatRetryRemaining(connection.retry_after);
  const lastLabel = category || latestError || "Runtime error";

  return (
    <div className="flex flex-wrap items-center gap-1.5 text-[10px]">
      <StatusBadge size="sm" tone="warning">
        Last: {lastLabel}
      </StatusBadge>
      {retryLabel ? (
        <StatusBadge size="sm" tone="info">
          Retry in: {retryLabel}
        </StatusBadge>
      ) : null}
      {latestError && latestError !== lastLabel ? (
        <span className="text-fg-muted truncate">{latestError}</span>
      ) : null}
    </div>
  );
}

function formatRetryRemaining(retryAfter?: number) {
  if (!retryAfter) {
    return "";
  }

  const diffMs = retryAfter * 1000 - Date.now();
  if (diffMs <= 0) {
    return "";
  }

  const totalSeconds = Math.ceil(diffMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }

  return `${seconds}s`;
}

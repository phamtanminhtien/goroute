import type { HTMLAttributes, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";

export function DetailList({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("space-y-3", className)} {...props} />;
}

export function KeyValueRow({
  className,
  label,
  value,
  ...props
}: HTMLAttributes<HTMLDivElement> & {
  label: ReactNode;
  value: ReactNode;
}) {
  return (
    <div
      className={cn(
        "border-border/85 bg-bg-primary/75 flex flex-col gap-2 rounded-[22px] border px-4 py-4 sm:flex-row sm:items-center sm:justify-between",
        className,
      )}
      {...props}
    >
      <p className="text-fg-muted text-[11px] font-semibold tracking-[0.2em] uppercase">
        {label}
      </p>
      <p className="text-fg-primary text-sm font-medium">{value}</p>
    </div>
  );
}

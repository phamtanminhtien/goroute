import type { HTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";

export function Skeleton({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "border-border/85 bg-bg-primary/72 animate-pulse rounded-[22px] border",
        className,
      )}
      {...props}
    />
  );
}

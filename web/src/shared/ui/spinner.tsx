import type { HTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";

export function Spinner({
  className,
  ...props
}: HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-flex size-4 animate-spin rounded-full border-2 border-current border-r-transparent",
        className,
      )}
      {...props}
    />
  );
}

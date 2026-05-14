import type { ReactNode } from "react";

import { createVariant } from "@/shared/lib/create-variant";

type StatusTone = "info" | "success" | "warning";
type StatusSize = "sm" | "md";

const badgeVariants = createVariant(
  "inline-flex items-center rounded-full border font-semibold",
  {
    defaultVariants: {
      size: "md",
      tone: "info",
    },
    variants: {
      size: {
        md: "gap-2 px-3 py-1 text-xs",
        sm: "gap-1.5 px-2.5 py-1 text-[11px]",
      },
      tone: {
        info: "border-primary/12 bg-primary/8 text-primary",
        success:
          "border-emerald-500/18 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
        warning:
          "border-amber-500/18 bg-amber-500/10 text-amber-700 dark:text-amber-300",
      },
    },
  },
);

export function StatusBadge({
  children,
  tone = "info",
  size = "md",
}: {
  children: ReactNode;
  size?: StatusSize;
  tone?: StatusTone;
}) {
  return <span className={badgeVariants({ size, tone })}>{children}</span>;
}

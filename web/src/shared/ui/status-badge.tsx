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
          "border-[color:color-mix(in_srgb,var(--success)_20%,transparent)] bg-[color:color-mix(in_srgb,var(--success)_12%,transparent)] text-[var(--success)]",
        warning:
          "border-[color:color-mix(in_srgb,var(--warning)_22%,transparent)] bg-[color:color-mix(in_srgb,var(--warning)_10%,transparent)] text-[var(--warning)]",
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

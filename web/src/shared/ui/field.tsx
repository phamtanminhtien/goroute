import type { PropsWithChildren, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";
import { createVariant } from "@/shared/lib/create-variant";

type FieldProps = PropsWithChildren<{
  error?: ReactNode;
  help?: ReactNode;
  label: ReactNode;
  labelSuffix?: ReactNode;
  required?: boolean;
  spacing?: "sm" | "md";
}>;

const fieldVariants = createVariant("flex flex-col", {
  defaultVariants: {
    spacing: "md",
  },
  variants: {
    spacing: {
      md: "gap-2",
      sm: "gap-1.5",
    },
  },
});

export function Field({
  children,
  error,
  help,
  label,
  labelSuffix,
  required,
  spacing = "md",
}: FieldProps) {
  return (
    <label className={fieldVariants({ spacing })}>
      <span className="text-fg-primary flex items-center justify-between gap-4 text-sm font-medium">
        <span className="inline-flex items-center gap-2">
          {label}
          {required ? <span className="text-primary">*</span> : null}
        </span>
        {labelSuffix ? (
          <span className="text-fg-muted text-xs font-medium">
            {labelSuffix}
          </span>
        ) : null}
      </span>
      {children}
      {help ? (
        <span
          className={cn(
            "text-fg-muted text-xs leading-5",
            error && "text-rose-700",
          )}
        >
          {error ?? help}
        </span>
      ) : error ? (
        <span className="text-xs leading-5 text-rose-700">{error}</span>
      ) : null}
    </label>
  );
}

import type { InputHTMLAttributes } from "react";

import { createVariant } from "@/shared/lib/create-variant";

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  inputSize?: "md" | "lg";
  tone?: "default" | "subtle";
};

const inputVariants = createVariant(
  "w-full min-h-[var(--control-height)] rounded-[var(--radius-control)] border px-[var(--control-padding-x)] py-[var(--control-padding-y)] text-sm text-fg-primary placeholder:text-fg-muted disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-none focus-visible:ring-4 [--tw-ring-color:var(--focus-ring)] [background:var(--field-bg)] [border-color:var(--field-border)] transition-[border-color,background-color,box-shadow] duration-150 ease-[cubic-bezier(0.4,0,0.2,1)] focus:[background:var(--field-bg-focus)] focus:[border-color:var(--field-border-focus)] focus:[box-shadow:0_0_0_4px_color-mix(in_srgb,var(--primary)_10%,transparent)]",
  {
    defaultVariants: {
      inputSize: "md",
      tone: "default",
    },
    variants: {
      inputSize: {
        lg: "min-h-[var(--control-height-lg)]",
        md: "",
      },
      tone: {
        default: "",
        subtle: "bg-bg-tertiary/70",
      },
    },
  },
);

export function Input({
  className,
  inputSize = "md",
  tone = "default",
  ...props
}: InputProps) {
  return (
    <input
      className={inputVariants({ inputSize, tone }, className)}
      {...props}
    />
  );
}

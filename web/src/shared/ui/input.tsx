import type { InputHTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";
import { createVariant } from "@/shared/lib/create-variant";
import { controlBaseClassName } from "@/shared/ui/ui-base";

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  inputSize?: "md" | "lg";
  tone?: "default" | "subtle";
};

const inputVariants = createVariant(
  cn(
    "w-full",
    controlBaseClassName,
    "transition-[border-color,background-color,box-shadow]",
  ),
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

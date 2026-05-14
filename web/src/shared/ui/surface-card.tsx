import { Slot } from "@radix-ui/react-slot";
import type { HTMLAttributes } from "react";

import { createVariant } from "@/shared/lib/create-variant";

type SurfaceCardProps = HTMLAttributes<HTMLDivElement> & {
  asChild?: boolean;
  padding?: "none" | "md";
  tone?: "glass" | "solid";
};

const surfaceVariants = createVariant("", {
  defaultVariants: {
    padding: "none",
    tone: "glass",
  },
  variants: {
    padding: {
      md: "p-5 sm:p-6",
      none: "",
    },
    tone: {
      glass:
        "rounded-[var(--radius-surface)] border border-border/90 bg-glass-bg backdrop-blur-[var(--glass-blur)] shadow-[var(--glass-shadow)]",
      solid:
        "rounded-[var(--radius-surface)] border border-border/90 bg-bg-secondary shadow-[var(--shadow-md)]",
    },
  },
});

export function SurfaceCard({
  asChild = false,
  className,
  padding = "none",
  tone = "glass",
  ...props
}: SurfaceCardProps) {
  const Comp = asChild ? Slot : "div";

  return (
    <Comp
      className={surfaceVariants({ padding, tone }, className)}
      {...props}
    />
  );
}

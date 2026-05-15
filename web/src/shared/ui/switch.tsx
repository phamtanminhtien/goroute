import * as SwitchPrimitive from "@radix-ui/react-switch";
import type { ComponentPropsWithoutRef } from "react";

import { cn } from "@/shared/lib/cn";

type SwitchProps = ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>;

export function Switch({ className, ...props }: SwitchProps) {
  return (
    <SwitchPrimitive.Root
      className={cn(
        "peer data-[state=checked]:border-primary/25 data-[state=checked]:bg-primary/18 inline-flex h-7 w-12 shrink-0 items-center rounded-full border border-[var(--field-border)] bg-[var(--field-bg)] p-1 [--tw-ring-color:var(--focus-ring)] transition-[background-color,border-color,box-shadow] duration-150 ease-[cubic-bezier(0.4,0,0.2,1)] focus-visible:ring-4 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-60",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb className="block size-5 rounded-full bg-white shadow-[var(--shadow-sm)] transition-transform duration-150 data-[state=checked]:translate-x-5" />
    </SwitchPrimitive.Root>
  );
}

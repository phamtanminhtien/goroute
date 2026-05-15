import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import type { ComponentPropsWithoutRef, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";

export const RadioGroup = RadioGroupPrimitive.Root;

type RadioGroupItemProps = ComponentPropsWithoutRef<
  typeof RadioGroupPrimitive.Item
> & {
  description?: ReactNode;
  label: ReactNode;
};

export function RadioGroupItem({
  className,
  description,
  label,
  ...props
}: RadioGroupItemProps) {
  return (
    <label
      className={cn(
        "hover:border-border-hover has-[[data-state=checked]]:border-primary/28 has-[[data-state=checked]]:bg-primary/8 flex cursor-pointer items-start gap-3 rounded-[18px] border border-[var(--field-border)] bg-[var(--field-bg)] px-4 py-3 transition-colors duration-150",
        props.disabled && "cursor-not-allowed opacity-60",
        className,
      )}
    >
      <RadioGroupPrimitive.Item
        className="bg-bg-secondary data-[state=checked]:border-primary mt-0.5 inline-flex size-5 shrink-0 items-center justify-center rounded-full border border-[var(--field-border)] [--tw-ring-color:var(--focus-ring)] focus-visible:ring-4 focus-visible:outline-none"
        {...props}
      >
        <RadioGroupPrimitive.Indicator className="flex items-center justify-center">
          <span className="bg-primary size-2.5 rounded-full" />
        </RadioGroupPrimitive.Indicator>
      </RadioGroupPrimitive.Item>
      <span className="space-y-1">
        <span className="text-fg-primary block text-sm font-medium">
          {label}
        </span>
        {description ? (
          <span className="text-fg-secondary block text-sm leading-6">
            {description}
          </span>
        ) : null}
      </span>
    </label>
  );
}

import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/shared/lib/cn";
import {
  controlBaseClassName,
  overlayPanelClassName,
} from "@/shared/ui/ui-base";

export type SelectOption = {
  disabled?: boolean;
  icon?: ReactNode;
  keywords?: string[];
  label: string;
  value: string;
};

type SelectProps = {
  className?: string;
  disabled?: boolean;
  onValueChange?: (value: string) => void;
  options: SelectOption[];
  placeholder?: string;
  value?: string;
};

export function Select({
  className,
  disabled,
  onValueChange,
  options,
  placeholder = "Select an option",
  value,
}: SelectProps) {
  return (
    <SelectPrimitive.Root
      disabled={disabled}
      onValueChange={onValueChange}
      value={value}
    >
      <SelectPrimitive.Trigger
        className={cn(
          controlBaseClassName,
          "inline-flex w-full items-center justify-between gap-3 text-left",
          className,
        )}
      >
        <SelectPrimitive.Value placeholder={placeholder} />
        <SelectPrimitive.Icon className="text-fg-muted">
          <ChevronDown className="size-4" />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>
      <SelectPrimitive.Portal>
        <SelectPrimitive.Content
          className={cn(
            overlayPanelClassName,
            "z-50 max-h-[320px] min-w-[var(--radix-select-trigger-width)] overflow-hidden p-2 outline-none",
          )}
          position="popper"
          sideOffset={10}
        >
          <SelectPrimitive.Viewport className="space-y-1">
            {options.map((option) => (
              <SelectPrimitive.Item
                className="text-fg-secondary data-[highlighted]:text-fg-primary flex min-h-10 cursor-pointer items-center justify-between gap-3 rounded-[14px] px-3 py-2 text-sm transition-colors outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-white/[0.06]"
                disabled={option.disabled}
                key={option.value}
                value={option.value}
              >
                <div className="flex items-center gap-2">
                  {option.icon ? (
                    <span className="shrink-0">{option.icon}</span>
                  ) : null}
                  <SelectPrimitive.ItemText>
                    {option.label}
                  </SelectPrimitive.ItemText>
                </div>
                <SelectPrimitive.ItemIndicator>
                  <Check className="text-primary size-4" />
                </SelectPrimitive.ItemIndicator>
              </SelectPrimitive.Item>
            ))}
          </SelectPrimitive.Viewport>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}

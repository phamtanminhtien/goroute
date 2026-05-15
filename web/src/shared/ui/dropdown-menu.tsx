import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import { Check, ChevronRight } from "lucide-react";
import type { ComponentPropsWithoutRef, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";
import { overlayPanelClassName } from "@/shared/ui/ui-base";

export const DropdownMenu = DropdownMenuPrimitive.Root;
export const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;
export const DropdownMenuPortal = DropdownMenuPrimitive.Portal;
export const DropdownMenuSub = DropdownMenuPrimitive.Sub;
export const DropdownMenuRadioGroup = DropdownMenuPrimitive.RadioGroup;

type DropdownMenuContentProps = ComponentPropsWithoutRef<
  typeof DropdownMenuPrimitive.Content
>;

export function DropdownMenuContent({
  align = "end",
  className,
  collisionPadding = 12,
  sideOffset = 8,
  ...props
}: DropdownMenuContentProps) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        align={align}
        className={cn(
          overlayPanelClassName,
          "z-50 min-w-[220px] p-2 outline-none",
          className,
        )}
        collisionPadding={collisionPadding}
        sideOffset={sideOffset}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  );
}

type DropdownMenuItemProps = ComponentPropsWithoutRef<
  typeof DropdownMenuPrimitive.Item
> & {
  destructive?: boolean;
  icon?: ReactNode;
};

export function DropdownMenuItem({
  className,
  destructive = false,
  icon,
  children,
  ...props
}: DropdownMenuItemProps) {
  return (
    <DropdownMenuPrimitive.Item
      className={cn(
        "data-[highlighted]:text-fg-primary flex min-h-10 cursor-pointer items-center gap-2 rounded-[14px] px-3 py-2 text-sm transition-colors outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-white/[0.06]",
        destructive
          ? "text-rose-300 data-[highlighted]:bg-rose-500/12"
          : "text-fg-secondary",
        className,
      )}
      {...props}
    >
      {icon ? <span className="shrink-0">{icon}</span> : null}
      <span>{children}</span>
    </DropdownMenuPrimitive.Item>
  );
}

export function DropdownMenuCheckboxItem({
  children,
  className,
  ...props
}: ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.CheckboxItem>) {
  return (
    <DropdownMenuPrimitive.CheckboxItem
      className={cn(
        "text-fg-secondary data-[highlighted]:text-fg-primary flex min-h-10 cursor-pointer items-center gap-2 rounded-[14px] px-3 py-2 pr-9 text-sm transition-colors outline-none data-[highlighted]:bg-white/[0.06]",
        className,
      )}
      {...props}
    >
      <DropdownMenuPrimitive.ItemIndicator>
        <Check className="text-primary size-4" />
      </DropdownMenuPrimitive.ItemIndicator>
      <span>{children}</span>
    </DropdownMenuPrimitive.CheckboxItem>
  );
}

export function DropdownMenuLabel({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Label>) {
  return (
    <DropdownMenuPrimitive.Label
      className={cn(
        "text-fg-muted px-3 py-2 text-[11px] font-semibold tracking-[0.18em] uppercase",
        className,
      )}
      {...props}
    />
  );
}

export function DropdownMenuSeparator({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Separator>) {
  return (
    <DropdownMenuPrimitive.Separator
      className={cn("bg-border/80 my-1 h-px", className)}
      {...props}
    />
  );
}

export function DropdownMenuSubTrigger({
  className,
  children,
  ...props
}: ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.SubTrigger>) {
  return (
    <DropdownMenuPrimitive.SubTrigger
      className={cn(
        "text-fg-secondary data-[highlighted]:text-fg-primary flex min-h-10 cursor-pointer items-center justify-between rounded-[14px] px-3 py-2 text-sm transition-colors outline-none data-[highlighted]:bg-white/[0.06]",
        className,
      )}
      {...props}
    >
      <span>{children}</span>
      <ChevronRight className="size-4" />
    </DropdownMenuPrimitive.SubTrigger>
  );
}

export function DropdownMenuSubContent({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.SubContent>) {
  return (
    <DropdownMenuPrimitive.SubContent
      className={cn(
        overlayPanelClassName,
        "z-50 min-w-[200px] p-2 outline-none",
        className,
      )}
      {...props}
    />
  );
}

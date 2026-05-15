import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ComponentPropsWithoutRef } from "react";

import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import { overlayPanelClassName } from "@/shared/ui/ui-base";

export const Drawer = DialogPrimitive.Root;
export const DrawerTrigger = DialogPrimitive.Trigger;
export const DrawerClose = DialogPrimitive.Close;

export function DrawerContent({
  children,
  className,
  side = "right",
  ...props
}: ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & {
  side?: "left" | "right";
}) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/58 backdrop-blur-[3px]" />
      <DialogPrimitive.Content
        className={cn(
          overlayPanelClassName,
          "fixed top-0 z-50 h-dvh w-[min(92vw,420px)] p-6 outline-none",
          side === "right" ? "right-0" : "left-0",
          className,
        )}
        {...props}
      >
        {children}
        <DialogPrimitive.Close asChild>
          <Button
            aria-label="Close drawer"
            className="border-border/80 text-fg-secondary absolute top-4 right-4 h-9 bg-transparent shadow-none hover:bg-white/[0.05]"
            iconOnly
            leadingIcon={<X className="size-4" />}
            ripple={false}
            tone="secondary"
          />
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

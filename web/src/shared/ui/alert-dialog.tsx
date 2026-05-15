import * as AlertDialogPrimitive from "@radix-ui/react-alert-dialog";
import type { ComponentPropsWithoutRef } from "react";

import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import { overlayPanelClassName } from "@/shared/ui/ui-base";

export const AlertDialog = AlertDialogPrimitive.Root;
export const AlertDialogTrigger = AlertDialogPrimitive.Trigger;
export const AlertDialogPortal = AlertDialogPrimitive.Portal;
export const AlertDialogCancel = AlertDialogPrimitive.Cancel;
export const AlertDialogAction = AlertDialogPrimitive.Action;

export function AlertDialogContent({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Content>) {
  return (
    <AlertDialogPortal>
      <AlertDialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/58 backdrop-blur-[3px]" />
      <AlertDialogPrimitive.Content
        className={cn(
          overlayPanelClassName,
          "fixed top-1/2 left-1/2 z-50 w-[min(92vw,540px)] -translate-x-1/2 -translate-y-1/2 p-6 outline-none",
          className,
        )}
        {...props}
      />
    </AlertDialogPortal>
  );
}

export function AlertDialogHeader({
  className,
  ...props
}: ComponentPropsWithoutRef<"div">) {
  return <div className={cn("space-y-1.5", className)} {...props} />;
}

export function AlertDialogTitle({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Title>) {
  return (
    <AlertDialogPrimitive.Title
      className={cn(
        "text-fg-primary text-lg font-semibold tracking-[-0.03em]",
        className,
      )}
      {...props}
    />
  );
}

export function AlertDialogDescription({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Description>) {
  return (
    <AlertDialogPrimitive.Description
      className={cn("text-fg-secondary text-sm leading-6", className)}
      {...props}
    />
  );
}

export function AlertDialogFooter({
  className,
  ...props
}: ComponentPropsWithoutRef<"div">) {
  return (
    <div
      className={cn("mt-6 flex flex-wrap justify-end gap-2", className)}
      {...props}
    />
  );
}

export function AlertDialogCancelButton({
  children = "Cancel",
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Cancel>) {
  return (
    <AlertDialogPrimitive.Cancel asChild>
      <Button className={className} tone="secondary" {...props}>
        {children}
      </Button>
    </AlertDialogPrimitive.Cancel>
  );
}

export function AlertDialogActionButton({
  children = "Confirm",
  className,
  tone = "primary",
  ...props
}: ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Action> & {
  tone?: "ghost" | "primary" | "secondary";
}) {
  return (
    <AlertDialogPrimitive.Action asChild>
      <Button className={className} tone={tone} {...props}>
        {children}
      </Button>
    </AlertDialogPrimitive.Action>
  );
}

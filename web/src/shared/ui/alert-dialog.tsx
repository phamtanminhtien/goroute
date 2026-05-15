import * as AlertDialogPrimitive from "@radix-ui/react-alert-dialog";
import { motion, useReducedMotion } from "motion/react";
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
  children,
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Content>) {
  const prefersReducedMotion = useReducedMotion();

  return (
    <AlertDialogPortal>
      <AlertDialogPrimitive.Overlay asChild>
        <motion.div
          animate={{ opacity: 1 }}
          className="fixed inset-0 z-50 bg-black/58 backdrop-blur-[3px]"
          initial={{ opacity: 0 }}
          transition={{
            duration: prefersReducedMotion ? 0.12 : 0.18,
            ease: "easeOut",
          }}
        />
      </AlertDialogPrimitive.Overlay>
      <AlertDialogPrimitive.Content asChild {...props}>
        <motion.div
          animate={{
            opacity: 1,
            scale: 1,
            x: "-50%",
            y: "-50%",
          }}
          className={cn(
            overlayPanelClassName,
            "fixed top-1/2 left-1/2 z-50 w-[min(92vw,540px)] p-6 outline-none",
            className,
          )}
          initial={
            prefersReducedMotion
              ? { opacity: 0, scale: 1, x: "-50%", y: "-50%" }
              : { opacity: 0, scale: 0.97, x: "-50%", y: "calc(-50% + 10px)" }
          }
          transition={
            prefersReducedMotion
              ? { duration: 0.14, ease: "easeOut" }
              : {
                  damping: 30,
                  mass: 0.92,
                  stiffness: 380,
                  type: "spring",
                }
          }
        >
          {children}
        </motion.div>
      </AlertDialogPrimitive.Content>
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

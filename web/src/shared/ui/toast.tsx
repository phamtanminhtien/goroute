import * as ToastPrimitive from "@radix-ui/react-toast";
import { X } from "lucide-react";
import type { ComponentPropsWithoutRef } from "react";

import { cn } from "@/shared/lib/cn";

export const ToastProvider = ToastPrimitive.Provider;
export const ToastViewport = ({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof ToastPrimitive.Viewport>) => (
  <ToastPrimitive.Viewport
    className={cn(
      "fixed right-4 bottom-4 z-[100] flex w-[min(92vw,380px)] flex-col gap-3 outline-none",
      className,
    )}
    {...props}
  />
);

export const Toast = ({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof ToastPrimitive.Root>) => (
  <ToastPrimitive.Root
    className={cn(
      "border-border/90 bg-glass-bg text-fg-primary rounded-[20px] border p-4 shadow-[var(--glass-shadow)] backdrop-blur-[var(--glass-blur)]",
      className,
    )}
    {...props}
  />
);

export const ToastTitle = ({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof ToastPrimitive.Title>) => (
  <ToastPrimitive.Title
    className={cn("text-fg-primary text-sm font-semibold", className)}
    {...props}
  />
);

export const ToastDescription = ({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof ToastPrimitive.Description>) => (
  <ToastPrimitive.Description
    className={cn("text-fg-secondary mt-1 text-sm leading-6", className)}
    {...props}
  />
);

export function ToastClose({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof ToastPrimitive.Close>) {
  return (
    <ToastPrimitive.Close
      className={cn(
        "text-fg-secondary hover:text-fg-primary absolute top-3 right-3 inline-flex size-8 items-center justify-center rounded-full transition-colors hover:bg-white/[0.05]",
        className,
      )}
      {...props}
    >
      <X className="size-4" />
    </ToastPrimitive.Close>
  );
}

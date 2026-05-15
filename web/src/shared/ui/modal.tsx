import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ComponentPropsWithoutRef, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import { overlayPanelClassName } from "@/shared/ui/ui-base";

export const Modal = DialogPrimitive.Root;
export const ModalTrigger = DialogPrimitive.Trigger;
export const ModalPortal = DialogPrimitive.Portal;
export const ModalClose = DialogPrimitive.Close;

export function ModalOverlay({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof DialogPrimitive.Overlay>) {
  return (
    <DialogPrimitive.Overlay
      className={cn(
        "fixed inset-0 z-50 bg-black/58 backdrop-blur-[3px]",
        className,
      )}
      {...props}
    />
  );
}

export function ModalContent({
  children,
  className,
  showClose = true,
  ...props
}: ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & {
  showClose?: boolean;
}) {
  return (
    <ModalPortal>
      <ModalOverlay />
      <DialogPrimitive.Content
        className={cn(
          overlayPanelClassName,
          "fixed top-1/2 left-1/2 z-50 w-[min(92vw,640px)] -translate-x-1/2 -translate-y-1/2 p-6 outline-none",
          className,
        )}
        {...props}
      >
        {children}
        {showClose ? (
          <DialogPrimitive.Close asChild>
            <Button
              aria-label="Close dialog"
              className="border-border/80 text-fg-secondary absolute top-4 right-4 h-9 bg-transparent shadow-none hover:bg-white/[0.05]"
              iconOnly
              leadingIcon={<X className="size-4" />}
              ripple={false}
              tone="secondary"
            />
          </DialogPrimitive.Close>
        ) : null}
      </DialogPrimitive.Content>
    </ModalPortal>
  );
}

export function ModalHeader({
  className,
  ...props
}: ComponentPropsWithoutRef<"div">) {
  return <div className={cn("space-y-1.5 pr-10", className)} {...props} />;
}

export function ModalTitle({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      className={cn(
        "text-fg-primary text-lg font-semibold tracking-[-0.03em]",
        className,
      )}
      {...props}
    />
  );
}

export function ModalDescription({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      className={cn("text-fg-secondary text-sm leading-6", className)}
      {...props}
    />
  );
}

export function ModalBody({
  className,
  ...props
}: ComponentPropsWithoutRef<"div">) {
  return <div className={cn("mt-5", className)} {...props} />;
}

export function ModalFooter({
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

export function ModalShell({
  children,
  description,
  title,
}: {
  children?: ReactNode;
  description?: ReactNode;
  title: ReactNode;
}) {
  return (
    <>
      <ModalHeader>
        <ModalTitle>{title}</ModalTitle>
        {description ? (
          <ModalDescription>{description}</ModalDescription>
        ) : null}
      </ModalHeader>
      {children ? <ModalBody>{children}</ModalBody> : null}
    </>
  );
}

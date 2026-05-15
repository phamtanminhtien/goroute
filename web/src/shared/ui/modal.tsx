import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { motion, useReducedMotion } from "motion/react";
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
  const prefersReducedMotion = useReducedMotion();

  return (
    <DialogPrimitive.Overlay asChild {...props}>
      <motion.div
        animate={{ opacity: 1 }}
        className={cn(
          "fixed inset-0 z-50 bg-black/58 backdrop-blur-[3px]",
          className,
        )}
        initial={{ opacity: 0 }}
        transition={{
          duration: prefersReducedMotion ? 0.12 : 0.18,
          ease: "easeOut",
        }}
      />
    </DialogPrimitive.Overlay>
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
  const prefersReducedMotion = useReducedMotion();

  return (
    <ModalPortal>
      <ModalOverlay />
      <DialogPrimitive.Content asChild {...props}>
        <motion.div
          animate={{
            opacity: 1,
            scale: 1,
            x: "-50%",
            y: "-50%",
          }}
          className={cn(
            overlayPanelClassName,
            "fixed top-1/2 left-1/2 z-50 w-[min(92vw,640px)] p-6 outline-none",
            className,
          )}
          initial={
            prefersReducedMotion
              ? { opacity: 0, scale: 1, x: "-50%", y: "-50%" }
              : { opacity: 0, scale: 0.97, x: "-50%", y: "calc(-50% + 14px)" }
          }
          transition={
            prefersReducedMotion
              ? { duration: 0.14, ease: "easeOut" }
              : {
                  damping: 28,
                  mass: 0.96,
                  stiffness: 360,
                  type: "spring",
                }
          }
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
        </motion.div>
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

export function ModalPanel({
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

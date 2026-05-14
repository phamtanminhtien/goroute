import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import {
  type ButtonHTMLAttributes,
  type PointerEvent,
  type ReactNode,
  useState,
} from "react";

import { cn } from "@/shared/lib/cn";
import { createVariant } from "@/shared/lib/create-variant";

type ButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> & {
  children?: ReactNode;
  iconOnly?: boolean;
  leadingIcon?: ReactNode;
  ripple?: boolean;
  size?: "md" | "lg";
  tone?: "primary" | "secondary" | "ghost";
};

type RippleState = {
  id: number;
  size: number;
  x: number;
  y: number;
};

const buttonVariants = createVariant(
  "relative isolate inline-flex shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-[var(--radius-control)] border text-sm font-semibold whitespace-nowrap select-none disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-none focus-visible:ring-4 [--tw-ring-color:var(--focus-ring)] transition-[background-color,border-color,color,box-shadow] duration-150 ease-[cubic-bezier(0.4,0,0.2,1)]",
  {
    defaultVariants: {
      iconOnly: "false",
      size: "md",
      tone: "primary",
    },
    variants: {
      iconOnly: {
        false:
          "gap-2 px-[var(--control-padding-x)] py-[var(--control-padding-y)]",
        true: "px-0",
      },
      size: {
        lg: "min-h-[var(--control-height-lg)]",
        md: "min-h-[var(--control-height)]",
      },
      tone: {
        ghost:
          "border-transparent bg-transparent text-fg-secondary hover:bg-bg-secondary hover:text-fg-primary",
        primary:
          "border-primary/15 bg-primary text-[var(--primary-foreground)] shadow-[var(--shadow-button)] hover:bg-[var(--primary-hover)]",
        secondary:
          "border-border bg-bg-secondary text-fg-primary shadow-[var(--shadow-sm)] hover:border-border-hover hover:bg-bg-tertiary",
      },
    },
  },
);

export function Button({
  children,
  className,
  disabled,
  iconOnly = false,
  leadingIcon,
  onPointerDown,
  ripple = true,
  size = "md",
  tone = "primary",
  type = "button",
  ...props
}: ButtonProps) {
  const prefersReducedMotion = useReducedMotion();
  const [ripples, setRipples] = useState<RippleState[]>([]);

  function handlePointerDown(event: PointerEvent<HTMLButtonElement>) {
    onPointerDown?.(event);

    if (event.defaultPrevented || disabled || prefersReducedMotion || !ripple) {
      return;
    }

    const bounds = event.currentTarget.getBoundingClientRect();
    const nextSize = Math.max(bounds.width, bounds.height) * 1.8;

    const nextRipple = {
      id: Date.now() + Math.random(),
      size: nextSize,
      x: event.clientX - bounds.left - nextSize / 2,
      y: event.clientY - bounds.top - nextSize / 2,
    };

    setRipples((currentRipples) => [...currentRipples.slice(-2), nextRipple]);
  }

  return (
    <button
      className={buttonVariants(
        { iconOnly: iconOnly ? "true" : "false", size, tone },
        className,
      )}
      disabled={disabled}
      onPointerDown={handlePointerDown}
      type={type}
      {...props}
    >
      <AnimatePresence>
        {ripples.map((rippleItem) => (
          <motion.span
            animate={{ opacity: 0, scale: 2.35 }}
            aria-hidden="true"
            className={
              tone === "primary"
                ? "pointer-events-none absolute rounded-full bg-white/32"
                : "bg-primary/14 pointer-events-none absolute rounded-full"
            }
            exit={{ opacity: 0 }}
            initial={{ opacity: 0.32, scale: 0 }}
            key={rippleItem.id}
            onAnimationComplete={() => {
              setRipples((currentRipples) =>
                currentRipples.filter(
                  (currentRipple) => currentRipple.id !== rippleItem.id,
                ),
              );
            }}
            style={{
              height: rippleItem.size,
              left: rippleItem.x,
              top: rippleItem.y,
              width: rippleItem.size,
            }}
            transition={{ duration: 0.5, ease: "easeOut" }}
          />
        ))}
      </AnimatePresence>
      <span
        className={cn(
          "relative z-10 inline-flex items-center justify-center",
          iconOnly
            ? size === "lg"
              ? "size-[var(--control-height-lg)]"
              : "size-[var(--control-height)]"
            : "gap-2",
        )}
      >
        {leadingIcon ? <span className="shrink-0">{leadingIcon}</span> : null}
        {children}
      </span>
    </button>
  );
}

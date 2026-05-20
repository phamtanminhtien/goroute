import { Check, Clipboard } from "lucide-react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import {
  type ButtonHTMLAttributes,
  type MouseEvent,
  type PointerEvent,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";

import { cn } from "@/shared/lib/cn";
import { createVariant } from "@/shared/lib/create-variant";

export type ButtonProps = Omit<
  ButtonHTMLAttributes<HTMLButtonElement>,
  "children"
> & {
  children?: ReactNode;
  iconOnly?: boolean;
  leadingIcon?: ReactNode;
  ripple?: boolean;
  size?: "sm" | "md" | "lg";
  tone?: "primary" | "secondary" | "ghost";
};

export type CopyButtonProps = Omit<ButtonProps, "children"> & {
  copiedIcon?: ReactNode;
  copiedLabel?: ReactNode;
  copyIcon?: ReactNode;
  copyLabel?: ReactNode;
  copyValue: string;
  onCopied?: () => void;
  onCopyError?: (error: unknown) => void;
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
      size: {
        lg: "min-h-[var(--control-height-lg)] px-[var(--control-padding-x-lg)] py-[var(--control-padding-y)] text-base",
        md: "min-h-[var(--control-height)] px-[var(--control-padding-x)] py-[var(--control-padding-y)] text-sm",
        sm: "min-h-9 px-3 py-2 text-sm",
      },
      iconOnly: {
        false: "gap-2",
        true: "px-0 py-0",
      },
      tone: {
        ghost:
          "border-transparent bg-transparent text-fg-secondary hover:bg-white/[0.04] hover:text-fg-primary",
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
              : size === "sm"
                ? "size-9"
                : "size-[var(--control-height)]"
            : "gap-2",
        )}
      >
        {leadingIcon ? (
          <span className="inline-flex shrink-0 items-center justify-center leading-none">
            {leadingIcon}
          </span>
        ) : null}
        {children}
      </span>
    </button>
  );
}

export function CopyButton({
  copiedIcon = <Check className="size-4" />,
  copiedLabel,
  copyIcon = <Clipboard className="size-4" />,
  copyLabel = "Copy",
  copyValue,
  disabled,
  iconOnly = false,
  leadingIcon,
  onClick,
  onCopied,
  onCopyError,
  type = "button",
  ...props
}: CopyButtonProps) {
  const [copiedValue, setCopiedValue] = useState<string | null>(null);
  const copiedTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isDisabled = disabled || !copyValue;
  const copied = copiedValue === copyValue;
  const displayedIcon = copied ? copiedIcon : (leadingIcon ?? copyIcon);
  const animatedIcon = (
    <AnimatePresence initial={false} mode="wait">
      <motion.span
        animate={{ opacity: 1, rotate: 0, scale: 1 }}
        className="inline-flex items-center justify-center leading-none"
        exit={{ opacity: 0, rotate: copied ? -12 : 12, scale: 0.75 }}
        initial={{ opacity: 0, rotate: copied ? 12 : -12, scale: 0.75 }}
        key={copied ? "copied" : "copy"}
        transition={{ duration: 0.16, ease: "easeOut" }}
      >
        {displayedIcon}
      </motion.span>
    </AnimatePresence>
  );

  useEffect(() => {
    return () => {
      if (copiedTimeoutRef.current) {
        clearTimeout(copiedTimeoutRef.current);
      }
    };
  }, []);

  async function handleClick(event: MouseEvent<HTMLButtonElement>) {
    onClick?.(event);

    if (event.defaultPrevented || isDisabled) {
      return;
    }

    try {
      if (!navigator?.clipboard?.writeText) {
        throw new Error("Clipboard is not available.");
      }

      await navigator.clipboard.writeText(copyValue);
      onCopied?.();
      setCopiedValue(copyValue);

      if (copiedTimeoutRef.current) {
        clearTimeout(copiedTimeoutRef.current);
      }

      copiedTimeoutRef.current = setTimeout(() => {
        setCopiedValue(null);
        copiedTimeoutRef.current = null;
      }, 1500);
    } catch (error) {
      onCopyError?.(error);
    }
  }

  return (
    <Button
      disabled={isDisabled}
      iconOnly={iconOnly}
      leadingIcon={animatedIcon}
      onClick={handleClick}
      type={type}
      {...props}
    >
      {iconOnly ? null : copied && copiedLabel ? copiedLabel : copyLabel}
    </Button>
  );
}

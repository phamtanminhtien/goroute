import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ChevronDown } from "lucide-react";
import type { ComponentPropsWithoutRef } from "react";

import { cn } from "@/shared/lib/cn";

export const Accordion = AccordionPrimitive.Root;

export const AccordionItem = ({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AccordionPrimitive.Item>) => (
  <AccordionPrimitive.Item
    className={cn(
      "border-border/85 bg-bg-primary/72 rounded-[20px] border",
      className,
    )}
    {...props}
  />
);

export function AccordionTrigger({
  children,
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AccordionPrimitive.Trigger>) {
  return (
    <AccordionPrimitive.Header>
      <AccordionPrimitive.Trigger
        className={cn(
          "text-fg-primary flex w-full items-center justify-between gap-3 px-4 py-4 text-left text-sm font-semibold",
          className,
        )}
        {...props}
      >
        <span>{children}</span>
        <ChevronDown className="text-fg-muted size-4 shrink-0 transition-transform duration-150 group-data-[state=open]:rotate-180" />
      </AccordionPrimitive.Trigger>
    </AccordionPrimitive.Header>
  );
}

export function AccordionContent({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof AccordionPrimitive.Content>) {
  return (
    <AccordionPrimitive.Content
      className={cn("text-fg-secondary px-4 pb-4 text-sm leading-6", className)}
      {...props}
    />
  );
}

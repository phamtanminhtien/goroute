import type { TextareaHTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";
import { controlBaseClassName } from "@/shared/ui/ui-base";

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

export function Textarea({ className, ...props }: TextareaProps) {
  return (
    <textarea
      className={cn(
        controlBaseClassName,
        "min-h-[120px] resize-y py-3 leading-6",
        className,
      )}
      {...props}
    />
  );
}

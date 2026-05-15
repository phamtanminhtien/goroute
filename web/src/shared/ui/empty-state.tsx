import type { HTMLAttributes, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";
import { mutedPanelClassName } from "@/shared/ui/ui-base";

export function EmptyState({
  action,
  body,
  className,
  title,
  ...props
}: HTMLAttributes<HTMLDivElement> & {
  action?: ReactNode;
  body: ReactNode;
  title: ReactNode;
}) {
  return (
    <div
      className={cn(mutedPanelClassName, "space-y-2 px-4 py-4", className)}
      {...props}
    >
      <h3 className="text-fg-primary text-sm font-semibold">{title}</h3>
      <p className="text-fg-secondary text-sm leading-6">{body}</p>
      {action ? <div className="pt-1">{action}</div> : null}
    </div>
  );
}

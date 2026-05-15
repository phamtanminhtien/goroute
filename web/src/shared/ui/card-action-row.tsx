import type { HTMLAttributes, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";
import { mutedPanelClassName } from "@/shared/ui/ui-base";

export function CardActionRow({
  actions,
  className,
  description,
  title,
  ...props
}: HTMLAttributes<HTMLDivElement> & {
  actions?: ReactNode;
  description?: ReactNode;
  title: ReactNode;
}) {
  return (
    <div
      className={cn(
        mutedPanelClassName,
        "flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-start sm:justify-between",
        className,
      )}
      {...props}
    >
      <div className="space-y-1">
        <h3 className="text-fg-primary text-sm font-semibold">{title}</h3>
        {description ? (
          <div className="text-fg-secondary text-sm leading-6">
            {description}
          </div>
        ) : null}
      </div>
      {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
    </div>
  );
}

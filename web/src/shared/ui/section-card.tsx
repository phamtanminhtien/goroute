import type { PropsWithChildren, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";
import { SurfaceCard } from "@/shared/ui/surface-card";

type SectionCardProps = PropsWithChildren<{
  description?: ReactNode;
  title: ReactNode;
  className?: string;
  headerAction?: ReactNode;
  tone?: "glass" | "solid";
}>;

export function SectionCard({
  children,
  className,
  description,
  headerAction,
  title,
  tone = "glass",
}: SectionCardProps) {
  return (
    <SurfaceCard className={cn(className)} padding="md" tone={tone}>
      <div className="space-y-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-1.5">
            <h2 className="text-fg-primary text-lg font-semibold">{title}</h2>
            {description ? (
              <p className="text-fg-secondary text-sm leading-6">
                {description}
              </p>
            ) : null}
          </div>
          {headerAction ? <div className="shrink-0">{headerAction}</div> : null}
        </div>
        {children}
      </div>
    </SurfaceCard>
  );
}

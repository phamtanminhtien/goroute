import { AlertCircle, CheckCircle2, Info, TriangleAlert } from "lucide-react";
import type { HTMLAttributes, ReactNode } from "react";

import { cn } from "@/shared/lib/cn";

type InlineAlertTone = "error" | "info" | "success" | "warning";

const toneMap: Record<
  InlineAlertTone,
  {
    className: string;
    icon: ReactNode;
    role: "alert" | "status";
  }
> = {
  error: {
    className:
      "border-rose-500/25 bg-rose-500/10 text-rose-700 dark:text-rose-300",
    icon: <AlertCircle className="mt-0.5 size-4 shrink-0" />,
    role: "alert",
  },
  info: {
    className: "border-primary/18 bg-primary/8 text-primary",
    icon: <Info className="mt-0.5 size-4 shrink-0" />,
    role: "status",
  },
  success: {
    className:
      "border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
    icon: <CheckCircle2 className="mt-0.5 size-4 shrink-0" />,
    role: "status",
  },
  warning: {
    className:
      "border-amber-500/25 bg-amber-500/10 text-amber-700 dark:text-amber-300",
    icon: <TriangleAlert className="mt-0.5 size-4 shrink-0" />,
    role: "alert",
  },
};

export function InlineAlert({
  children,
  className,
  tone = "info",
  ...props
}: HTMLAttributes<HTMLDivElement> & {
  tone?: InlineAlertTone;
}) {
  const config = toneMap[tone];

  return (
    <div
      className={cn(
        "flex gap-3 rounded-[20px] border px-4 py-3 text-sm",
        config.className,
        className,
      )}
      role={config.role}
      {...props}
    >
      {config.icon}
      <div className="min-w-0 flex-1 leading-6">{children}</div>
    </div>
  );
}

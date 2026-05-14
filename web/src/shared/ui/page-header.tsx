import type { PropsWithChildren, ReactNode } from "react";

type PageHeaderProps = PropsWithChildren<{
  eyebrow?: ReactNode;
  description?: ReactNode;
  title: ReactNode;
}>;

export function PageHeader({
  children,
  eyebrow,
  description,
  title,
}: PageHeaderProps) {
  return (
    <header className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div className="space-y-2">
        {eyebrow ? (
          <p className="text-fg-muted text-xs font-semibold tracking-[0.24em] uppercase">
            {eyebrow}
          </p>
        ) : null}
        <div className="space-y-2">
          <h1 className="text-fg-primary text-3xl font-semibold tracking-[-0.03em] sm:text-4xl">
            {title}
          </h1>
          {description ? (
            <p className="text-fg-secondary max-w-3xl text-sm leading-6">
              {description}
            </p>
          ) : null}
        </div>
      </div>
      {children ? (
        <div className="flex items-center gap-3">{children}</div>
      ) : null}
    </header>
  );
}

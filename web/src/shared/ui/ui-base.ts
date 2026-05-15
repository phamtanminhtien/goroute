export const controlBaseClassName =
  "min-h-[var(--control-height)] rounded-[var(--radius-control)] border px-[var(--control-padding-x)] py-[var(--control-padding-y)] text-sm text-fg-primary placeholder:text-fg-muted disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-none focus-visible:ring-4 [--tw-ring-color:var(--focus-ring)] [background:var(--field-bg)] [border-color:var(--field-border)] transition-[border-color,background-color,box-shadow,color] duration-150 ease-[cubic-bezier(0.4,0,0.2,1)] focus:[background:var(--field-bg-focus)] focus:[border-color:var(--field-border-focus)] focus:[box-shadow:0_0_0_4px_color-mix(in_srgb,var(--primary)_10%,transparent)]";

export const overlayPanelClassName =
  "rounded-[var(--radius-surface)] border border-border/90 bg-glass-bg shadow-[var(--glass-shadow)] backdrop-blur-[var(--glass-blur)]";

export const overlayAnimationClassName =
  "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0 duration-150";

export const mutedPanelClassName =
  "rounded-[20px] border border-border/85 bg-bg-primary/72";

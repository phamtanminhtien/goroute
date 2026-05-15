import { Check, ChevronDown, Search } from "lucide-react";
import type { ReactNode } from "react";
import {
  type ChangeEvent,
  type KeyboardEvent,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";

import { cn } from "@/shared/lib/cn";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/ui/popover";
import { controlBaseClassName } from "@/shared/ui/ui-base";

export type ComboboxOption = {
  disabled?: boolean;
  icon?: ReactNode;
  keywords?: string[];
  label: string;
  value: string;
};

type ComboboxProps = {
  className?: string;
  emptyText?: string;
  onValueChange?: (value: string) => void;
  options: ComboboxOption[];
  placeholder?: string;
  searchPlaceholder?: string;
  value?: string;
};

export function Combobox({
  className,
  emptyText = "No matching options.",
  onValueChange,
  options,
  placeholder = "Select an option",
  searchPlaceholder = "Search options",
  value,
}: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const listboxID = useId();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const selectedOption = options.find((option) => option.value === value);

  const filteredOptions = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    if (!normalizedQuery) {
      return options;
    }

    return options.filter((option) => {
      const haystack = [option.label, option.value, ...(option.keywords ?? [])]
        .join(" ")
        .toLowerCase();

      return haystack.includes(normalizedQuery);
    });
  }, [options, query]);

  function handleSelect(nextValue: string) {
    onValueChange?.(nextValue);
    setOpen(false);
  }

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (nextOpen) {
      setQuery("");
      queueMicrotask(() => inputRef.current?.focus());
    }
  }

  function handleKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (
      event.key === "ArrowDown" ||
      event.key === "Enter" ||
      event.key === " "
    ) {
      event.preventDefault();
      handleOpenChange(true);
    }
  }

  return (
    <Popover onOpenChange={handleOpenChange} open={open}>
      <PopoverTrigger asChild>
        <button
          aria-controls={listboxID}
          aria-expanded={open}
          className={cn(
            controlBaseClassName,
            "inline-flex w-full items-center justify-between gap-3 text-left",
            className,
          )}
          onKeyDown={handleKeyDown}
          type="button"
        >
          <span className={cn(!selectedOption && "text-fg-muted")}>
            {selectedOption?.label ?? placeholder}
          </span>
          <ChevronDown className="text-fg-muted size-4" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="space-y-2 p-2" sideOffset={10}>
        <div className="relative">
          <Search className="text-fg-muted pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2" />
          <input
            aria-label={searchPlaceholder}
            className={cn(controlBaseClassName, "min-h-10 pl-9")}
            onChange={(event: ChangeEvent<HTMLInputElement>) =>
              setQuery(event.target.value)
            }
            placeholder={searchPlaceholder}
            ref={inputRef}
            value={query}
          />
        </div>
        <div
          className="max-h-[240px] space-y-1 overflow-y-auto"
          id={listboxID}
          role="listbox"
        >
          {filteredOptions.length === 0 ? (
            <p className="text-fg-muted px-3 py-2 text-sm">{emptyText}</p>
          ) : (
            filteredOptions.map((option) => {
              const selected = option.value === value;

              return (
                <button
                  aria-selected={selected}
                  className="text-fg-secondary hover:text-fg-primary flex min-h-10 w-full items-center justify-between gap-3 rounded-[14px] px-3 py-2 text-left text-sm transition-colors hover:bg-white/[0.06] disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={option.disabled}
                  key={option.value}
                  onClick={() => handleSelect(option.value)}
                  role="option"
                  type="button"
                >
                  <span className="flex items-center gap-2">
                    {option.icon ? (
                      <span className="shrink-0">{option.icon}</span>
                    ) : null}
                    <span>{option.label}</span>
                  </span>
                  {selected ? <Check className="text-primary size-4" /> : null}
                </button>
              );
            })
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

import { Check, ChevronDown, Search, X } from "lucide-react";
import {
  type ChangeEvent,
  type KeyboardEvent,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";

import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/ui/button";
import type { ComboboxOption } from "@/shared/ui/combobox";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/ui/popover";
import { controlBaseClassName } from "@/shared/ui/ui-base";

type MultiSelectProps = {
  className?: string;
  defaultValue?: string[];
  emptyText?: string;
  onValueChange?: (value: string[]) => void;
  options: ComboboxOption[];
  placeholder?: string;
  searchPlaceholder?: string;
  value?: string[];
};

export function MultiSelect({
  className,
  defaultValue,
  emptyText = "No matching options.",
  onValueChange,
  options,
  placeholder = "Select one or more options",
  searchPlaceholder = "Search options",
  value,
}: MultiSelectProps) {
  const [uncontrolledValue, setUncontrolledValue] = useState(
    defaultValue ?? [],
  );
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const currentValue = value ?? uncontrolledValue;
  const listboxID = useId();
  const inputRef = useRef<HTMLInputElement | null>(null);

  const selectedOptions = options.filter((option) =>
    currentValue.includes(option.value),
  );

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

  function setNextValue(nextValue: string[]) {
    if (value === undefined) {
      setUncontrolledValue(nextValue);
    }
    onValueChange?.(nextValue);
  }

  function toggleOption(optionValue: string) {
    setNextValue(
      currentValue.includes(optionValue)
        ? currentValue.filter((item) => item !== optionValue)
        : [...currentValue, optionValue],
    );
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
            "flex w-full items-center justify-between gap-3 text-left",
            className,
          )}
          onKeyDown={handleKeyDown}
          type="button"
        >
          <span className="flex min-w-0 flex-1 flex-wrap gap-2">
            {selectedOptions.length === 0 ? (
              <span className="text-fg-muted">{placeholder}</span>
            ) : (
              selectedOptions.map((option, index) => {
                if (index >= 2) {
                  return null;
                }

                return (
                  <span
                    className="border-primary/18 bg-primary/10 text-primary inline-flex max-w-full items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-semibold"
                    key={option.value}
                  >
                    <span className="truncate">{option.label}</span>
                  </span>
                );
              })
            )}
            {selectedOptions.length > 2 ? (
              <span className="border-border/80 text-fg-secondary inline-flex items-center rounded-full border bg-white/[0.04] px-2.5 py-1 text-xs font-semibold">
                +{selectedOptions.length - 2} more
              </span>
            ) : null}
          </span>
          <ChevronDown className="text-fg-muted size-4 shrink-0" />
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
        {selectedOptions.length > 0 ? (
          <div className="flex flex-wrap gap-2 px-1">
            {selectedOptions.map((option) => (
              <span
                className="border-primary/18 bg-primary/10 text-primary inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-semibold"
                key={option.value}
              >
                <span>{option.label}</span>
                <Button
                  aria-label={`Remove ${option.label}`}
                  className="text-primary h-auto min-h-0 border-none bg-transparent px-0 py-0 shadow-none hover:bg-transparent"
                  iconOnly
                  leadingIcon={<X className="size-3" />}
                  onClick={(event) => {
                    event.stopPropagation();
                    toggleOption(option.value);
                  }}
                  ripple={false}
                  tone="ghost"
                />
              </span>
            ))}
          </div>
        ) : null}
        <div
          className="max-h-[240px] space-y-1 overflow-y-auto"
          id={listboxID}
          role="listbox"
        >
          {filteredOptions.length === 0 ? (
            <p className="text-fg-muted px-3 py-2 text-sm">{emptyText}</p>
          ) : (
            filteredOptions.map((option) => {
              const selected = currentValue.includes(option.value);

              return (
                <button
                  aria-selected={selected}
                  className="text-fg-secondary hover:text-fg-primary flex min-h-10 w-full items-center justify-between gap-3 rounded-[14px] px-3 py-2 text-left text-sm transition-colors hover:bg-white/[0.06] disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={option.disabled}
                  key={option.value}
                  onClick={() => toggleOption(option.value)}
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

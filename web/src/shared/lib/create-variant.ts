import { cn } from "@/shared/lib/cn";

type VariantGroups = Record<string, Record<string, string>>;
type VariantSelection<TVariants extends VariantGroups> = {
  [TKey in keyof TVariants]?: keyof TVariants[TKey];
};

type VariantConfig<TVariants extends VariantGroups> = {
  defaultVariants?: VariantSelection<TVariants>;
  variants: TVariants;
};

export function createVariant<TVariants extends VariantGroups>(
  base: string,
  config: VariantConfig<TVariants>,
) {
  return (selection?: VariantSelection<TVariants>, className?: string) => {
    const resolvedSelection = {
      ...config.defaultVariants,
      ...selection,
    } as VariantSelection<TVariants>;

    const variantClassNames = Object.entries(config.variants).map(
      ([groupName, groupValues]) => {
        const resolvedValue = resolvedSelection[
          groupName as keyof TVariants
        ] as keyof typeof groupValues | undefined;

        if (!resolvedValue) {
          return "";
        }

        return groupValues[resolvedValue];
      },
    );

    return cn(base, variantClassNames, className);
  };
}

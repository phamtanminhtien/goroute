import { codexProviderConnectionFormEntry } from "@/features/providers/registry/cx";
import { defaultProviderConnectionFormEntry } from "@/features/providers/registry/default";
import { openAIProviderConnectionFormEntry } from "@/features/providers/registry/openai";
import type { ProviderConnectionFormRegistryEntry } from "@/features/providers/registry/types";

const providerConnectionFormRegistry: Record<
  string,
  ProviderConnectionFormRegistryEntry
> = {
  cx: codexProviderConnectionFormEntry,
  openai: openAIProviderConnectionFormEntry,
};

export { defaultProviderConnectionFormEntry } from "@/features/providers/registry/default";
export type {
  ConnectionFormDialogMeta,
  ConnectionFormFeedback,
  ConnectionFormValues,
  CreateConnectionFormRenderProps,
  EditConnectionFormRenderProps,
  ProviderConnectionFormRegistryEntry,
} from "@/features/providers/registry/types";
export { emptyConnectionFormValues } from "@/features/providers/registry/types";

export function getProviderConnectionFormEntry(providerID: string) {
  return (
    providerConnectionFormRegistry[providerID] ??
    defaultProviderConnectionFormEntry
  );
}

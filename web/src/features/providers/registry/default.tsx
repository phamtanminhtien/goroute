import {
  buildDefaultCreateMeta,
  buildDefaultEditMeta,
  GenericCreateConnectionForm,
  GenericEditConnectionForm,
} from "@/features/providers/registry/shared";
import type { ProviderConnectionFormRegistryEntry } from "@/features/providers/registry/types";

export const defaultProviderConnectionFormEntry: ProviderConnectionFormRegistryEntry =
  {
    getCreateMeta: buildDefaultCreateMeta,
    getEditMeta: buildDefaultEditMeta,
    renderCreate: (props) => <GenericCreateConnectionForm {...props} />,
    renderEdit: (props) => <GenericEditConnectionForm {...props} />,
  };

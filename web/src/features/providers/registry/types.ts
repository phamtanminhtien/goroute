import type { ReactNode } from "react";

import type {
  ProviderConnection,
  ProviderItem,
} from "@/features/providers/api";

export type ConnectionFormFeedback = {
  text: string;
  tone: "error" | "success";
} | null;

export type ConnectionFormValues = {
  accessToken: string;
  apiKey: string;
  id: string;
  name: string;
  refreshToken: string;
};

export const emptyConnectionFormValues: ConnectionFormValues = {
  accessToken: "",
  apiKey: "",
  id: "",
  name: "",
  refreshToken: "",
};

export type ConnectionFormDialogMeta = {
  description: string;
  title: string;
};

type BaseConnectionFormRenderProps = {
  busy: boolean;
  feedback: ConnectionFormFeedback;
  onCancel: () => void;
  onSubmit: (values: ConnectionFormValues) => void | Promise<void>;
  provider: ProviderItem;
};

export type CreateConnectionFormRenderProps = BaseConnectionFormRenderProps;

export type EditConnectionFormRenderProps = BaseConnectionFormRenderProps & {
  connection: ProviderConnection;
  initialValues: ConnectionFormValues;
};

export type ProviderConnectionFormRegistryEntry = {
  getCreateMeta?: (provider: ProviderItem) => ConnectionFormDialogMeta;
  getEditMeta?: (
    provider: ProviderItem,
    connection: ProviderConnection,
  ) => ConnectionFormDialogMeta;
  renderCreate: (props: CreateConnectionFormRenderProps) => ReactNode;
  renderEdit: (props: EditConnectionFormRenderProps) => ReactNode;
};

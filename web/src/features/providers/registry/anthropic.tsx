import { useState } from "react";

import {
  buildDefaultCreateMeta,
  buildDefaultEditMeta,
  ConnectionFormActions,
  FormFeedback,
} from "@/features/providers/registry/shared";
import type {
  CreateConnectionFormRenderProps,
  EditConnectionFormRenderProps,
  ProviderConnectionFormRegistryEntry,
} from "@/features/providers/registry/types";
import { emptyConnectionFormValues } from "@/features/providers/registry/types";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";

function AnthropicCreateConnectionForm({
  busy,
  feedback,
  onCancel,
  onSubmit,
}: CreateConnectionFormRenderProps) {
  const [values, setValues] = useState(emptyConnectionFormValues);
  const [formError, setFormError] = useState<string | null>(null);

  return (
    <form
      className="grid gap-4"
      onSubmit={async (event) => {
        event.preventDefault();

        if (!values.id.trim() || !values.name.trim()) {
          setFormError("Connection ID and display name are required.");
          return;
        }

        if (!values.apiKey.trim()) {
          setFormError("API key is required for Anthropic connections.");
          return;
        }

        setFormError(null);
        await onSubmit(values);
      }}
    >
      <div className="grid gap-4 md:grid-cols-2">
        <Field label="Connection ID" required>
          <Input
            onChange={(event) =>
              setValues((current) => ({ ...current, id: event.target.value }))
            }
            placeholder="anthropic-1"
            value={values.id}
          />
        </Field>
        <Field label="Display name" required>
          <Input
            onChange={(event) =>
              setValues((current) => ({ ...current, name: event.target.value }))
            }
            placeholder="Anthropic API key"
            value={values.name}
          />
        </Field>
      </div>

      <Field help="Primary credential for Anthropic requests." label="API key">
        <Input
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              apiKey: event.target.value,
            }))
          }
          placeholder="Enter a new API key"
          type="password"
          value={values.apiKey}
        />
      </Field>

      <FormFeedback feedback={feedback} formError={formError} />
      <ConnectionFormActions
        busy={busy}
        onCancel={onCancel}
        submitLabel="Create connection"
      />
    </form>
  );
}

function AnthropicEditConnectionForm({
  busy,
  feedback,
  initialValues,
  onCancel,
  onSubmit,
}: EditConnectionFormRenderProps) {
  const [values, setValues] = useState(initialValues);
  const [formError, setFormError] = useState<string | null>(null);

  return (
    <form
      className="grid gap-4"
      onSubmit={async (event) => {
        event.preventDefault();

        if (!values.id.trim() || !values.name.trim()) {
          setFormError("Connection ID and display name are required.");
          return;
        }

        setFormError(null);
        await onSubmit(values);
      }}
    >
      <div className="grid gap-4 md:grid-cols-2">
        <Field label="Connection ID" required>
          <Input
            onChange={(event) =>
              setValues((current) => ({ ...current, id: event.target.value }))
            }
            placeholder="anthropic-1"
            value={values.id}
          />
        </Field>
        <Field label="Display name" required>
          <Input
            onChange={(event) =>
              setValues((current) => ({ ...current, name: event.target.value }))
            }
            placeholder="Anthropic API key"
            value={values.name}
          />
        </Field>
      </div>

      <Field
        help="Leave blank unless you want to replace the current API key."
        label="API key"
      >
        <Input
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              apiKey: event.target.value,
            }))
          }
          placeholder="Enter a new API key"
          type="password"
          value={values.apiKey}
        />
      </Field>

      <FormFeedback feedback={feedback} formError={formError} />
      <ConnectionFormActions
        busy={busy}
        onCancel={onCancel}
        submitLabel="Save changes"
      />
    </form>
  );
}

export const anthropicProviderConnectionFormEntry: ProviderConnectionFormRegistryEntry =
  {
    getCreateMeta: buildDefaultCreateMeta,
    getEditMeta: buildDefaultEditMeta,
    renderCreate: (props) => <AnthropicCreateConnectionForm {...props} />,
    renderEdit: (props) => <AnthropicEditConnectionForm {...props} />,
  };

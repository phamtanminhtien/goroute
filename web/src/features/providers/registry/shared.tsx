import { useState } from "react";

import type {
  ProviderConnection,
  ProviderItem,
} from "@/features/providers/api";
import type {
  ConnectionFormDialogMeta,
  ConnectionFormFeedback,
  CreateConnectionFormRenderProps,
  EditConnectionFormRenderProps,
} from "@/features/providers/registry/types";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { Input } from "@/shared/ui/input";

type ConnectionFormActionsProps = {
  busy: boolean;
  onCancel: () => void;
  submitLabel: string;
};

export function ConnectionFormActions({
  busy,
  onCancel,
  submitLabel,
}: ConnectionFormActionsProps) {
  return (
    <div className="flex flex-wrap gap-2">
      <Button disabled={busy} type="submit">
        {busy ? "Saving..." : submitLabel}
      </Button>
      <Button disabled={busy} onClick={onCancel} tone="secondary" type="button">
        Cancel
      </Button>
    </div>
  );
}

export function FormFeedback({
  feedback,
  formError,
}: {
  feedback: ConnectionFormFeedback;
  formError: string | null;
}) {
  return (
    <>
      {feedback?.tone === "error" ? (
        <InlineAlert tone="error">{feedback.text}</InlineAlert>
      ) : null}

      {formError ? (
        <p className="text-sm leading-6 text-rose-700" role="alert">
          {formError}
        </p>
      ) : null}
    </>
  );
}

export function GenericCreateConnectionForm({
  busy,
  feedback,
  onCancel,
  onSubmit,
}: CreateConnectionFormRenderProps) {
  const [values, setValues] = useState({
    accessToken: "",
    apiKey: "",
    id: "",
    name: "",
    refreshToken: "",
  });
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
            placeholder="connection-1"
            value={values.id}
          />
        </Field>
        <Field label="Display name" required>
          <Input
            onChange={(event) =>
              setValues((current) => ({ ...current, name: event.target.value }))
            }
            placeholder="user@example.com"
            value={values.name}
          />
        </Field>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Field label="Access token">
          <Input
            onChange={(event) =>
              setValues((current) => ({
                ...current,
                accessToken: event.target.value,
              }))
            }
            placeholder="Enter a new access token"
            type="password"
            value={values.accessToken}
          />
        </Field>
        <Field label="API key">
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
      </div>

      <Field label="Refresh token">
        <Input
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              refreshToken: event.target.value,
            }))
          }
          placeholder="Enter a new refresh token"
          type="password"
          value={values.refreshToken}
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

export function GenericEditConnectionForm({
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
            placeholder="connection-1"
            value={values.id}
          />
        </Field>
        <Field label="Display name" required>
          <Input
            onChange={(event) =>
              setValues((current) => ({ ...current, name: event.target.value }))
            }
            placeholder="user@example.com"
            value={values.name}
          />
        </Field>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Field label="Access token">
          <Input
            onChange={(event) =>
              setValues((current) => ({
                ...current,
                accessToken: event.target.value,
              }))
            }
            placeholder="Enter a new access token"
            type="password"
            value={values.accessToken}
          />
        </Field>
        <Field label="API key">
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
      </div>

      <Field label="Refresh token">
        <Input
          onChange={(event) =>
            setValues((current) => ({
              ...current,
              refreshToken: event.target.value,
            }))
          }
          placeholder="Enter a new refresh token"
          type="password"
          value={values.refreshToken}
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

export function buildDefaultCreateMeta(
  provider: ProviderItem,
): ConnectionFormDialogMeta {
  return {
    description: `Add a new ${provider.name} connection. Credentials are stored securely, and secrets will remain redacted after save.`,
    title: `Add ${provider.name} connection`,
  };
}

export function buildDefaultEditMeta(
  provider: ProviderItem,
  connection: ProviderConnection,
): ConnectionFormDialogMeta {
  return {
    description:
      "Secrets stay blank on edit. Enter only the values you want to replace.",
    title: `Edit ${connection.name} for ${provider.name}`,
  };
}

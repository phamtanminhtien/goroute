import { Copy } from "lucide-react";
import { type ReactNode, useState } from "react";

import type {
  ProviderConnection,
  ProviderItem,
} from "@/features/providers/api";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { Input } from "@/shared/ui/input";
import { Spinner } from "@/shared/ui/spinner";

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

const codexAuthorizationURL = "https://auth.openai.com/oauth/authorize";

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

type ConnectionFormActionsProps = {
  busy: boolean;
  onCancel: () => void;
  submitLabel: string;
};

function ConnectionFormActions({
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

function FormFeedback({
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

function GenericCreateConnectionForm({
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

function GenericEditConnectionForm({
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

function CodexCreateConnectionForm({
  busy,
  feedback,
  onCancel,
  onSubmit,
}: CreateConnectionFormRenderProps) {
  const [callbackURL, setCallbackURL] = useState("");
  const [copyFeedback, setCopyFeedback] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);

  async function handleCopyAuthorizationURL() {
    try {
      if (!navigator?.clipboard?.writeText) {
        setCopyFeedback("Copy is not available in this browser.");
        return;
      }

      await navigator.clipboard.writeText(codexAuthorizationURL);
      setCopyFeedback("Authorization URL copied.");
    } catch {
      setCopyFeedback("Copy failed. Please copy the URL manually.");
    }
  }

  return (
    <form
      className="grid gap-4"
      onSubmit={async (event) => {
        event.preventDefault();

        if (!callbackURL.trim()) {
          setFormError("Callback URL is required.");
          return;
        }

        const derivedValues = buildCodexValuesFromCallbackURL(callbackURL);
        if (!derivedValues) {
          setFormError("Callback URL is invalid.");
          return;
        }

        setFormError(null);
        await onSubmit(derivedValues);
      }}
    >
      <div className="border-border/80 bg-bg-secondary/55 flex items-center gap-3 rounded-[20px] border px-4 py-4">
        <span className="text-[#ff7a4d]">
          <Spinner className="size-7 border-[3px]" />
        </span>
        <div className="min-w-0">
          <p className="text-fg-primary text-sm font-semibold tracking-[-0.03em]">
            Waiting for popup authorization...
          </p>
          <p className="text-fg-muted mt-1 text-xs leading-5">
            If the browser popup does not finish, continue with the manual
            authorization flow below.
          </p>
        </div>
      </div>

      <div className="space-y-4">
        <div className="space-y-1">
          <p className="text-fg-primary text-sm font-semibold tracking-[-0.03em]">
            Step 1: Open this URL in your browser
          </p>
          <p className="text-fg-muted text-xs leading-5">
            Start the Codex OAuth flow, then come back and continue with your
            connection details.
          </p>
        </div>

        <div className="flex flex-col gap-3 md:flex-row">
          <Input
            aria-label="Codex authorization URL"
            readOnly
            value={codexAuthorizationURL}
            disabled
          />

          <Button
            leadingIcon={<Copy className="size-5" />}
            onClick={handleCopyAuthorizationURL}
            tone="secondary"
            type="button"
          >
            Copy
          </Button>
        </div>

        {copyFeedback ? (
          <p className="text-fg-muted text-sm leading-6" role="status">
            {copyFeedback}
          </p>
        ) : null}
      </div>

      <div className="space-y-4">
        <div className="space-y-1">
          <p className="text-fg-primary text-sm font-semibold tracking-[-0.03em]">
            Step 2: Paste the callback URL here
          </p>
          <p className="text-fg-muted text-xs leading-5">
            After authorization, copy the full callback URL from your browser.
          </p>
        </div>

        <Input
          aria-label="Codex callback URL"
          onChange={(event) => setCallbackURL(event.target.value)}
          placeholder="http://localhost:20128/callback?code=..."
          value={callbackURL}
        />
      </div>

      <FormFeedback feedback={feedback} formError={formError} />
      <ConnectionFormActions
        busy={busy}
        onCancel={onCancel}
        submitLabel="Create connection"
      />
    </form>
  );
}

function CodexEditConnectionForm({
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
            placeholder="codex-1"
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

      <Field
        help="Leave blank unless you want to replace the current access token."
        label="Access token"
      >
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

      <Field
        help="Leave blank unless you want to replace the current refresh token."
        label="Refresh token"
      >
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

function OpenAICreateConnectionForm({
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
          setFormError("API key is required for OpenAI connections.");
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
            placeholder="openai-1"
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

      <Field help="Primary credential for OpenAI requests." label="API key">
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

function OpenAIEditConnectionForm({
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
            placeholder="openai-1"
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

function buildDefaultCreateMeta(
  provider: ProviderItem,
): ConnectionFormDialogMeta {
  return {
    description: `Add a new ${provider.name} connection. Credentials are stored securely, and secrets will remain redacted after save.`,
    title: `Add ${provider.name} connection`,
  };
}

function buildDefaultEditMeta(
  provider: ProviderItem,
  connection: ProviderConnection,
): ConnectionFormDialogMeta {
  return {
    description:
      "Secrets stay blank on edit. Enter only the values you want to replace.",
    title: `Edit ${connection.name} for ${provider.name}`,
  };
}

export const defaultProviderConnectionFormEntry: ProviderConnectionFormRegistryEntry =
  {
    getCreateMeta: buildDefaultCreateMeta,
    getEditMeta: buildDefaultEditMeta,
    renderCreate: (props) => <GenericCreateConnectionForm {...props} />,
    renderEdit: (props) => <GenericEditConnectionForm {...props} />,
  };

const providerConnectionFormRegistry: Record<
  string,
  ProviderConnectionFormRegistryEntry
> = {
  cx: {
    getCreateMeta: buildDefaultCreateMeta,
    getEditMeta: buildDefaultEditMeta,
    renderCreate: (props) => <CodexCreateConnectionForm {...props} />,
    renderEdit: (props) => <CodexEditConnectionForm {...props} />,
  },
  openai: {
    getCreateMeta: buildDefaultCreateMeta,
    getEditMeta: buildDefaultEditMeta,
    renderCreate: (props) => <OpenAICreateConnectionForm {...props} />,
    renderEdit: (props) => <OpenAIEditConnectionForm {...props} />,
  },
};

export function getProviderConnectionFormEntry(providerID: string) {
  return (
    providerConnectionFormRegistry[providerID] ??
    defaultProviderConnectionFormEntry
  );
}

function buildCodexValuesFromCallbackURL(
  callbackURL: string,
): ConnectionFormValues | null {
  try {
    const parsedURL = new URL(callbackURL);
    const code =
      parsedURL.searchParams.get("code")?.trim() ||
      parsedURL.searchParams.get("state")?.trim() ||
      parsedURL.pathname.split("/").filter(Boolean).at(-1) ||
      "connection";
    const normalizedCode = code.toLowerCase().replace(/[^a-z0-9]+/g, "-");
    const suffix = normalizedCode.replace(/^-+|-+$/g, "") || "connection";

    return {
      ...emptyConnectionFormValues,
      id: `codex-${suffix}`,
      name: `Codex ${suffix}`,
    };
  } catch {
    return null;
  }
}

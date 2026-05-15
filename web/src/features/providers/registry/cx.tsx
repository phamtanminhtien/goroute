import { Copy } from "lucide-react";
import { useState } from "react";

import {
  buildDefaultCreateMeta,
  buildDefaultEditMeta,
  ConnectionFormActions,
  FormFeedback,
} from "@/features/providers/registry/shared";
import type {
  ConnectionFormValues,
  CreateConnectionFormRenderProps,
  EditConnectionFormRenderProps,
  ProviderConnectionFormRegistryEntry,
} from "@/features/providers/registry/types";
import { emptyConnectionFormValues } from "@/features/providers/registry/types";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Spinner } from "@/shared/ui/spinner";

const codexAuthorizationURL = "https://auth.openai.com/oauth/authorize";

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

export const codexProviderConnectionFormEntry: ProviderConnectionFormRegistryEntry =
  {
    getCreateMeta: buildDefaultCreateMeta,
    getEditMeta: buildDefaultEditMeta,
    renderCreate: (props) => <CodexCreateConnectionForm {...props} />,
    renderEdit: (props) => <CodexEditConnectionForm {...props} />,
  };

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

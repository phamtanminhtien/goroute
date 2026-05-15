import { Copy } from "lucide-react";
import { useEffect, useState } from "react";

import { generateProviderOAuthURL } from "@/features/providers/api";
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
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Spinner } from "@/shared/ui/spinner";

function CodexCreateConnectionForm({
  busy,
  feedback,
  onCancel,
  onSubmit,
}: CreateConnectionFormRenderProps) {
  const [authorizationURL, setAuthorizationURL] = useState("");
  const [oauthSessionID, setOAuthSessionID] = useState("");
  const [authorizationError, setAuthorizationError] = useState<string | null>(
    null,
  );
  const [authorizationLoading, setAuthorizationLoading] = useState(true);
  const [callbackURL, setCallbackURL] = useState("");
  const [copyFeedback, setCopyFeedback] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadAuthorizationURL() {
      setAuthorizationLoading(true);
      setAuthorizationError(null);

      try {
        const nextAuthorization = await generateProviderOAuthURL("cx");
        if (cancelled) {
          return;
        }

        setAuthorizationURL(nextAuthorization.url);
        setOAuthSessionID(nextAuthorization.sessionID);
      } catch {
        if (cancelled) {
          return;
        }

        setAuthorizationURL("");
        setOAuthSessionID("");
        setAuthorizationError(
          "Could not generate the Codex authorization URL right now.",
        );
      } finally {
        if (!cancelled) {
          setAuthorizationLoading(false);
        }
      }
    }

    void loadAuthorizationURL();

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleCopyAuthorizationURL() {
    try {
      if (!authorizationURL) {
        setCopyFeedback("Authorization URL is not ready yet.");
        return;
      }

      if (!navigator?.clipboard?.writeText) {
        setCopyFeedback("Copy is not available in this browser.");
        return;
      }

      await navigator.clipboard.writeText(authorizationURL);
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

        if (!authorizationURL) {
          setFormError("Authorization URL is not ready yet.");
          return;
        }

        if (!callbackURL.trim()) {
          setFormError("Callback URL is required.");
          return;
        }

        if (!oauthSessionID) {
          setFormError("OAuth session is not ready yet.");
          return;
        }

        setFormError(null);
        await onSubmit({
          ...emptyConnectionFormValues,
          callbackURL: callbackURL.trim(),
          oauthSessionID,
        });
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
            value={
              authorizationLoading
                ? "Generating authorization URL..."
                : authorizationURL
            }
            disabled
          />

          <Button
            leadingIcon={<Copy className="size-5" />}
            onClick={handleCopyAuthorizationURL}
            tone="secondary"
            type="button"
            disabled={authorizationLoading || !authorizationURL}
          >
            Copy
          </Button>
        </div>

        {authorizationError ? (
          <p className="text-danger-600 text-sm leading-6" role="alert">
            {authorizationError}
          </p>
        ) : null}

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
        busy={busy || authorizationLoading}
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

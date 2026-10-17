import { useCallback, useEffect, useRef, useState } from "react";
import type { GeminiKeyStatus, GeminiTestResponse } from "./types";

type Operation = "idle" | "loading" | "saving" | "testing" | "removing";

type Feedback = { tone: "success" | "error" | "info"; text: string } | null;

const SETTINGS_URL = "/api/settings/gemini";

// The key is only ever held in component state while being typed and is
// cleared after every operation. It is never persisted in the browser.
async function readError(response: Response): Promise<string> {
  try {
    const body = (await response.json()) as { error?: string };
    if (body?.error) return body.error;
  } catch {
    // Fall through to generic message.
  }
  return `Request failed (${response.status}).`;
}

function sourceLabel(status: GeminiKeyStatus | null): string {
  if (!status) return "Unknown";
  switch (status.source) {
    case "user":
      return "Your key (Settings)";
    case "environment":
      return "Server environment (GEMINI_API_KEY)";
    default:
      return "None";
  }
}

export function SettingsPanel({
  open,
  onClose,
  onStatusChange,
}: {
  open: boolean;
  onClose: () => void;
  onStatusChange?: (status: GeminiKeyStatus | null) => void;
}) {
  const [status, setStatus] = useState<GeminiKeyStatus | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [showKey, setShowKey] = useState(false);
  const [operation, setOperation] = useState<Operation>("idle");
  const [feedback, setFeedback] = useState<Feedback>(null);
  const [connection, setConnection] = useState<"untested" | "ok" | "failed">(
    "untested",
  );
  const inputRef = useRef<HTMLInputElement>(null);

  const busy = operation !== "idle";

  const applyStatus = useCallback(
    (next: GeminiKeyStatus | null) => {
      setStatus(next);
      onStatusChange?.(next);
    },
    [onStatusChange],
  );

  const refreshStatus = useCallback(async () => {
    setOperation("loading");
    try {
      const response = await fetch(SETTINGS_URL, { cache: "no-store" });
      if (!response.ok) throw new Error(await readError(response));
      applyStatus((await response.json()) as GeminiKeyStatus);
    } catch (err: unknown) {
      applyStatus(null);
      setFeedback({
        tone: "error",
        text:
          err instanceof Error && err.message !== "Failed to fetch"
            ? err.message
            : "Could not reach the SignalSleuth backend on port 8090.",
      });
    } finally {
      setOperation("idle");
    }
  }, [applyStatus]);

  // Fetch status (never the secret) whenever the panel opens.
  useEffect(() => {
    if (!open) {
      // Clear any typed key when the panel closes.
      setApiKey("");
      setShowKey(false);
      return;
    }
    setFeedback(null);
    void refreshStatus();
    const timer = window.setTimeout(() => inputRef.current?.focus(), 150);
    return () => window.clearTimeout(timer);
  }, [open, refreshStatus]);

  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !busy) onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, busy, onClose]);

  const trimmed = apiKey.trim();
  const keyInvalid = apiKey.length > 0 && trimmed.length === 0;

  async function saveKey() {
    if (busy) return;
    if (!trimmed) {
      setFeedback({ tone: "error", text: "Enter a Gemini API key first." });
      return;
    }
    if (/\s/.test(trimmed)) {
      setFeedback({
        tone: "error",
        text: "The API key must not contain spaces.",
      });
      return;
    }

    setOperation("saving");
    setFeedback(null);
    try {
      const response = await fetch(SETTINGS_URL, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ api_key: trimmed }),
      });
      if (!response.ok) throw new Error(await readError(response));
      applyStatus((await response.json()) as GeminiKeyStatus);
      setConnection("untested");
      setFeedback({
        tone: "success",
        text: "Key saved. New investigations will use it immediately.",
      });
    } catch (err: unknown) {
      setFeedback({
        tone: "error",
        text:
          err instanceof Error && err.message !== "Failed to fetch"
            ? err.message
            : "Network error while saving the key.",
      });
    } finally {
      // Never keep the key around after submission.
      setApiKey("");
      setShowKey(false);
      setOperation("idle");
    }
  }

  async function testConnection() {
    if (busy) return;
    setOperation("testing");
    setFeedback(null);
    try {
      const response = await fetch(`${SETTINGS_URL}/test`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) throw new Error(await readError(response));
      const result = (await response.json()) as GeminiTestResponse;
      applyStatus({
        configured: result.configured,
        source: result.source,
        env_fallback_available: result.env_fallback_available,
        user_key_configured: result.user_key_configured,
      });
      setConnection(result.ok ? "ok" : "failed");
      setFeedback({
        tone: result.ok ? "success" : "error",
        text: result.message,
      });
    } catch (err: unknown) {
      setConnection("failed");
      setFeedback({
        tone: "error",
        text:
          err instanceof Error && err.message !== "Failed to fetch"
            ? err.message
            : "Network error while testing the connection.",
      });
    } finally {
      setOperation("idle");
    }
  }

  async function removeKey() {
    if (busy) return;
    setOperation("removing");
    setFeedback(null);
    try {
      const response = await fetch(SETTINGS_URL, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) throw new Error(await readError(response));
      const next = (await response.json()) as GeminiKeyStatus;
      applyStatus(next);
      setConnection("untested");
      setFeedback({
        tone: "info",
        text: next.configured
          ? "Your key was removed. Falling back to the server environment key."
          : "Your key was removed. Gemini is now not configured.",
      });
    } catch (err: unknown) {
      setFeedback({
        tone: "error",
        text:
          err instanceof Error && err.message !== "Failed to fetch"
            ? err.message
            : "Network error while removing the key.",
      });
    } finally {
      setOperation("idle");
    }
  }

  let pillTone = "neutral";
  let pillText = "Not configured";
  if (operation === "loading") {
    pillText = "Checking…";
  } else if (operation === "saving") {
    pillText = "Saving…";
  } else if (operation === "testing") {
    pillText = "Testing connection…";
  } else if (operation === "removing") {
    pillText = "Removing key…";
  } else if (status?.configured) {
    if (connection === "ok") {
      pillTone = "ok";
      pillText = "Connection successful";
    } else if (connection === "failed") {
      pillTone = "bad";
      pillText = "Connection failed";
    } else {
      pillTone = "configured";
      pillText = "Configured";
    }
  } else if (status) {
    pillTone = "warn";
  }

  return (
    <div
      className={`settings-overlay ${open ? "open" : ""}`}
      aria-hidden={!open}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !busy) onClose();
      }}
    >
      <aside
        className="settings-drawer"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
      >
        <div className="settings-header">
          <div>
            <div className="eyebrow">CONFIGURATION</div>
            <h2 id="settings-title">Settings</h2>
          </div>
          <button
            id="settings-close"
            className="icon-button"
            onClick={onClose}
            disabled={busy}
            aria-label="Close settings"
          >
            ✕
          </button>
        </div>

        <section className="settings-card">
          <div className="settings-card-top">
            <div className="settings-card-title">
              <span className="settings-icon">✦</span>
              <div>
                <h3>Gemini API</h3>
                <p className="muted">
                  Used only when no verified knowledge article matches an
                  incident.
                </p>
              </div>
            </div>
            <span className={`key-status ${pillTone}`} id="gemini-key-status">
              {busy && <span className="status-spinner" />}
              {pillText}
            </span>
          </div>

          <dl className="settings-meta">
            <div>
              <dt>Active source</dt>
              <dd>{sourceLabel(status)}</dd>
            </div>
            <div>
              <dt>Environment fallback</dt>
              <dd>
                {status
                  ? status.env_fallback_available
                    ? "Available"
                    : "Not set"
                  : "—"}
              </dd>
            </div>
          </dl>

          <form
            className="key-form"
            onSubmit={(event) => {
              event.preventDefault();
              void saveKey();
            }}
            autoComplete="off"
          >
            <label className="field key-field" htmlFor="gemini-api-key">
              <span>
                {status?.user_key_configured
                  ? "Replace your API key"
                  : "Gemini API key"}
              </span>
              <div className={`key-input-wrap ${keyInvalid ? "invalid" : ""}`}>
                <input
                  ref={inputRef}
                  id="gemini-api-key"
                  name="gemini-api-key"
                  type={showKey ? "text" : "password"}
                  value={apiKey}
                  onChange={(event) => setApiKey(event.target.value)}
                  placeholder={
                    status?.user_key_configured
                      ? "•••••••••••• (saved — enter a new key to replace)"
                      : "Paste your Gemini API key"
                  }
                  autoComplete="off"
                  autoCorrect="off"
                  autoCapitalize="off"
                  spellCheck={false}
                  data-lpignore="true"
                  disabled={busy}
                />
                <button
                  type="button"
                  id="gemini-key-toggle"
                  className="key-toggle"
                  onClick={() => setShowKey((value) => !value)}
                  disabled={busy || apiKey.length === 0}
                  aria-label={showKey ? "Hide API key" : "Show API key"}
                  aria-pressed={showKey}
                >
                  {showKey ? "Hide" : "Show"}
                </button>
              </div>
              {keyInvalid && (
                <span className="field-error">Key cannot be blank.</span>
              )}
            </label>

            <div className="settings-actions">
              <button
                type="submit"
                id="gemini-key-save"
                className="button button-primary"
                disabled={busy || !trimmed}
              >
                {operation === "saving" ? (
                  <>
                    <span className="spinner" /> Saving…
                  </>
                ) : (
                  "Save key"
                )}
              </button>
              <button
                type="button"
                id="gemini-key-test"
                className="button button-secondary"
                onClick={() => void testConnection()}
                disabled={busy || !status?.configured}
              >
                {operation === "testing" ? "Testing…" : "Test connection"}
              </button>
              <button
                type="button"
                id="gemini-key-remove"
                className="button button-danger"
                onClick={() => void removeKey()}
                disabled={busy || !status?.user_key_configured}
              >
                {operation === "removing" ? "Removing…" : "Remove key"}
              </button>
            </div>
          </form>

          {feedback && (
            <div
              className={`settings-feedback ${feedback.tone}`}
              role={feedback.tone === "error" ? "alert" : "status"}
            >
              {feedback.text}
            </div>
          )}

          <ul className="settings-notes">
            <li>
              Your key is kept only in the backend's memory. It is never
              stored in this browser or written to disk, and is not shown
              again after saving.
            </li>
            <li>
              Restarting the backend clears it — re-enter it afterwards, or
              set <code>GEMINI_API_KEY</code> as a fallback.
            </li>
            <li>
              The key is shared by everyone using this backend. Settings
              accept changes from this machine only.
            </li>
          </ul>
        </section>
      </aside>
    </div>
  );
}

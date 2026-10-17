
import { useState } from "react";
import type {
  FailureScenario,
  InvestigationResponse,
  KnowledgeArticle,
} from "./types";
import "./style.css";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

const scenarios: { value: FailureScenario; label: string }[] = [
  { value: "normal", label: "Normal" },
  { value: "db_failure", label: "Database failure" },
  { value: "redis_failure", label: "Redis failure" },
  { value: "payment_failure", label: "Payment failure" },
  { value: "error_spike", label: "HTTP error spike" },
  { value: "slow", label: "High latency" },
];

function readable(value: string): string {
  return value
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "—";

  if (typeof value === "number") {
    return Number.isFinite(value)
      ? value.toLocaleString("en-US", { maximumFractionDigits: 4 })
      : String(value);
  }

  if (typeof value === "object") {
    return JSON.stringify(value, null, 2);
  }

  return String(value);
}

function formatPercent(value: unknown): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "—";
  }

  // Assumes values between 0 and 1 are ratios.
  const percent = value >= 0 && value <= 1 ? value * 100 : value;
  return `${percent.toFixed(2)}%`;
}

function App() {
  const [scenario, setScenario] =
    useState<FailureScenario>("db_failure");
  const [result, setResult] =
    useState<InvestigationResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [scenarioLoading, setScenarioLoading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [service, setService] = useState("orderflow");
const [duration, setDuration] = useState("10m");



  async function triggerScenario(nextScenario: FailureScenario) {
    setScenarioLoading(true);
    setError("");
    setNotice("");

    try {
      const response = await fetch("/api/failure-scenario", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ scenario: nextScenario }),
      });

      if (!response.ok) {
        throw new Error(
          `Could not set scenario (${response.status}). Check orderflow on port 8080.`,
        );
      }

      setScenario(nextScenario);
      setNotice(`Scenario set to ${readable(nextScenario)}.`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Scenario request failed.");
    } finally {
      setScenarioLoading(false);
    }
  }

  
async function investigate() {
  setLoading(true);
  setError("");
  setNotice("");

  try {
    const response = await fetch("/api/investigations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        service,
        duration,
      }),
    });

    if (!response.ok) {
      const message = await response.text();
      throw new Error(
        message || `Investigation failed (${response.status}).`,
      );
    }

    const data = (await response.json()) as InvestigationResponse;
    setResult(data);
    setNotice(
      `Investigation completed for ${service} over ${duration}.`,
    );
  } catch (err: unknown) {
    setError(
      err instanceof Error
        ? err.message
        : "Investigation request failed.",
    );
  } finally {
    setLoading(false);
  }
}


  const evidence = result?.evidence;
  const articles: KnowledgeArticle[] = result?.knowledge ?? [];
  const status = evidence?.status ?? "Not investigated";
  const severity = evidence?.severity ?? "—";
  const isHealthy = status.toLowerCase() === "healthy";
  const isNormal = severity.toLowerCase() === "normal";

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-icon">S</div>
          <div>
            <div className="brand-name">SignalSleuth</div>
            <div className="brand-subtitle">AI OBSERVABILITY</div>
          </div>
        </div>

        <div className="nav-label">WORKSPACE</div>
        <a className="nav-item active" href="#overview">
          <span>◈</span> Incident overview
        </a>
        <a className="nav-item" href="#evidence">
          <span>⌕</span> Evidence explorer
        </a>
        <a className="nav-item" href="#knowledge">
          <span>▤</span> Knowledge Hub
        </a>

        <div className="sidebar-bottom">
          <div className="connection-dot" />
          <div>
            <strong>Local environment</strong>
            <span>Orderflow · Prometheus · Loki</span>
          </div>
        </div>
      </aside>

      <main className="main-content" id="overview">
        <header className="topbar">
          <div>
            <div className="eyebrow">OBSERVABILITY / INVESTIGATIONS</div>
            <h1>Incident Command Center</h1>
            <p className="muted">
              Investigate service health, correlate evidence, and reuse
              incident knowledge.
            </p>
          </div>
          <div className="environment-pill">
            <span className="connection-dot" /> LOCAL DEMO
          </div>
        </header>

        <section className="control-panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">CONTROL ROOM</div>
              <h2>Run an investigation</h2>
            </div>
            <span className="small-tag">
  SERVICE: {service.toUpperCase()}
</span>
          </div>

          
<div className="controls">
  <label className="field">
    <span>Service</span>
    <select
      value={service}
      onChange={(event) => setService(event.target.value)}
    >
      <option value="orderflow">Orderflow</option>
    </select>
  </label>

  <label className="field">
    <span>Investigation timeline</span>
    <select
      value={duration}
      onChange={(event) => setDuration(event.target.value)}
    >
      <option value="5m">Last 5 minutes</option>
      <option value="10m">Last 10 minutes</option>
      <option value="15m">Last 15 minutes</option>
      <option value="1h">Last 1 hour</option>
    </select>
  </label>

  <label className="field">
    <span>Failure scenario</span>
    <select
      value={scenario}
      onChange={(event) =>
        setScenario(event.target.value as FailureScenario)
      }
    >
      {scenarios.map((item) => (
        <option key={item.value} value={item.value}>
          {item.label}
        </option>
      ))}
    </select>
  </label>

  <button
    className="button button-secondary"
    disabled={scenarioLoading}
    onClick={() => void triggerScenario(scenario)}
  >
    {scenarioLoading ? "Applying..." : "Apply scenario"}
  </button>

  <button
    className="button button-primary"
    disabled={loading}
    onClick={() => void investigate()}
  >
    {loading ? (
      <>
        <span className="spinner" /> Investigating...
      </>
    ) : (
      <>✦ Investigate incident</>
    )}
  </button>
</div>


          <p className="helper-text">
            Apply a scenario first, then investigate to collect fresh evidence.
          </p>

          {notice && <div className="notice">{notice}</div>}
          {error && <div className="error-banner">{error}</div>}
        </section>

        <section className="metric-grid">
          <MetricCard
            label="SERVICE STATUS"
            value={readable(status)}
            detail="From latest investigation"
            tone={isHealthy ? "green" : evidence ? "amber" : "neutral"}
            icon="◉"
          />
          <MetricCard
            label="SEVERITY"
            value={readable(severity)}
            detail="Evidence-based assessment"
            tone={isNormal ? "green" : evidence ? "red" : "neutral"}
            icon="⚑"
          />
          <MetricCard
            label="INVESTIGATION RUNTIME"
            value={
              result
                ? `${result.duration_ms.toLocaleString()} ms`
                : "—"
            }
            detail="API-reported duration"
            tone="blue"
            icon="◷"
          />
          <MetricCard
            label="KNOWLEDGE MATCHES"
            value={String(articles.length)}
            detail="Runbooks and saved articles"
            tone="purple"
            icon="▤"
          />
        </section>

        <section className="content-grid">
          <div className="panel explanation-panel">
            <div className="panel-heading">
              <div>
                <div className="eyebrow">AI-ASSISTED ANALYSIS</div>
                <h2>Investigation summary</h2>
              </div>
              <span className="ai-badge">✦ AI</span>
            </div>

            {result ? (
              <>
                <div className="ai-markdown">
  <ReactMarkdown remarkPlugins={[remarkGfm]}>
    {result.ai_explanation}
  </ReactMarkdown>
</div>

                <div className="summary-footer">
  <span>Service: {result.service}</span>
  <span>Lookback: {result.duration}</span>
</div>
              </>
            ) : (
              <EmptyState
                title="No investigation yet"
                description="Run an investigation to see the AI explanation and collected evidence."
              />
            )}
          </div>

          <div className="panel impact-panel">
            <div className="panel-heading">
              <div>
                <div className="eyebrow">SERVICE IMPACT</div>
                <h2>Impact overview</h2>
              </div>
              <span className="panel-icon">⌁</span>
            </div>

            {evidence ? (
              <>
                <div className="impact-number">
  {formatPercent(evidence.impact?.http_error_rate)}
</div>
                <div className="muted">HTTP error rate</div>
                <div className="impact-description">
                  {evidence.impact?.description || "No impact description returned."}
                </div>
                <div className="affected-list">
                  <strong>Affected paths</strong>
                  {(evidence.impact?.affected_paths ?? []).length ? (
                    evidence.impact.affected_paths.map((path) => (
                      <div className="path-chip" key={path}>
                        <code>{path}</code>
                      </div>
                    ))
                  ) : (
                    <span className="muted">No affected paths detected.</span>
                  )}
                </div>
              </>
            ) : (
              <EmptyState
                title="Impact not available"
                description="Impact details appear after the first investigation."
              />
            )}
          </div>
        </section>

        <section className="panel evidence-panel" id="evidence">
          <div className="panel-heading">
            <div>
              <div className="eyebrow">OBSERVABILITY SIGNALS</div>
              <h2>Evidence explorer</h2>
            </div>
            <span className="small-tag">
              {evidence?.evidence?.length ?? 0} SIGNALS
            </span>
          </div>

          {!evidence ? (
            <EmptyState
              title="Waiting for evidence"
              description="Metrics and investigation signals will appear here."
            />
          ) : evidence.evidence.length === 0 ? (
            <EmptyState
              title="No evidence signals returned"
              description="The investigation completed without structured evidence signals."
            />
          ) : (
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Signal</th>
                    <th>Source</th>
                    <th>Observed value</th>
                  </tr>
                </thead>
                <tbody>
                  {evidence.evidence.map((item, index) => (
                    <tr key={`${item.signal}-${index}`}>
                      <td>{readable(item.signal)}</td>
                      <td><span className="source-tag">{item.source}</span></td>
                      <td className="value-cell">
                         {formatValue(item.value)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {evidence && (
            <div className="evidence-meta">
              <span>
                <span className="meta-dot" /> Status: {readable(evidence.status)}
              </span>
              <span>Service: {evidence.service}</span>
            </div>
          )}
        </section>

        <section className="panel knowledge-panel" id="knowledge">
          <div className="panel-heading">
            <div>
              <div className="eyebrow">ORGANIZATIONAL MEMORY</div>
              <h2>Knowledge Hub</h2>
              <p className="muted">
                Runbooks and learned incident patterns for faster future triage.
              </p>
            </div>
            <span className="knowledge-count">{articles.length} ARTICLES</span>
            
          </div>

          {articles.length === 0 ? (
            <EmptyState
              title="No matching knowledge"
              description="Relevant runbooks or saved incident patterns will appear here when the backend returns them."
            />
          ) : (
            <div className="knowledge-grid">
              {articles.map((article, index) => (
                <article
                  className="knowledge-card"
                  key={`${article.title}-${index}`}
                >
                  <div className="knowledge-card-top">
                    <span className="book-icon">▤</span>
                    <span
                      className={`verification ${
                        article.verified ? "verified" : "unverified"
                      }`}
                    >
                      {article.verified ? "Verified" : "Unverified"}
                    </span>
                    <span className={`origin-badge ${article.match_origin ?? "unknown"}`}>
  {{
    runbook: "Predefined runbook",
    learned: "Newly learned",
    saved: "Reused from knowledge.json",
  }[article.match_origin ?? "unknown"] ?? "Origin unknown"}
</span>
                  </div>
                  <h3>{article.title}</h3>
                  <div className="knowledge-summary markdown-content">
  <ReactMarkdown remarkPlugins={[remarkGfm]}>
    {article.summary}
  </ReactMarkdown>
</div>

                  {article.matched_on?.length > 0 && (
                    <div className="match-list">
                      <strong>Matched on</strong>
                      <div className="tag-list">
                        {article.matched_on.map((match) => (
                          <span className="match-tag" key={match}>
                            {match}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  <div className="steps-heading">Suggested steps</div>
                  <ol className="steps-list">
                    {article.steps?.map((step, stepIndex) => (
                      <li key={`${stepIndex}-${step}`}>{step}</li>
                    ))}
                  </ol>

                  {article.source && (
                    <div className="article-source">Source: {article.source}</div>
                  )}
                </article>
              ))}
            </div>
          )}
        </section>

        <footer className="footer">
          <span>SignalSleuth AI · Hackathon prototype</span>
          <span>
            Runtime measures the API investigation, not total human
            investigation time.
          </span>
        </footer>
      </main>
    </div>
  );
}

function MetricCard({
  label,
  value,
  detail,
  tone,
  icon,
}: {
  label: string;
  value: string;
  detail: string;
  tone: "green" | "amber" | "red" | "blue" | "purple" | "neutral";
  icon: string;
}) {
  return (
    <article className="metric-card">
      <div className="metric-top">
        <span className="metric-label">{label}</span>
        <span className={`metric-icon ${tone}`}>{icon}</span>
      </div>
      <div className="metric-value">{value}</div>
      <div className="metric-detail">{detail}</div>
    </article>
  );
}

function EmptyState({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="empty-state">
      <div className="empty-icon">⌁</div>
      <strong>{title}</strong>
      <p>{description}</p>
    </div>
  );
}

export default App;

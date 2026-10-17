import { useState } from "react";
import type { FailureScenario, GeminiKeyStatus } from "./types";

interface CommandBlockProps {
  title: string;
  environment: "GCP VM Terminal (OrderFlow :8080)" | "GCP VM Terminal (Signal Agent :8090)" | "GCP VM Terminal / Any" | "Browser / Dashboard UI";
  description: string;
  command: string;
  expectedOutput: string;
  verificationTip: string;
  troubleshootingTip: string;
}

function CommandBlock({
  title,
  environment,
  description,
  command,
  expectedOutput,
  verificationTip,
  troubleshootingTip,
}: CommandBlockProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    void navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2200);
  };

  return (
    <div className="guide-command-card">
      <div className="guide-command-header">
        <div>
          <h4 className="guide-command-title">{title}</h4>
          <span className="guide-env-tag">{environment}</span>
        </div>
        <button
          type="button"
          className={`guide-copy-btn ${copied ? "copied" : ""}`}
          onClick={handleCopy}
          aria-label="Copy command to clipboard"
        >
          {copied ? (
            <>
              <span className="guide-copy-icon">✓</span> Copied!
            </>
          ) : (
            <>
              <span className="guide-copy-icon">⎘</span> Copy command
            </>
          )}
        </button>
      </div>

      <p className="guide-command-desc">{description}</p>

      <div className="guide-code-wrapper">
        <pre className="guide-code-block">
          <code>{command}</code>
        </pre>
      </div>

      <div className="guide-details-grid">
        <div className="guide-detail-box">
          <strong className="guide-detail-label">Expected Output:</strong>
          <pre className="guide-detail-content output">
            <code>{expectedOutput}</code>
          </pre>
        </div>
        <div className="guide-detail-box">
          <strong className="guide-detail-label">Verification Tip:</strong>
          <p className="guide-detail-content tip">{verificationTip}</p>
        </div>
      </div>

      {troubleshootingTip && (
        <div className="guide-trouble-alert">
          <span className="guide-trouble-icon">ℹ</span>
          <span><strong>Troubleshooting:</strong> {troubleshootingTip}</span>
        </div>
      )}
    </div>
  );
}

export function HowToUsePage({
  geminiStatus,
  onOpenSettings,
  onNavigateToOverview,
  onNavigateToKnowledge,
}: {
  geminiStatus?: GeminiKeyStatus | null;
  onOpenSettings?: () => void;
  onNavigateToOverview?: () => void;
  onNavigateToKnowledge?: () => void;
}) {
  const [selectedScenario, setSelectedScenario] = useState<FailureScenario>("redis_failure");
  const [activeTab, setActiveTab] = useState<"walkthrough" | "troubleshooting">("walkthrough");

  return (
    <div className="how-to-use-page">
      {/* Header Card */}
      <div className="guide-hero-card">
        <div className="guide-hero-content">
          <span className="eyebrow">PRACTICAL INTERACTIVE PLAYBOOK</span>
          <h1 className="guide-hero-title">How to Use SignalSleuth AI</h1>
          <p className="guide-hero-subtitle">
            An end-to-end, single-page interactive guide. All steps are performed directly within the web dashboard,
            with copyable Bash traffic loops provided in Step 3 to feed real-time incident telemetry into Prometheus and Loki.
          </p>

          <div className="guide-env-strip">
            <div className="guide-env-pill">
              <span className="guide-pill-dot green" />
              <span>Public Dashboard: <strong>https://34.69.34.84</strong> (or <strong>http://localhost</strong>)</span>
            </div>
            <div className="guide-env-pill">
              <span className="guide-pill-dot blue" />
              <span>VM Backend Ports: <strong>OrderFlow :8080</strong> · <strong>Signal Agent :8090</strong></span>
            </div>
            <div className="guide-env-pill">
              <span className={`guide-pill-dot ${geminiStatus?.configured ? "green" : "amber"}`} />
              <span>Gemini AI: <strong>{geminiStatus?.configured ? `Configured (${geminiStatus.source})` : "Unconfigured"}</strong></span>
            </div>
          </div>
        </div>

        {/* View Switcher Tabs */}
        <div className="guide-tab-nav">
          <button
            type="button"
            className={`guide-tab-btn ${activeTab === "walkthrough" ? "active" : ""}`}
            onClick={() => setActiveTab("walkthrough")}
          >
            <span>📖</span> Interactive Playbook (All 9 Steps)
          </button>
          <button
            type="button"
            className={`guide-tab-btn ${activeTab === "troubleshooting" ? "active" : ""}`}
            onClick={() => setActiveTab("troubleshooting")}
          >
            <span>🛠</span> Troubleshooting & Diagnostics
          </button>
        </div>
      </div>

      {/* WALKTHROUGH: ALL 9 STEPS ON ONE CONTINUOUS PAGE */}
      {activeTab === "walkthrough" && (
        <div className="guide-all-steps-container">


          {/* =========================================================================
              STEP 1: CONFIGURE GEMINI API KEY
             ========================================================================= */}
          <div id="guide-step-1" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 1</span>
              <h2>Configure Gemini API Key in the UI</h2>
              <p className="muted">
                Signal Agent uses Google Gemini to synthesize root cause diagnoses, impact assessments, and remediation plans when an incident is active.
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>Frontend Setup Steps:</h4>
                  <ol>
                    <li>Click the <strong>⚙ Settings</strong> button in the top-right navigation bar (or use the button below).</li>
                    <li>Paste your Google Gemini API key into the <strong>Gemini API Key</strong> field.</li>
                    <li>Click <strong>Test & Save Connection</strong> to validate the key directly against Google Gemini.</li>
                    <li>Look for the green <strong>Connection Verified</strong> badge and confirmation message.</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-primary"
                    onClick={onOpenSettings}
                  >
                    ⚙ Open Settings Modal Now
                  </button>
                </div>
                <div className="guide-action-col">
                  <div className="guide-info-card">
                    <div className="guide-info-icon">🔑</div>
                    <div className="guide-info-body">
                      <strong>Current Gemini Status:</strong>
                      <p>
                        {geminiStatus?.configured ? (
                          <span className="status-badge status-healthy">
                            ✓ Configured via {geminiStatus.source === "user" ? "Frontend Settings" : "Environment Variable"}
                          </span>
                        ) : (
                          <span className="status-badge status-degraded">
                            ⚠ Not Configured — Open Settings to configure
                          </span>
                        )}
                      </p>
                      <p style={{ marginTop: "10px", fontSize: "12px", color: "var(--muted)" }}>
                        Configured keys are saved securely in <code>gemini_settings.json</code> and persist across server restarts.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 2: ACTIVATE FAILURE SCENARIO
             ========================================================================= */}
          <div id="guide-step-2" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 2</span>
              <h2>Activate a Failure Scenario in the UI</h2>
              <p className="muted">
                OrderFlow contains an in-memory fault injector. Select which failure scenario to activate using the dropdown selector on the Incident Overview dashboard.
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>How to Activate in the UI:</h4>
                  <ol>
                    <li>Go to the <strong>◈ Incident Overview</strong> page.</li>
                    <li>Locate the <strong>Failure Scenario</strong> dropdown in the top bar.</li>
                    <li>Select one of the 5 incident scenarios (e.g. <code>redis_failure</code>, <code>db_failure</code>, or <code>payment_failure</code>).</li>
                    <li>The system immediately begins simulating the chosen failure condition in memory.</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-primary"
                    onClick={onNavigateToOverview}
                  >
                    ◈ Go to Incident Overview Dashboard
                  </button>
                </div>
                <div className="guide-action-col">
                  <h4>Scenario Breakdown:</h4>
                  <div className="guide-scenario-list">
                    <div className="guide-scenario-item">
                      <strong>🔴 redis_failure</strong>
                      <p>Redis cache disconnects. OrderFlow gracefully falls back to Postgres while logging Redis cache degradation.</p>
                    </div>
                    <div className="guide-scenario-item">
                      <strong>🗄️ db_failure</strong>
                      <p>PostgreSQL write transactions fail on order creations, returning HTTP 500 Internal Server Error.</p>
                    </div>
                    <div className="guide-scenario-item">
                      <strong>💳 payment_failure</strong>
                      <p>Upstream payment provider is unavailable, causing payment processing to return HTTP 502 Bad Gateway.</p>
                    </div>
                    <div className="guide-scenario-item">
                      <strong>⏱️ slow</strong>
                      <p>Injects 2000ms latency on order creation; response time elevates P95/P99 latency without 500 errors.</p>
                    </div>
                    <div className="guide-scenario-item">
                      <strong>💥 error_spike</strong>
                      <p>Product catalog throws unexpected exceptions, driving service error rate to 100%.</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 3: GENERATE SCENARIO-SPECIFIC TRAFFIC (THE ONLY BASH STEP)
             ========================================================================= */}
          <div id="guide-step-3" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 3</span>
              <h2>Generate Scenario-Specific Traffic (Bash Commands)</h2>
              <p className="muted">
                Each scenario affects a specific endpoint and dependency. Select your scenario below to get the exact copyable Bash loop,
                then run it in your terminal to feed live telemetry into Prometheus and Loki.
              </p>
            </div>

            <div className="guide-card-content">
              {/* Scenario Selector Tabs */}
              <div className="guide-scenario-nav">
                <button
                  type="button"
                  className={`guide-scenario-btn ${selectedScenario === "redis_failure" ? "active" : ""}`}
                  onClick={() => setSelectedScenario("redis_failure")}
                >
                  🔴 Redis Failure
                </button>
                <button
                  type="button"
                  className={`guide-scenario-btn ${selectedScenario === "db_failure" ? "active" : ""}`}
                  onClick={() => setSelectedScenario("db_failure")}
                >
                  🗄️ DB Failure
                </button>
                <button
                  type="button"
                  className={`guide-scenario-btn ${selectedScenario === "payment_failure" ? "active" : ""}`}
                  onClick={() => setSelectedScenario("payment_failure")}
                >
                  💳 Payment Failure
                </button>
                <button
                  type="button"
                  className={`guide-scenario-btn ${selectedScenario === "slow" ? "active" : ""}`}
                  onClick={() => setSelectedScenario("slow")}
                >
                  ⏱️ High Latency
                </button>
                <button
                  type="button"
                  className={`guide-scenario-btn ${selectedScenario === "error_spike" ? "active" : ""}`}
                  onClick={() => setSelectedScenario("error_spike")}
                >
                  💥 Error Spike
                </button>
                <button
                  type="button"
                  className={`guide-scenario-btn ${selectedScenario === "normal" ? "active" : ""}`}
                  onClick={() => setSelectedScenario("normal")}
                >
                  🌿 Normal Baseline
                </button>
              </div>

              {/* Scenario Content */}
              {selectedScenario === "redis_failure" && (
                <div>
                  <div className="guide-tag-row">
                    <span className="environment-pill">Active Target: <strong>GET /api/products/1</strong></span>
                    <span className="environment-pill">Target Dependency: <strong>redis</strong></span>
                  </div>
                  <CommandBlock
                    title="Generate Redis Cache Degradation Traffic"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends 10 product requests at 1-second intervals. Redis cache misses trigger internal fallback logs."
                    command={`for i in $(seq 1 10); do
  echo -n "Request $i: "
  curl -sS -o /dev/null -w 'HTTP %{http_code} | Time %{time_total}s\\n' \\
    http://localhost:8080/api/products/1
  sleep 1
done`}
                    expectedOutput={`Request 1: HTTP 200 | Time 0.00512s
Request 2: HTTP 200 | Time 0.00488s
...
Request 10: HTTP 200 | Time 0.00495s`}
                    verificationTip="HTTP status is 200 (graceful fallback), but Loki captures Redis connection errors behind the scenes."
                    troubleshootingTip="Wait 10-15 seconds after running this loop for Prometheus scrape intervals to register the degraded window."
                  />
                </div>
              )}

              {selectedScenario === "db_failure" && (
                <div>
                  <div className="guide-tag-row">
                    <span className="environment-pill">Active Target: <strong>POST /api/orders</strong></span>
                    <span className="environment-pill">Target Dependency: <strong>postgresql</strong></span>
                  </div>
                  <CommandBlock
                    title="Generate Database Failure Traffic"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends 10 order creation requests that fail when attempting database writes."
                    command={`for i in $(seq 1 10); do
  echo -n "Order $i: "
  curl -sS -o /dev/null -w 'HTTP %{http_code} | Time %{time_total}s\\n' \\
    -X POST http://localhost:8080/api/orders \\
    -H 'Content-Type: application/json' \\
    -d '{"customerName":"DemoCustomer","items":[{"productId":1,"quantity":1}]}'
  sleep 1
done`}
                    expectedOutput={`Order 1: HTTP 500 | Time 0.00421s
Order 2: HTTP 500 | Time 0.00398s
...
Order 10: HTTP 500 | Time 0.00410s`}
                    verificationTip="Observe HTTP 500 responses indicating database transaction failure."
                    troubleshootingTip="If you get HTTP 400, verify the JSON syntax and product ID (product ID 1 exists in migrations)."
                  />
                </div>
              )}

              {selectedScenario === "payment_failure" && (
                <div>
                  <div className="guide-tag-row">
                    <span className="environment-pill">Active Target: <strong>POST /api/payment</strong></span>
                    <span className="environment-pill">Target Dependency: <strong>payment_provider</strong></span>
                  </div>
                  <CommandBlock
                    title="Generate Payment Gateway Failure Traffic"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends 10 payment processing requests simulating gateway downtime."
                    command={`for i in $(seq 1 10); do
  echo -n "Payment $i: "
  curl -sS -o /dev/null -w 'HTTP %{http_code} | Time %{time_total}s\\n' \\
    -X POST http://localhost:8080/api/payment \\
    -H 'Content-Type: application/json' \\
    -d "{\\"orderId\\":$i,\\"amount\\":49.99}"
  sleep 1
done`}
                    expectedOutput={`Payment 1: HTTP 502 | Time 0.1042s
Payment 2: HTTP 502 | Time 0.1039s
...
Payment 10: HTTP 502 | Time 0.1041s`}
                    verificationTip="Notice HTTP 502 Bad Gateway and ~100ms response time indicating upstream gateway failure."
                    troubleshootingTip="Make sure orderId is > 0 and amount is > 0 (validation will reject negative numbers with 400)."
                  />
                </div>
              )}

              {selectedScenario === "slow" && (
                <div>
                  <div className="guide-tag-row">
                    <span className="environment-pill">Active Target: <strong>POST /api/orders</strong></span>
                    <span className="environment-pill">Target Metric: <strong>P95 / P99 Latency</strong></span>
                  </div>
                  <CommandBlock
                    title="Generate High Latency Traffic"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends 5 order creation requests that incur a 2.0s sleep."
                    command={`for i in $(seq 1 5); do
  echo -n "Order $i: "
  curl -sS -o /dev/null -w 'HTTP %{http_code} | Time %{time_total}s\\n' \\
    -X POST http://localhost:8080/api/orders \\
    -H 'Content-Type: application/json' \\
    -d '{"customerName":"SlowOrder","items":[{"productId":1,"quantity":1}]}'
  sleep 1
done`}
                    expectedOutput={`Order 1: HTTP 201 | Time 2.0142s
Order 2: HTTP 201 | Time 2.0129s
...
Order 5: HTTP 201 | Time 2.0155s`}
                    verificationTip="Observe HTTP 201 Created but total elapsed time > 2.0 seconds."
                    troubleshootingTip="Keep request count to 5 so testing remains swift."
                  />
                </div>
              )}

              {selectedScenario === "error_spike" && (
                <div>
                  <div className="guide-tag-row">
                    <span className="environment-pill">Active Target: <strong>GET /api/products</strong></span>
                    <span className="environment-pill">Target Metric: <strong>HTTP 500 Error Rate</strong></span>
                  </div>
                  <CommandBlock
                    title="Generate HTTP Error Spike Traffic"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends 10 product catalog queries to trigger sudden 500 errors."
                    command={`for i in $(seq 1 10); do
  echo -n "Catalog Request $i: "
  curl -sS -o /dev/null -w 'HTTP %{http_code} | Time %{time_total}s\\n' \\
    http://localhost:8080/api/products
  sleep 1
done`}
                    expectedOutput={`Catalog Request 1: HTTP 500 | Time 0.0031s
Catalog Request 2: HTTP 500 | Time 0.0028s
...
Catalog Request 10: HTTP 500 | Time 0.0029s`}
                    verificationTip="Look for HTTP 500 status on all catalog queries."
                    troubleshootingTip="Check that the failure scenario was set to 'error_spike' in Step 2."
                  />
                </div>
              )}

              {selectedScenario === "normal" && (
                <div>
                  <div className="guide-tag-row">
                    <span className="environment-pill">Active Target: <strong>GET /api/products/1</strong></span>
                    <span className="environment-pill">Target Metric: <strong>Baseline Health</strong></span>
                  </div>
                  <CommandBlock
                    title="Generate Baseline Healthy Traffic"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends 10 fast healthy requests to verify baseline operations."
                    command={`for i in $(seq 1 10); do
  echo -n "Healthy Request $i: "
  curl -sS -o /dev/null -w 'HTTP %{http_code} | Time %{time_total}s\\n' \\
    http://localhost:8080/api/products/1
  sleep 1
done`}
                    expectedOutput={`Healthy Request 1: HTTP 200 | Time 0.0048s
...
Healthy Request 10: HTTP 200 | Time 0.0046s`}
                    verificationTip="All requests should complete with HTTP 200 in <10ms."
                    troubleshootingTip="If errors persist, make sure scenario 'normal' was accepted."
                  />
                </div>
              )}

              {/* Wait Callout */}
              <div className="guide-warning-banner">
                <span className="guide-warn-icon">⏳</span>
                <div>
                  <strong>Crucial Telemetry Timing: Wait 10 to 15 seconds after generating traffic</strong>
                  <p>
                    Prometheus scrapes OrderFlow metrics every 10 seconds, and Grafana Alloy flushes container logs into Loki.
                    Waiting 10–15 seconds ensures full telemetry is available before launching the investigation in Step 4.
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 4: RUN AI INVESTIGATION
             ========================================================================= */}
          <div id="guide-step-4" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 4</span>
              <h2>Run an AI Incident Investigation in the UI</h2>
              <p className="muted">
                Launch an automated investigation. Signal Agent connects to the Observability MCP server to query Prometheus and Loki,
                gathers real-time telemetry, checks knowledge matches, and uses Gemini to synthesize a diagnosis.
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>How to Investigate in the UI:</h4>
                  <ol>
                    <li>Click <strong>◈ Incident overview</strong> in the sidebar.</li>
                    <li>Ensure Service is set to <strong>orderflow</strong> and Duration to <strong>5m</strong>.</li>
                    <li>Click the purple <strong>Investigate Incident</strong> button.</li>
                    <li>Watch the live spinner while evidence is gathered and analyzed by Gemini.</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-primary"
                    onClick={onNavigateToOverview}
                  >
                    ◈ Go to Incident Overview Dashboard
                  </button>
                </div>
                <div className="guide-action-col">
                  <h4>What You Will See in the Result:</h4>
                  <ul className="guide-result-bullets">
                    <li>
                      <strong>Investigation Mode:</strong> <code>Mode: AI</code> (Cold incident — full LLM diagnostic).
                    </li>
                    <li>
                      <strong>Root Cause Diagnosis:</strong> Rich Markdown explanation identifying the exact failure mechanism, affected dependencies, and code files.
                    </li>
                    <li>
                      <strong>Service Status:</strong> Degraded or Critical with HTTP error rate and P95 latency indicators.
                    </li>
                    <li>
                      <strong>Telemetry Signals:</strong> Real logs from Loki and Prometheus metrics queried via MCP.
                    </li>
                  </ul>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 5: AUTOMATED KNOWLEDGE LEARNING
             ========================================================================= */}
          <div id="guide-step-5" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 5</span>
              <h2>Automated Knowledge Capture</h2>
              <p className="muted">
                When Gemini successfully diagnoses an incident, Signal Agent automatically captures the incident signature,
                synthesizes a structured runbook article, and persists it into <code>knowledge.json</code>.
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-info-card">
                <div className="guide-info-icon">🧠</div>
                <div className="guide-info-body">
                  <strong>Automatic Incident Knowledge Learning Rules:</strong>
                  <ul>
                    <li>An article is created ONLY when Gemini successfully diagnoses an active incident (status is not &apos;healthy&apos; or &apos;normal&apos;).</li>
                    <li>Newly learned articles <strong>always default to unverified (verified: false)</strong> to require human-in-the-loop review.</li>
                    <li>The article is assigned a deterministic stable ID (e.g. <code>learned-orderflow-redis</code>).</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 6: INVESTIGATE AGAIN (UNVERIFIED CHECK)
             ========================================================================= */}
          <div id="guide-step-6" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 6</span>
              <h2>Investigate Again: Unverified Safety Gate Enforcement</h2>
              <p className="muted">
                Run an investigation a second time while the learned article remains unverified. This proves that SignalSleuth AI protects against unvetted automation.
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>Run Second Investigation:</h4>
                  <ol>
                    <li>Return to <strong>◈ Incident Overview</strong>.</li>
                    <li>Click the <strong>Investigate Incident</strong> button again.</li>
                    <li>Notice that Gemini runs again and the investigation takes ~2 seconds.</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-secondary"
                    onClick={onNavigateToOverview}
                  >
                    ◈ Go to Incident Overview
                  </button>
                </div>
                <div className="guide-action-col">
                  <div className="guide-info-card">
                    <div className="guide-info-icon">🛡</div>
                    <div className="guide-info-body">
                      <strong>Verification Safety Gate Rule:</strong>
                      <p>
                        Even though a matching learned article now exists in <code>knowledge.json</code>, the knowledge router explicitly checks:
                        <em>&quot;Unverified articles must not independently qualify for skipping Gemini.&quot;</em>
                      </p>
                      <p>
                        Because <code>verified: false</code>, the router decides:
                        <code>mode=&quot;ai&quot;</code>, <code>ai_skipped=false</code>, and Gemini runs a fresh investigation.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 7: VERIFY ARTICLE IN KNOWLEDGE HUB
             ========================================================================= */}
          <div id="guide-step-7" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 7</span>
              <h2>Verify the Article in Knowledge Hub</h2>
              <p className="muted">
                Open the Knowledge Hub page to inspect your learned articles, filter by origin, and officially verify the diagnosis.
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>How to Verify in the UI:</h4>
                  <ol>
                    <li>Click <strong>▤ Knowledge Hub</strong> in the sidebar.</li>
                    <li>Notice the metric cards: <strong>5 Static Runbooks</strong>, <strong>1+ Learned</strong>, and the count of <strong>Unverified</strong> articles.</li>
                    <li>Click the <strong>Learned</strong> filter tab to see your AI-generated article.</li>
                    <li>Click on the card to expand and review the AI diagnosis and recommended steps.</li>
                    <li>Click the <strong>Verification Switch</strong> to toggle status from <code>Unverified</code> to <code>Verified</code>.</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-primary"
                    onClick={onNavigateToKnowledge}
                  >
                    ▤ Open Knowledge Hub Now
                  </button>
                </div>
                <div className="guide-action-col">
                  <div className="guide-info-card">
                    <div className="guide-info-icon">✓</div>
                    <div className="guide-info-body">
                      <strong>Human-in-the-Loop Approval:</strong>
                      <p>
                        Once verified, the article&apos;s status badge turns green (<code>Verified</code>) and the change is permanently saved into <code>knowledge.json</code>.
                        The knowledge router is now authorized to use this article for sub-30ms automated incident matching!
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 8: EXACT MATCH & INSTANT REPORT (<30ms)
             ========================================================================= */}
          <div id="guide-step-8" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 8</span>
              <h2>Demonstrate Exact Match & Instant Report (&lt;30ms)</h2>
              <p className="muted">
                With the learned article now verified, run the investigation a third time. Watch the knowledge router achieve a 100% exact match and return an instant diagnosis without calling Gemini!
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>Trigger Third Investigation:</h4>
                  <ol>
                    <li>Return to <strong>◈ Incident overview</strong>.</li>
                    <li>Click <strong>Investigate Incident</strong>.</li>
                    <li>The result appears <strong>instantly</strong> in under 30 milliseconds!</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-primary"
                    onClick={onNavigateToOverview}
                  >
                    ◈ Go to Incident Overview
                  </button>
                </div>
                <div className="guide-action-col">
                  <div className="guide-info-card">
                    <div className="guide-info-icon">⚡</div>
                    <div className="guide-info-body">
                      <strong>The Power of Knowledge-Driven Investigation:</strong>
                      <ul>
                        <li><strong>Match Score:</strong> Exactly <code>1.00</code> (100% match).</li>
                        <li><strong>Mode:</strong> <code>knowledge</code> (deterministic runbook retrieval).</li>
                        <li><strong>AI Skipped:</strong> <code>true</code> (no Gemini API latency or cost).</li>
                        <li><strong>Speed:</strong> Execution drops from ~2500ms down to <strong>&lt; 30ms</strong>!</li>
                      </ul>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* =========================================================================
              STEP 9: RESET SCENARIO & RECOVERY
             ========================================================================= */}
          <div id="guide-step-9" className="guide-step-section">
            <div className="guide-step-header">
              <span className="guide-step-badge">STEP 9</span>
              <h2>Reset Scenario & Verify Recovery</h2>
              <p className="muted">
                Conclude the demonstration by resetting OrderFlow to the normal baseline, sending healthy traffic, and confirming full system recovery.
              </p>
            </div>

            <div className="guide-card-content">
              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>Recovery Steps in the UI:</h4>
                  <ol>
                    <li>On the Incident Overview page, switch the <strong>Failure Scenario</strong> dropdown to <strong>Normal</strong>.</li>
                    <li>(Optional) In Step 3 above, select <strong>🌿 Normal Baseline</strong> and run the healthy traffic command.</li>
                    <li>Wait 10 seconds for Prometheus metrics to reflect normal baseline conditions.</li>
                    <li>Click <strong>Investigate Incident</strong> to confirm all systems are healthy.</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-primary"
                    onClick={onNavigateToOverview}
                  >
                    ◈ Go to Incident Overview
                  </button>
                </div>
                <div className="guide-action-col">
                  <div className="guide-success-card">
                    <span className="guide-success-icon">🎉</span>
                    <div>
                      <strong>Full Demonstration Complete!</strong>
                      <p>
                        You have successfully walked through the complete SignalSleuth AI lifecycle: fault injection, automated MCP observability collection,
                        Gemini root cause analysis, automated knowledge capture, verification safety gates, and sub-30ms exact runbook reuse.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* TROUBLESHOOTING TAB */}
      {activeTab === "troubleshooting" && (
        <div className="guide-trouble-container">
          <div className="guide-trouble-header">
            <h2>Troubleshooting & Diagnostics Guide</h2>
            <p className="muted">
              Actionable diagnostics and copyable commands for the 10 most common setup, routing, and investigation issues.
            </p>
          </div>

          <div className="guide-trouble-grid">
            <div className="guide-trouble-card">
              <h4>1. Gemini Key Missing or Invalid</h4>
              <p>
                <strong>Symptom:</strong> Investigation returns &quot;Gemini API key is not configured&quot; or &quot;Gemini rejected the API key&quot;.
              </p>
              <p>
                <strong>Fix:</strong> Click <strong>⚙ Settings</strong> in the top-right header, paste your Google Gemini API key, and click <strong>Test & Save Connection</strong>.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>2. Traffic Not Reflected in Investigation</h4>
              <p>
                <strong>Symptom:</strong> Investigation says service is healthy even after running failure loop.
              </p>
              <p>
                <strong>Fix:</strong> Prometheus scrapes OrderFlow every 10 seconds. Wait 10–15 seconds after finishing your curl loop before clicking <strong>Investigate Incident</strong>.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>3. Nginx 502 Bad Gateway</h4>
              <p>
                <strong>Symptom:</strong> Accessing <code>https://34.69.34.84</code> gives HTTP 502.
              </p>
              <p>
                <strong>Fix:</strong> Check if Docker containers are running:
                <code>docker compose ps</code>. Ensure <code>orderflow</code> and <code>signal-agent</code> containers are up.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>4. CORS Errors in Browser Console</h4>
              <p>
                <strong>Symptom:</strong> Browser console shows <code>blocked by CORS policy</code>.
              </p>
              <p>
                <strong>Fix:</strong> Access the application through Nginx reverse proxy (port 80 or 443), which handles proxying and CORS headers automatically.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>5. OrderFlow Failure Scenario Not Persisting</h4>
              <p>
                <strong>Symptom:</strong> Scenario returns to normal unexpectedly.
              </p>
              <p>
                <strong>Fix:</strong> OrderFlow stores scenarios in memory. If the OrderFlow container restarts, it resets to normal. Reactivate the scenario from the dropdown.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>6. Knowledge Hub Verification Fails</h4>
              <p>
                <strong>Symptom:</strong> Clicking verify toggle returns an error.
              </p>
              <p>
                <strong>Fix:</strong> Static runbooks (<code>static-*</code>) are immutable and cannot be toggled. Only learned articles (<code>learned-*</code>) can be toggled.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>7. Third Investigation Still Calls Gemini</h4>
              <p>
                <strong>Symptom:</strong> Step 8 takes 2 seconds and does not skip AI.
              </p>
              <p>
                <strong>Fix:</strong> Ensure the learned article in Knowledge Hub has its toggle set to <strong>Verified</strong> (green badge).
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>8. Prometheus Metric Scrape Errors</h4>
              <p>
                <strong>Symptom:</strong> Signal Agent reports Prometheus queries returned empty results.
              </p>
              <p>
                <strong>Fix:</strong> Verify Prometheus is running on port 9090 and that target <code>orderflow:8080</code> is listed as UP in <code>http://localhost:9090/targets</code>.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>9. Loki Logs Not Showing in Evidence</h4>
              <p>
                <strong>Symptom:</strong> Signals list shows metrics but no log lines.
              </p>
              <p>
                <strong>Fix:</strong> Verify Grafana Alloy is forwarding container logs to Loki (port 3100). Alloy flushes logs periodically.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>10. Resetting All Learned Knowledge</h4>
              <p>
                <strong>Symptom:</strong> Want to reset the demo from scratch for a new presentation.
              </p>
              <p>
                <strong>Fix:</strong> Knowledge articles are saved in <code>knowledge.json</code>. You can remove learned items from <code>knowledge.json</code> to start fresh.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

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
  const [activeStep, setActiveStep] = useState<number>(1);
  const [selectedScenario, setSelectedScenario] = useState<FailureScenario>("redis_failure");
  const [activeTab, setActiveTab] = useState<"walkthrough" | "troubleshooting">("walkthrough");

  const steps = [
    { num: 1, title: "Configure Gemini API", short: "1. Gemini Setup" },
    { num: 2, title: "Activate Failure Scenario", short: "2. Activate Scenario" },
    { num: 3, title: "Generate Traffic", short: "3. Generate Traffic" },
    { num: 4, title: "Run AI Investigation", short: "4. Run Investigation" },
    { num: 5, title: "Verify Knowledge Learning", short: "5. Auto-Learning" },
    { num: 6, title: "Investigate Again (Unverified)", short: "6. Unverified Check" },
    { num: 7, title: "Verify in Knowledge Hub", short: "7. Verify Article" },
    { num: 8, title: "Exact Match & Instant Report", short: "8. 100% Match" },
    { num: 9, title: "Reset Scenario & Recovery", short: "9. Reset & Recovery" },
  ];

  return (
    <div className="how-to-use-page">
      {/* Header */}
      <div className="guide-hero-card">
        <div className="guide-hero-content">
          <span className="eyebrow">PRACTICAL INTERACTIVE PLAYBOOK</span>
          <h1 className="guide-hero-title">How to Use SignalSleuth AI</h1>
          <p className="guide-hero-subtitle">
            An end-to-end, zero-guesswork demonstration walkthrough with verified runnable Bash commands,
            traffic generation loops, Gemini root cause investigations, and automated knowledge learning.
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
            <span>📖</span> Step-by-Step Walkthrough
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

      {activeTab === "walkthrough" && (
        <div className="guide-walkthrough-layout">
          {/* Step Stepper Navigation Bar */}
          <div className="guide-stepper-bar">
            <div className="guide-stepper-label">
              <span>STEP {activeStep} OF {steps.length}</span>
              <strong>{steps[activeStep - 1]?.title}</strong>
            </div>

            <div className="guide-step-pills">
              {steps.map((s) => (
                <button
                  key={s.num}
                  type="button"
                  className={`guide-step-pill ${activeStep === s.num ? "active" : ""} ${s.num < activeStep ? "completed" : ""}`}
                  onClick={() => setActiveStep(s.num)}
                >
                  <span className="step-circle">{s.num < activeStep ? "✓" : s.num}</span>
                  <span className="step-text">{s.short}</span>
                </button>
              ))}
            </div>
          </div>

          {/* STEP 1 */}
          {activeStep === 1 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 1</span>
                <h2>Configure Gemini API Key</h2>
                <p className="muted">
                  SignalSleuth AI uses Gemini 2.5 Flash to automatically analyze telemetry evidence (Loki logs, Prometheus metrics, and deployment history).
                  You can configure your key via the secure Settings UI or via environment variable.
                </p>
              </div>

              <div className="guide-info-card">
                <div className="guide-info-icon">💡</div>
                <div className="guide-info-body">
                  <strong>Recommended: Use the Dashboard Settings Interface</strong>
                  <p>
                    Click the <strong>⚙ Settings</strong> button in the sidebar or top bar to open the configuration modal.
                    Paste your Gemini API key, click <strong>Test Connection</strong> to verify, and click <strong>Save Key</strong>.
                    The key is encrypted in memory, masked in the UI, and never exposed in browser or server logs.
                  </p>
                  <button
                    type="button"
                    className="button button-primary"
                    style={{ marginTop: 8 }}
                    onClick={onOpenSettings}
                  >
                    Open Settings Modal Now
                  </button>
                </div>
              </div>

              <CommandBlock
                title="Verify Gemini Key Configuration Status via API"
                environment="GCP VM Terminal (Signal Agent :8090)"
                description="Check whether Gemini is configured on the backend without leaking secret keys."
                command="curl -sS http://localhost:8090/api/settings/gemini"
                expectedOutput={`{
  "configured": true,
  "source": "user",
  "env_fallback_available": false,
  "user_key_configured": true
}`}
                verificationTip="The 'configured' field will be true, and 'source' will report either 'user' or 'environment'."
                troubleshootingTip="If 'configured' is false, open Settings in the UI or export GEMINI_API_KEY in .env and restart signal-agent."
              />

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(2)}
                >
                  Continue to Step 2: Activate Failure Scenario →
                </button>
              </div>
            </div>
          )}

          {/* STEP 2 */}
          {activeStep === 2 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 2</span>
                <h2>Activate a Failure Scenario</h2>
                <p className="muted">
                  OrderFlow includes an in-memory fault injection simulator. Select a failure scenario below to inspect its mechanics and view the exact activation command.
                </p>
              </div>

              {/* Scenario Selector Tabs */}
              <div className="guide-scenario-tabs">
                {[
                  { id: "redis_failure" as FailureScenario, name: "Redis Failure (Cache)", badge: "Degradation" },
                  { id: "db_failure" as FailureScenario, name: "Database Failure (Postgres)", badge: "Outage 500" },
                  { id: "payment_failure" as FailureScenario, name: "Payment Failure", badge: "Gateway 502" },
                  { id: "slow" as FailureScenario, name: "High Latency (Slow)", badge: "P99 Spike >2s" },
                  { id: "error_spike" as FailureScenario, name: "HTTP Error Spike", badge: "Catalog 500" },
                  { id: "normal" as FailureScenario, name: "Normal (Baseline)", badge: "Healthy" },
                ].map((sc) => (
                  <button
                    key={sc.id}
                    type="button"
                    className={`guide-scenario-btn ${selectedScenario === sc.id ? "active" : ""}`}
                    onClick={() => setSelectedScenario(sc.id)}
                  >
                    <strong>{sc.name}</strong>
                    <span className="guide-sc-badge">{sc.badge}</span>
                  </button>
                ))}
              </div>

              {/* Scenario-specific explanation & command */}
              {selectedScenario === "redis_failure" && (
                <div>
                  <div className="guide-info-card">
                    <div className="guide-info-icon">⚡</div>
                    <div className="guide-info-body">
                      <strong>Redis Failure Mechanics</strong>
                      <p>
                        Simulates Redis cache timeouts and connection failures on <code>product_cache.go</code>.
                        Because OrderFlow implements graceful fallback, requests still return HTTP 200 via PostgreSQL,
                        but Loki logs 40+ dependency errors and Prometheus marks the Redis window status as <strong>degraded</strong>.
                      </p>
                    </div>
                  </div>
                  <CommandBlock
                    title="Activate Redis Failure Scenario"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends a POST request to OrderFlow's failure injection endpoint to enable redis_failure."
                    command={`curl -sS -i -X POST http://localhost:8080/api/admin/failure-scenario \\
  -H 'Content-Type: application/json' \\
  -d '{"scenario":"redis_failure"}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{"scenario":"redis_failure"}`}
                    verificationTip="Verify the HTTP 200 response and JSON payload confirming scenario 'redis_failure'."
                    troubleshootingTip="Ensure orderflow is running on port 8080 (check 'docker compose ps orderflow')."
                  />
                </div>
              )}

              {selectedScenario === "db_failure" && (
                <div>
                  <div className="guide-info-card">
                    <div className="guide-info-icon">🗄</div>
                    <div className="guide-info-body">
                      <strong>Database Failure Mechanics</strong>
                      <p>
                        Simulates connection exhaustion and transaction failures in PostgreSQL.
                        Order creation (<code>POST /api/orders</code>) and catalog queries immediately fail with HTTP 500,
                        triggering database error logs in Loki and dependency degradation metrics.
                      </p>
                    </div>
                  </div>
                  <CommandBlock
                    title="Activate Database Failure Scenario"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends a POST request to activate db_failure in OrderFlow."
                    command={`curl -sS -i -X POST http://localhost:8080/api/admin/failure-scenario \\
  -H 'Content-Type: application/json' \\
  -d '{"scenario":"db_failure"}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{"scenario":"db_failure"}`}
                    verificationTip='Look for HTTP 200 with scenario "db_failure".'
                    troubleshootingTip="If using public HTTPS, route /api/failure-scenario through Nginx or use port 8080 locally on the VM."
                  />
                </div>
              )}

              {selectedScenario === "payment_failure" && (
                <div>
                  <div className="guide-info-card">
                    <div className="guide-info-icon">💳</div>
                    <div className="guide-info-body">
                      <strong>Payment Failure Mechanics</strong>
                      <p>
                        Simulates an outage of the external payment gateway provider in <code>payment_service.go</code>.
                        Payment transactions (<code>POST /api/payment</code>) return HTTP 502 Bad Gateway and log
                        <code>payment processing failed: payment provider unavailable (PAYMENT_GATEWAY_DOWN)</code>.
                      </p>
                    </div>
                  </div>
                  <CommandBlock
                    title="Activate Payment Failure Scenario"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends a POST request to activate payment_failure."
                    command={`curl -sS -i -X POST http://localhost:8080/api/admin/failure-scenario \\
  -H 'Content-Type: application/json' \\
  -d '{"scenario":"payment_failure"}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{"scenario":"payment_failure"}`}
                    verificationTip="Confirm scenario status is 'payment_failure'."
                    troubleshootingTip='Check that the JSON request body contains scenario "payment_failure" exactly.'
                  />
                </div>
              )}

              {selectedScenario === "slow" && (
                <div>
                  <div className="guide-info-card">
                    <div className="guide-info-icon">⏱</div>
                    <div className="guide-info-body">
                      <strong>High Latency Mechanics</strong>
                      <p>
                        Injects a 2000ms delay in <code>OrderService.CreateOrder()</code>.
                        Requests succeed with HTTP 201 Created, but response times increase to &gt;2.0s,
                        triggering elevated P95 and P99 latency alerts in Prometheus.
                      </p>
                    </div>
                  </div>
                  <CommandBlock
                    title="Activate High Latency Scenario"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Activates simulated 2-second sleep on order processing."
                    command={`curl -sS -i -X POST http://localhost:8080/api/admin/failure-scenario \\
  -H 'Content-Type: application/json' \\
  -d '{"scenario":"slow"}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{"scenario":"slow"}`}
                    verificationTip="Look for HTTP 200 confirming scenario 'slow'."
                    troubleshootingTip="Make sure the scenario name is 'slow', not 'latency'."
                  />
                </div>
              )}

              {selectedScenario === "error_spike" && (
                <div>
                  <div className="guide-info-card">
                    <div className="guide-info-icon">💥</div>
                    <div className="guide-info-body">
                      <strong>HTTP Error Spike Mechanics</strong>
                      <p>
                        Simulates an unhandled exception in <code>product_service.go</code> on <code>GET /api/products</code>.
                        Returns immediate HTTP 500 and logs catalog failure spike, driving the service error rate to 100%.
                      </p>
                    </div>
                  </div>
                  <CommandBlock
                    title="Activate Error Spike Scenario"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Activates simulated catalog error spike."
                    command={`curl -sS -i -X POST http://localhost:8080/api/admin/failure-scenario \\
  -H 'Content-Type: application/json' \\
  -d '{"scenario":"error_spike"}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{"scenario":"error_spike"}`}
                    verificationTip="Confirm scenario payload shows 'error_spike'."
                    troubleshootingTip="Check Docker logs if OrderFlow fails to accept the scenario."
                  />
                </div>
              )}

              {selectedScenario === "normal" && (
                <div>
                  <div className="guide-info-card">
                    <div className="guide-info-icon">🌿</div>
                    <div className="guide-info-body">
                      <strong>Normal Baseline Mechanics</strong>
                      <p>
                        Clears all active fault simulations. All database, cache, and payment operations run normally
                        with &lt;10ms response times and 0% error rate.
                      </p>
                    </div>
                  </div>
                  <CommandBlock
                    title="Reset to Normal Scenario"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Restores OrderFlow to healthy baseline operation."
                    command={`curl -sS -i -X POST http://localhost:8080/api/admin/failure-scenario \\
  -H 'Content-Type: application/json' \\
  -d '{"scenario":"normal"}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{"scenario":"normal"}`}
                    verificationTip="Confirm scenario status is 'normal'."
                    troubleshootingTip="Use this command whenever you want to reset the environment."
                  />
                </div>
              )}

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(1)}
                >
                  ← Back to Step 1
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(3)}
                >
                  Continue to Step 3: Generate Traffic →
                </button>
              </div>
            </div>
          )}

          {/* STEP 3 */}
          {activeStep === 3 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 3</span>
                <h2>Generate Scenario-Specific Traffic</h2>
                <p className="muted">
                  Different scenarios affect different application dependencies. Run the exact Bash loop matching your active scenario to generate telemetry evidence in Prometheus and Loki.
                </p>
              </div>

              {/* Scenario-specific traffic commands */}
              {selectedScenario === "redis_failure" && (
                <div>
                  <div className="guide-tag-row">
                    <span className="environment-pill">Active Target: <strong>GET /api/products/1</strong></span>
                    <span className="environment-pill">Target Dependency: <strong>redis</strong></span>
                  </div>
                  <CommandBlock
                    title="Generate Redis Cache Degradation Traffic"
                    environment="GCP VM Terminal (OrderFlow :8080)"
                    description="Sends 10 product requests at 1-second intervals. Redis cache misses will trigger internal fallback logs."
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
                    verificationTip="Observe HTTP 201 Created but total elapsed time &gt; 2.0 seconds."
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
                    verificationTip="All requests should complete with HTTP 200 in &lt;10ms."
                    troubleshootingTip="If errors persist, make sure scenario 'normal' was accepted."
                  />
                </div>
              )}

              {/* Crucial Telemetry Wait Callout */}
              <div className="guide-warning-banner">
                <span className="guide-warn-icon">⏳</span>
                <div>
                  <strong>Important: Wait 10 to 15 seconds after generating traffic</strong>
                  <p>
                    Prometheus scrapes OrderFlow metrics every 10 seconds, and Grafana Alloy flushes container logs into Loki.
                    Waiting 10–15 seconds ensures full telemetry is available before launching the investigation.
                  </p>
                </div>
              </div>

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(2)}
                >
                  ← Back to Step 2
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(4)}
                >
                  Continue to Step 4: Run AI Investigation →
                </button>
              </div>
            </div>
          )}

          {/* STEP 4 */}
          {activeStep === 4 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 4</span>
                <h2>Run an Incident Investigation</h2>
                <p className="muted">
                  Launch an automated investigation. Signal Agent connects to the Observability MCP server to query Prometheus and Loki,
                  assembles live telemetry, checks knowledge matches, and uses Gemini to synthesize a diagnosis.
                </p>
              </div>

              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>Option A — Through the Web UI (Recommended)</h4>
                  <ol>
                    <li>Click <strong>◈ Incident overview</strong> in the sidebar.</li>
                    <li>Ensure Service is set to <strong>orderflow</strong> and Duration to <strong>5m</strong> or <strong>10m</strong>.</li>
                    <li>Click the purple <strong>Investigate Incident</strong> button.</li>
                    <li>Observe the live spinner while evidence is gathered and analyzed.</li>
                  </ol>
                  <button
                    type="button"
                    className="button button-secondary"
                    onClick={onNavigateToOverview}
                  >
                    Go to Incident Overview Dashboard ➔
                  </button>
                </div>
                <div className="guide-action-col">
                  <h4>Option B — Via Terminal / API</h4>
                  <p>You can also invoke the investigation API directly from Bash:</p>
                  <CommandBlock
                    title="Execute Investigation via API"
                    environment="GCP VM Terminal (Signal Agent :8090)"
                    description="Calls the Signal Agent investigation endpoint directly."
                    command={`curl -sS -i -X POST http://localhost:8090/api/investigations \\
  -H 'Content-Type: application/json' \\
  -d '{"service":"orderflow","duration":"5m"}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{
  "service": "orderflow",
  "duration": "5m",
  "investigation_mode": "ai",
  "ai_skipped": false,
  "match_score": 0,
  "duration_ms": 2341,
  "evidence": {
    "status": "degraded",
    "severity": "medium",
    "signals": [...]
  },
  "ai_explanation": "### Summary\\nThe orderflow service is marked as degraded..."
}`}
                    verificationTip="Check that investigation_mode is 'ai', duration_ms is ~2000ms, and ai_explanation contains a full diagnosis."
                    troubleshootingTip="If duration_ms is very high, check network access to Google Gemini API."
                  />
                </div>
              </div>

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(3)}
                >
                  ← Back to Step 3
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(5)}
                >
                  Continue to Step 5: Verify Knowledge Learning →
                </button>
              </div>
            </div>
          )}

          {/* STEP 5 */}
          {activeStep === 5 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 5</span>
                <h2>Verify that New Knowledge Was Learned</h2>
                <p className="muted">
                  Notice what happens after the first AI investigation completes: Signal Agent extracts the error signature, synthesizes a structured runbook article, and persists it into <code>knowledge.json</code>.
                </p>
              </div>

              <div className="guide-info-card">
                <div className="guide-info-icon">🧠</div>
                <div className="guide-info-body">
                  <strong>Automatic Incident Knowledge Learning Rules</strong>
                  <ul>
                    <li>An article is created ONLY when Gemini successfully diagnoses an active incident (status is not 'healthy' or 'normal').</li>
                    <li>Newly learned articles <strong>always default to unverified (verified: false)</strong> to require human-in-the-loop validation.</li>
                    <li>The article is identified with a deterministic stable ID (e.g., <code>learned-orderflow-redis</code>).</li>
                  </ul>
                </div>
              </div>

              <CommandBlock
                title="List Persisted Knowledge via API"
                environment="GCP VM Terminal (Signal Agent :8090)"
                description="Inspect all static runbooks and newly learned articles stored in the system."
                command={`curl -sS http://localhost:8090/api/knowledge | grep -E '"(title|id|verified|source)":'`}
                expectedOutput={`"id": "static-postgresql-failure", "source": "static", "title": "PostgreSQL Failure", "verified": true,
"id": "static-redis-failure", "source": "static", "title": "Redis Failure", "verified": true,
...
"id": "learned-orderflow-redis", "source": "AI-generated", "title": "Learned Incident: orderflow - redis", "verified": false`}
                verificationTip="Notice that static articles are always verified: true, while the newly learned article has verified: false."
                troubleshootingTip="If no learned article appeared, make sure Gemini was configured and the incident had active errors (status != healthy)."
              />

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(4)}
                >
                  ← Back to Step 4
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(6)}
                >
                  Continue to Step 6: Investigate Again (Unverified Check) →
                </button>
              </div>
            </div>
          )}

          {/* STEP 6 */}
          {activeStep === 6 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 6</span>
                <h2>Investigate Again: Unverified Articles Cannot Skip AI</h2>
                <p className="muted">
                  Run an investigation again while the learned article remains unverified. This step proves that SignalSleuth AI protects against unvetted automation.
                </p>
              </div>

              <div className="guide-info-card">
                <div className="guide-info-icon">🛡</div>
                <div className="guide-info-body">
                  <strong>Verification Safety Gate Rule</strong>
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

              <CommandBlock
                title="Execute Second Investigation (Unverified Article Test)"
                environment="GCP VM Terminal (Signal Agent :8090)"
                description="Trigger an investigation while the learned article is still unverified."
                command={`curl -sS -i -X POST http://localhost:8090/api/investigations \\
  -H 'Content-Type: application/json' \\
  -d '{"service":"orderflow","duration":"5m"}' | grep -E '"(investigation_mode|ai_skipped|decision_reason|duration_ms)":'`}
                expectedOutput={`"duration_ms": 2185,
"investigation_mode": "ai",
"ai_skipped": false,
"decision_reason": "No verified article matches the current incident sufficiently."`}
                verificationTip="Notice ai_skipped is false and duration_ms took ~2 seconds because Gemini had to run."
                troubleshootingTip="If ai_skipped was true, someone already verified the article in Knowledge Hub."
              />

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(5)}
                >
                  ← Back to Step 5
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(7)}
                >
                  Continue to Step 7: Verify in Knowledge Hub →
                </button>
              </div>
            </div>
          )}

          {/* STEP 7 */}
          {activeStep === 7 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 7</span>
                <h2>Verify the Article in Knowledge Hub</h2>
                <p className="muted">
                  Open the Knowledge Hub page to inspect your learned articles, filter by origin, and officially verify the diagnosis.
                </p>
              </div>

              <div className="guide-action-box">
                <div className="guide-action-col">
                  <h4>UI Verification Walkthrough</h4>
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
                    Open Knowledge Hub Now ➔
                  </button>
                </div>
                <div className="guide-action-col">
                  <h4>Equivalent API Command</h4>
                  <p>You can also toggle verification status via the REST API:</p>
                  <CommandBlock
                    title="Verify Learned Article via PATCH API"
                    environment="GCP VM Terminal (Signal Agent :8090)"
                    description="Updates verified status of a learned article to true in knowledge.json."
                    command={`curl -sS -i -X PATCH http://localhost:8090/api/knowledge/learned-orderflow-redis/verification \\
  -H 'Content-Type: application/json' \\
  -d '{"verified":true}'`}
                    expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "learned-orderflow-redis",
  "title": "Learned Incident: orderflow - redis",
  "verified": true,
  "match_origin": "learned"
}`}
                    verificationTip="Look for HTTP 200 and 'verified': true in the returned payload."
                    troubleshootingTip="Attempts to modify static runbooks (static-*) are rejected with HTTP 400 Bad Request."
                  />
                </div>
              </div>

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(6)}
                >
                  ← Back to Step 6
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(8)}
                >
                  Continue to Step 8: Exact Match & Instant Report →
                </button>
              </div>
            </div>
          )}

          {/* STEP 8 */}
          {activeStep === 8 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 8</span>
                <h2>Demonstrate Exact Match & Instant Report (&lt;30ms)</h2>
                <p className="muted">
                  With the learned article now verified, run the investigation a third time. Watch the knowledge router achieve a 100% exact match and return an instant diagnosis without calling Gemini!
                </p>
              </div>

              <div className="guide-info-card">
                <div className="guide-info-icon">⚡</div>
                <div className="guide-info-body">
                  <strong>The Power of Knowledge-Driven Investigation</strong>
                  <p>
                    When an incoming incident signature matches a verified article above the threshold (score &ge; 0.90):
                  </p>
                  <ul>
                    <li><strong>Match Score:</strong> Exactly <code>1.00</code> (100% match).</li>
                    <li><strong>Mode:</strong> <code>knowledge</code> (deterministic runbook retrieval).</li>
                    <li><strong>AI Skipped:</strong> <code>true</code> (no Gemini API latency or cost).</li>
                    <li><strong>Speed:</strong> Execution drops from ~2500ms down to <strong>&lt; 30ms</strong>!</li>
                  </ul>
                </div>
              </div>

              <CommandBlock
                title="Execute Third Investigation (Verified 100% Match)"
                environment="GCP VM Terminal (Signal Agent :8090)"
                description="Triggers the third investigation against the verified knowledge base."
                command={`curl -sS -i -X POST http://localhost:8090/api/investigations \\
  -H 'Content-Type: application/json' \\
  -d '{"service":"orderflow","duration":"5m"}' | grep -E '"(investigation_mode|ai_skipped|match_score|decision_reason|duration_ms)":'`}
                expectedOutput={`"duration_ms": 18,
"investigation_mode": "knowledge",
"ai_skipped": true,
"match_score": 1,
"decision_reason": "A verified article matches the current incident above the threshold."`}
                verificationTip="Verify that duration_ms is < 30ms, ai_skipped is true, and match_score is 1."
                troubleshootingTip="If match_score is 0, verify that the article in Knowledge Hub has its toggle set to 'Verified'."
              />

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(7)}
                >
                  ← Back to Step 7
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveStep(9)}
                >
                  Continue to Step 9: Reset Scenario & Recovery →
                </button>
              </div>
            </div>
          )}

          {/* STEP 9 */}
          {activeStep === 9 && (
            <div className="guide-step-section">
              <div className="guide-step-header">
                <span className="guide-step-badge">STEP 9</span>
                <h2>Reset Scenario & Verify Recovery</h2>
                <p className="muted">
                  Conclude the demonstration by resetting OrderFlow to the normal baseline, sending healthy traffic, and confirming full system recovery.
                </p>
              </div>

              <CommandBlock
                title="1. Reset Failure Scenario to Normal"
                environment="GCP VM Terminal (OrderFlow :8080)"
                description="Clears all active fault simulations."
                command={`curl -sS -i -X POST http://localhost:8080/api/admin/failure-scenario \\
  -H 'Content-Type: application/json' \\
  -d '{"scenario":"normal"}'`}
                expectedOutput={`HTTP/1.1 200 OK
Content-Type: application/json

{"scenario":"normal"}`}
                verificationTip="Look for HTTP 200 confirming scenario 'normal'."
                troubleshootingTip="OrderFlow failure simulator is in-memory and resets immediately."
              />

              <CommandBlock
                title="2. Generate Healthy Recovery Traffic"
                environment="GCP VM Terminal (OrderFlow :8080)"
                description="Sends 10 healthy requests to clear error rate windows."
                command={`for i in $(seq 1 10); do
  curl -sS -o /dev/null -w 'HTTP %{http_code} | Time %{time_total}s\\n' \\
    http://localhost:8080/api/products/1
  sleep 1
done`}
                expectedOutput={`HTTP 200 | Time 0.0049s
HTTP 200 | Time 0.0047s
...`}
                verificationTip="All requests complete with HTTP 200 in &lt;10ms."
                troubleshootingTip="Wait 10 seconds for Prometheus metrics to reflect the healthy window."
              />

              <CommandBlock
                title="3. Run Final Investigation to Confirm Health"
                environment="GCP VM Terminal (Signal Agent :8090)"
                description="Verifies that service status is restored to healthy."
                command={`curl -sS -i -X POST http://localhost:8090/api/investigations \\
  -H 'Content-Type: application/json' \\
  -d '{"service":"orderflow","duration":"5m"}' | grep -E '"(status|severity|http_error_rate)":'`}
                expectedOutput={`"status": "healthy",
"severity": "normal",
"http_error_rate": 0`}
                verificationTip="Status should be 'healthy', severity 'normal', and error rate 0."
                troubleshootingTip="If errors are still shown, wait 10 seconds for older error timestamps to expire from the 5m query window."
              />

              <div className="guide-success-card">
                <span className="guide-success-icon">🎉</span>
                <div>
                  <strong>Congratulations! Full Demonstration Complete</strong>
                  <p>
                    You have successfully demonstrated the entire SignalSleuth AI lifecycle: fault injection, automated MCP observability collection,
                    Gemini root cause analysis, automated knowledge capture, verification safety gates, and sub-30ms exact runbook reuse.
                  </p>
                </div>
              </div>

              <div className="guide-step-actions">
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() => setActiveStep(8)}
                >
                  ← Back to Step 8
                </button>
                <button
                  type="button"
                  className="button button-primary"
                  onClick={() => setActiveTab("troubleshooting")}
                >
                  Open Troubleshooting & Diagnostics →
                </button>
              </div>
            </div>
          )}
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
                <strong>Fix:</strong> Open ⚙ Settings in the sidebar, paste a valid Gemini 2.5 key, and click Test Connection.
                Ensure your key has quota enabled in Google AI Studio.
              </p>
              <pre className="guide-mini-code"><code>curl -sS http://localhost:8090/api/settings/gemini</code></pre>
            </div>

            <div className="guide-trouble-card">
              <h4>2. Scenario Activation Returning 400 Bad Request</h4>
              <p>
                <strong>Symptom:</strong> <code>invalid request body</code> or <code>unknown failure scenario</code>.
              </p>
              <p>
                <strong>Fix:</strong> Supported scenario strings are: <code>normal</code>, <code>db_failure</code>, <code>redis_failure</code>, <code>payment_failure</code>, <code>error_spike</code>, <code>slow</code>.
                Ensure request header includes <code>Content-Type: application/json</code>.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>3. Bash Commands Failing Due to Routing / Port Mismatch</h4>
              <p>
                <strong>Symptom:</strong> Connection refused on port 8080 or 8090.
              </p>
              <p>
                <strong>Fix:</strong> On the GCP VM, services are published on their native ports:
                OrderFlow is on <code>http://localhost:8080</code>, Signal Agent is on <code>http://localhost:8090</code>,
                and the public Nginx proxy is on <code>https://34.69.34.84</code> (ports 80 and 443).
              </p>
              <pre className="guide-mini-code"><code>docker compose ps</code></pre>
            </div>

            <div className="guide-trouble-card">
              <h4>4. Unexpected HTTP Status Codes During Traffic Generation</h4>
              <p>
                <strong>Symptom:</strong> Getting 404 Not Found on traffic endpoints.
              </p>
              <p>
                <strong>Fix:</strong> Verify endpoints against OrderFlow:
                <code>GET /api/products</code>, <code>GET /api/products/1</code>, <code>POST /api/orders</code>, <code>POST /api/payment</code>.
                Make sure you are calling port 8080 directly on the VM.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>5. No Metrics or Logs Appearing in Investigation</h4>
              <p>
                <strong>Symptom:</strong> Investigation shows healthy or 0 errors despite generating failure traffic.
              </p>
              <p>
                <strong>Fix:</strong> Prometheus scrape interval is 10s and Loki batches logs.
                Wait 10–15 seconds after running your traffic loop before clicking Investigate.
              </p>
              <pre className="guide-mini-code"><code>docker compose logs --tail=30 orderflow-loki</code></pre>
            </div>

            <div className="guide-trouble-card">
              <h4>6. Investigation Producing No Useful Findings</h4>
              <p>
                <strong>Symptom:</strong> Status is healthy with 0 signals.
              </p>
              <p>
                <strong>Fix:</strong> Make sure traffic was sent during the selected duration window (e.g. 5m).
                If you ran traffic 15 minutes ago, select &quot;15m&quot; in the investigation duration selector.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>7. Incident Article Not Being Learned</h4>
              <p>
                <strong>Symptom:</strong> No new article appears in Knowledge Hub after investigation.
              </p>
              <p>
                <strong>Fix:</strong> An article is only learned when: (a) Gemini was configured and completed analysis without error, (b) the service status was not 'healthy', and (c) an identical article signature was not already saved.
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>8. Knowledge Hub Returning No Match or Score Below 100%</h4>
              <p>
                <strong>Symptom:</strong> Investigation mode stays &quot;ai&quot; even though article exists.
              </p>
              <p>
                <strong>Fix:</strong> Check that the article's verification switch in Knowledge Hub is toggled to <strong>Verified</strong>.
                Unverified articles cannot skip AI. Also ensure you are investigating the same service (<code>orderflow</code>).
              </p>
            </div>

            <div className="guide-trouble-card">
              <h4>9. HTTPS / Let&apos;s Encrypt / Nginx Proxy Failures</h4>
              <p>
                <strong>Symptom:</strong> Browser shows SSL error or 502 Bad Gateway.
              </p>
              <p>
                <strong>Fix:</strong> Check Nginx container logs and upstream availability:
              </p>
              <pre className="guide-mini-code"><code>docker compose logs --tail=50 signal-ui</code></pre>
            </div>

            <div className="guide-trouble-card">
              <h4>10. Docker Containers Crashing or Restarting</h4>
              <p>
                <strong>Symptom:</strong> Service unavailable or 502.
              </p>
              <p>
                <strong>Fix:</strong> Check container health status across the stack:
              </p>
              <pre className="guide-mini-code"><code>docker compose ps -a</code></pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

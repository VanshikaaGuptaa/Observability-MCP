# OrderFlow Service Runbook

## Overview
OrderFlow is the core backend service in the Observability Platform. It handles product catalog requests, order processing, and payment orchestration.

- **Primary Port**: 8080
- **Health Endpoint**: `/api/health`
- **Metrics Endpoint**: `/metrics`
- **Dependencies**: PostgreSQL (database), Redis (product cache), External Payment Provider

---

## 1. Symptoms
- Service health reports `down` or `unknown` in `get_service_health`.
- HTTP 5xx error rate spikes above 5% (`get_error_rate`).
- P99 latency exceeds 2.0s on critical paths like `/api/orders` or `/api/products` (`get_latency`).
- Elevated error logs in Loki (`get_recent_errors`).

---

## 2. Evidence to Inspect
- **Service Health**: Check Prometheus `up{job="orderflow"}` via `get_service_health`.
- **Traffic & Errors**: Compare `get_request_rate` and `get_error_rate` over 5m, 15m, and 1h windows.
- **Latency**: Check P50, P95, and P99 latency for `/api/orders`, `/api/products`, and `/api/payment`.
- **Recent Deployments**: Run `get_recent_deployments` to check for recent changes that temporally correlate with incident onset.
- **Root Investigation**: Run `investigate_service` or `investigate_incident` to correlate metrics, logs, and dependencies.

---

## 3. Expected Impact
- **Database Degradation**: Causes user-facing outages on `/api/orders` and potential failures on `/api/products`. Critical impact.
- **Redis Degradation**: Product reads fall back to PostgreSQL. Error rate typically remains 0%, but latency may increase slightly. Medium impact.
- **Payment Degradation**: Payment transactions fail on `/api/payment`. Browsing and order drafting unaffected. High impact.

---

## 4. Safe Diagnostic Checks
- Query `/api/health` to check database and cache status without modifying state.
- Inspect application logs using `search_logs` with queries like `error`, `timeout`, or specific dependencies.
- Verify container and network status via standard Docker / orchestration read-only commands.

---

## 5. Recovery Verification
- `up{job="orderflow"}` returns 1.
- HTTP 5xx error rate drops to 0.0%.
- Latencies (P95, P99) return to normal baselines (< 1.0s).
- Endpoint statuses in `investigate_service` return to `healthy`.

---

## 6. Dangerous Actions (DO NOT PERFORM AUTOMATICALLY)
- Do not restart production database or redis containers without verifying data integrity and backup status.
- Do not execute destructive SQL statements (`DROP`, `TRUNCATE`, or manual migration rollbacks).
- Do not flush Redis cache under heavy production load, as the resulting cache stampede may crash PostgreSQL.

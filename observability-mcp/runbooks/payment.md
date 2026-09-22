# Payment Provider Runbook

## Overview
OrderFlow integrates with an external payment provider service to authorize and process payment transactions during checkout (`POST /api/payment`).

- **Endpoint**: `/api/payment`
- **Metric**: `orderflow_payment_failure_total`

---

## 1. Symptoms
- Prometheus metric `orderflow_payment_failure_total` increasing over time.
- HTTP 500 or 502 responses on `POST /api/payment`.
- Loki logs containing `payment provider unavailable`, `payment request failed`, or `payment transaction failed`.
- `investigate_incident` identifies payment provider hypothesis with high confidence.

---

## 2. Evidence to Inspect
- Run `investigate_service` and inspect `dependencies` where `name == "payment-provider"`.
- Run `get_latency` on `/api/payment` to verify if upstream timeouts are causing the failures.
- Check Prometheus metric rate: `increase(orderflow_payment_failure_total[5m])`.
- Review recent error logs via `search_logs` with query `"payment"`.

---

## 3. Expected Impact
- Users cannot complete checkouts or finalize payments on `/api/payment`.
- Browsing products (`/api/products`) and drafting orders (`/api/orders`) may remain functional.
- High severity impact on revenue transactions.

---

## 4. Safe Diagnostic Checks
- Check upstream payment gateway status and public dashboard for known outages.
- Test external network connectivity and DNS resolution from the OrderFlow host.
- Inspect payment provider error codes and response headers in Loki logs.

---

## 5. Recovery Verification
- `orderflow_payment_failure_total` stops incrementing.
- `investigate_service` reports `current_status: healthy` for `payment-provider`.
- A test payment request to `/api/payment` succeeds with HTTP 200.

---

## 6. Dangerous Actions (DO NOT PERFORM AUTOMATICALLY)
- Do NOT blindly replay failed transactions without checking provider idempotent transaction state to avoid double-charging users.
- Do NOT store or expose raw payment credentials or card details in logs or tickets.
- Do NOT disable fraud prevention or payment validation middlewares to bypass errors.

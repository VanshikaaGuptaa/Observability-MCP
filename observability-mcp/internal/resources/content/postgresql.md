# PostgreSQL Database Runbook

## Overview
OrderFlow utilizes PostgreSQL (pgx connection pool) as the primary system of record for inventory, products, and order data.

- **Default Port**: 5432
- **Database Name**: `orderflow_db`
- **Tables**: `orders`, `order_items`, `products`

---

## 1. Symptoms
- Loki error logs matching `{service_name="orderflow"} | json | dependency="database"`.
- Error messages containing `pq: connection refused`, `failed to acquire connection`, or `database query failed`.
- Spikes in HTTP 5xx errors on `POST /api/orders` and `GET /api/products`.
- Total HTTP error rate > 5% reported by `get_error_rate`.

---

## 2. Evidence to Inspect
- Run `investigate_service` or `investigate_incident`:
  - Review `dependencies` where `name == "postgresql"`.
  - Review `impact.endpoints` for `/api/orders`.
- Run `get_error_rate` for duration `5m` and `15m`.
- Search error logs via `get_recent_errors` to identify exact PostgreSQL error messages.

---

## 3. Expected Impact
- **Critical User-Facing Impact**: Users cannot place new orders (`POST /api/orders` fails with 500).
- If cache misses occur or Redis is cold, `/api/products` also fails.
- This represents a true service degradation/outage directly affecting business transactions.

---

## 4. Safe Diagnostic Checks
- Check PostgreSQL readiness: `pg_isready -U postgres -d orderflow_db`.
- Inspect container status and resource usage: `docker stats orderflow-postgres`.
- Run safe, read-only status query: `SELECT count(*) FROM pg_stat_activity WHERE state = 'active';`.
- Check database disk space and WAL storage.

---

## 5. Recovery Verification
- Database error logs cease in Loki.
- `investigate_service` reflects `current_status: healthy` for `postgresql`.
- Test order creation request to `POST /api/orders` succeeds with HTTP 201.
- Service HTTP error rate returns to 0.0%.

---

## 6. Dangerous Actions (DO NOT PERFORM AUTOMATICALLY)
- Do NOT run `DROP TABLE`, `TRUNCATE`, or manual delete queries on production data.
- Do NOT perform hard restarts or `kill -9` on the PostgreSQL primary process during active writes.
- Do NOT alter schema constraints or re-run initialization scripts (`001_init.sql`) without explicit DBA sign-off.

# Redis Cache Runbook

## Overview
OrderFlow uses Redis to cache product catalog data (`/api/products` and `/api/products/{id}`). The application is architected with a database fallback mechanism: if Redis operations fail, product queries are served directly from PostgreSQL.

- **Default Port**: 6379
- **Key Prefix**: `product:*`

---

## 1. Symptoms
- Loki error logs matching `{service_name="orderflow"} | json | dependency="redis"`.
- Error messages indicating `connection refused`, `i/o timeout`, or `redis connection failed`.
- Elevated Redis error count reported in `investigate_service` under dependencies.
- Slight increase in database query latency and connection pool pressure.

---

## 2. Evidence to Inspect
- Run `investigate_service` and inspect `dependencies` where `name == "redis"`. Note whether `current_status` is degraded or healthy (historical).
- Check `get_latency` on `/api/products` to observe fallback latency overhead.
- Check `get_error_rate` on `orderflow`: if the fallback is working correctly, the HTTP error rate should remain 0%.
- Search logs using `search_logs` with query `"redis"`.

---

## 3. Expected Impact
- Product catalog requests succeed via PostgreSQL database fallback.
- User-facing error rate does NOT increase.
- Database query volume increases proportional to product catalog traffic.
- Degraded performance is limited to cache misses; this does NOT constitute a full service outage.

---

## 4. Safe Diagnostic Checks
- Check Redis container health status: `docker inspect orderflow-redis`.
- Run safe non-destructive ping check: `redis-cli ping`.
- Verify memory usage and connection count via `redis-cli info memory` and `redis-cli info clients`.
- Check PostgreSQL connection pool metrics to ensure database is handling the fallback load safely.

---

## 5. Recovery Verification
- Redis errors stop appearing in Loki logs.
- `investigate_service` reports `current_status: healthy` for Redis (even if `window_status` was degraded earlier).
- Latencies on `/api/products` return to sub-10ms cache baseline.

---

## 6. Dangerous Actions (DO NOT PERFORM AUTOMATICALLY)
- Do NOT execute `FLUSHALL` or `FLUSHDB` on active Redis instances.
- Do NOT kill or restart the Redis container without checking database load, to prevent a cache stampede.
- Do NOT modify Redis persistent configuration files without version control tracking.

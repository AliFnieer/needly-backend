# Load Test Results

Baseline performance results for the v1.0 roadmap. Reproduce with:

```bash
k6 run loadtests/load.js              # defaults to http://localhost:8080
BASE_URL=http://staging k6 run loadtests/load.js
```

## Run — 2026-08-22

| | |
| --- | --- |
| k6 version | v2.2.0 |
| Build | release mode (`GIN_MODE=release`), commit `6f08494` + open changes |
| Database | PostgreSQL 16.15 (native), fresh database, all SQL migrations applied |
| Cache/queue | Redis protocol endpoint (in-process miniredis) |
| Topology | Single API instance, HTTP (no TLS), loopback network |
| Profile | Ramp 20 → 50 → 100 VUs over 5 minutes (script default) |

### Results

| Metric | Value |
| --- | --- |
| Total requests | 60,958 (~203 req/s sustained) |
| Iterations | 15,239 |
| p50 latency | 0.94 ms |
| p90 latency | 2.42 ms |
| **p95 latency** | **2.94 ms** |
| Max latency | 64.03 ms |
| Failed requests | 1 / 60,958 (0.00%) |

Checks: health 200 ✓ · auth/me 200 ✓ · households list 200 ✓ · create household 201 ✓

### Thresholds

| Threshold | Target | Actual | Status |
| --- | --- | --- | --- |
| p95 response time | < 500 ms | 2.94 ms | ✅ PASS |
| Error rate | < 10% | 0.00% | ✅ PASS |

### Notes

* Latency budget is dominated by bcrypt login cost only at authentication; steady-state
  request handling sits in the low single-digit milliseconds.
* The Redis component used for this run was a protocol-compatible in-process server
  (`miniredis`), not `redis-server`. Rate limiting and cache paths were exercised, but
  absolute numbers should be re-validated on the real staging topology before capacity
  planning.
* Single-instance numbers do not include WebSocket Pub/Sub fan-out across instances.

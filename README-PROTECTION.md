# Fleet API Protection

Adds protection against transient Fleet API issues that cause application deletions.

## Problem

During Redis restarts or maintenance, Fleet API occasionally returns incomplete responses.
The plugin didn't validate or retry, causing ArgoCD to delete applications.

**Incident 2026-01-30:**
- API returned 6 bindings instead of 12
- 3 applications deleted
- 2-12 minutes downtime each

## Solution

Three-layer protection:

1. **Detection** - Identifies transient issues by pattern:
   - Oscillation: count changes 2+ times in 10 minutes (12→6→12)
   - Sudden drop: >30% decrease

2. **Retry** - 3 attempts with exponential backoff (2s, 4s, 8s)

3. **Cache** - Falls back to last known good response (60 min TTL)

## Configuration

Environment variables with defaults:

```bash
MAX_API_RETRIES=3
RETRY_BASE_DELAY_SECONDS=2
CACHE_MAX_AGE_MINUTES=60
DETECTION_WINDOW_MINUTES=10
OSCILLATION_THRESHOLD=2
DROP_THRESHOLD_PERCENT=30
```

## Testing

```bash
cd fleet-argocd-plugin
go test ./protection/...
go test ./fleetclient/...
```

All tests pass. Incident replay confirms 2026-01-30 pattern is detected.

## Changes

**Modified files (2):**
- `fleetclient/fleetclient.go` - Added protection wrapper
- `main.go` - Added config loading

**New files (4):**
- `protection/detector.go` - Transient issue detection
- `protection/cache.go` - Response caching
- `protection/*_test.go` - Test coverage

Zero breaking changes. Protection is transparent in normal operation.

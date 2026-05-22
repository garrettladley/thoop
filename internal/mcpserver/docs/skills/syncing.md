---
name: thoop/syncing
description: Load this when data looks stale, missing, pending, or recently changed.
related: thoop/local-data, thoop/metrics
---

# Syncing

thoop's cache service owns staleness decisions. MCP tools should not bypass it
with direct WHOOP client calls.

Recent `PENDING_SCORE` and current-day `SCORED` records may refresh. Historical
scored data changes less often. If a user asks for the latest data, trust the
tool metadata and mention when data came from cache versus a refresh path.

WHOOP score fields are only present for scored objects. A refresh can turn
`PENDING_SCORE` records into `SCORED` records, but `UNSCORABLE` commonly means
WHOOP did not have enough metric data for that time range.

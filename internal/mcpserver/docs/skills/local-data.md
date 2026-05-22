---
name: thoop/local-data
description: Load this when reasoning about SQLite cache freshness, local-first behavior, or sync boundaries.
related: thoop/syncing, thoop/privacy
---

# Local Data

The MCP server is local-first. It opens the same SQLite database used by thoop's
TUI and routes reads through the existing cache service.

Do not describe data as complete unless the result metadata indicates enough
coverage for the requested question. If a response says it was refreshed or is
partial, mention that in the answer.

List tools paginate after data is gathered from the cache service. Use the
metadata's next-page hint when a result is truncated.

Date questions should still be answered through cycle-aware tools. Calendar
dates are a user convenience, but WHOOP's own model is physiological cycles,
and sleep/workout records carry explicit start/end timestamps and timezone
offsets.

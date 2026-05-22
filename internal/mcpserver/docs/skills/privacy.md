---
name: thoop/privacy
description: Load this when explaining or auditing what data leaves the user's machine.
related: thoop/overview, thoop/local-data
---

# Privacy

For Claude and Codex local MCP usage, the MCP server runs on the user's machine.
WHOOP access and refresh tokens stay in the OS keyring. The server does not
upload credentials to thoop infrastructure for local MCP.

Network calls happen only through the existing cache service and WHOOP/proxy
client path when data is stale or missing. Local SQLite remains the primary
data source.

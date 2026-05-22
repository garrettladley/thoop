Return the latest local-first WHOOP summary.

This tool uses thoop's existing cache service. It checks local SQLite first and
refreshes through the existing keyring-backed WHOOP path when the cache service
decides data is stale or missing.

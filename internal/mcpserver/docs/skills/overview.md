---
name: thoop/overview
description: Load this first when using thoop MCP for general WHOOP data questions.
related: thoop/metrics, thoop/local-data, thoop/privacy
---

# Thoop Overview

thoop exposes local-first WHOOP data to MCP clients. The MCP server runs on the
user's machine over stdio and uses the same local SQLite database and OS keyring
credentials as the terminal UI.

Prefer narrow tools. Use single-date tools for direct questions, list tools for
trends, and load `thoop/metrics` before interpreting WHOOP domain terms.

Data tools use the existing thoop cache service. That means they read SQLite
first and only reach WHOOP through the established cache/staleness path.

WHOOP data is organized around three pillars: Strain, Recovery, and Sleep. The
core API objects exposed by thoop are Cycle, Sleep, Recovery, and Workout.
Cycle is the key boundary: WHOOP references a member's activity in the context
of a physiological cycle rather than assuming calendar days always match sleep
and wake patterns.

MCP output keeps raw WHOOP values and adds readable unit fields next to them
when units are easy to misread. Use raw values for exact calculations and the
readable fields for prose. Examples: millisecond durations also include `7h4m`
style fields, percentages also include `*_pct`, heart rates include `bpm`,
distances keep meters and add km/miles, energy keeps kilojoules and adds
kilocalories, and timezone offsets keep their original value while local
date/time fields make timestamps easier to interpret. Do not infer an IANA
timezone from an offset alone.

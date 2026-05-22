---
name: thoop/metrics
description: Load this before answering questions about recovery, sleep, strain, cycles, or workouts.
related: thoop/overview, thoop/local-data
---

# WHOOP Metric Idioms

## Core Objects

Cycle is WHOOP's physiological day. It represents a biological rhythm of asleep,
awake, asleep, awake, and it may not align cleanly to a calendar day. Current
cycles may have only a start time; past cycles usually have both start and end.
Cycle score contains day strain, kilojoules, average heart rate, and max heart
rate when `score_state` is `SCORED`.

Sleep is a sleep activity belonging to a cycle. The main sleep starts a cycle,
but a member can also have naps during the cycle. Naps have `nap=true` and can
reduce the sleep needed for the next cycle.

Recovery is cycle-based, not a generic calendar-day health score. It represents
how recovered the user is for a physiological cycle and is associated with a
sleep ID. Recovery score is a percentage from 0 to 100 when scored.

Workout is an activity record with sport name, start/end bounds, score state,
and a workout score when scored. Workout score includes workout strain, heart
rate summary, kilojoules, percent recorded, distance/elevation when available,
and heart-rate zone durations.

## Interpretation Rules

Cycle strain is part of cycle score. Workout strain is part of a workout score.
Do not add workout strain values to cycle strain unless the user explicitly asks
for workout sum analysis.

Strain is on a 0 to 21 scale and is not linear. Moving from 16 to 17 requires
more stress than moving from 4 to 5. WHOOP describes strain categories as:

- Light: 0-9
- Moderate: 10-13
- High: 14-17
- All Out: 18-21

Recovery colors are:

- Green: 67-100
- Yellow: 34-66
- Red: 0-33

Sleep records can cross midnight. Do not assume that sleep belongs only to the
calendar date where it ends. Use start, end, cycle ID, timezone offset, and nap
flag.

Score states matter:

- `SCORED` means WHOOP has produced a score.
- `PENDING_SCORE` can change soon and may refresh.
- `UNSCORABLE` should generally be treated as final missing score data.

Score objects are present only when the corresponding object is `SCORED`. If a
tool returns `PENDING_SCORE` or `UNSCORABLE`, do not infer score fields that are
missing.

Use list tools for trends and detail tools for a specific date or record.

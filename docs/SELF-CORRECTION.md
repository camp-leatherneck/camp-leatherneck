---
title: Self-Correction Loop
audience: LT, Top, any standing role that accumulates judgment calls over time
last-updated: 2026-04-22
author: Scribe
---

# The Self-Correction Loop

An agent that does not encode its mistakes will make the same mistake forever. Memory without curation rots into noise. The Self-Correction Loop is the discipline that turns "I was wrong" into "future sessions cannot be wrong this way again."

This doc is for anyone running LT, Top, or any persona with a durable directive. It is the overlay's answer to the question *how does Camp Leatherneck get better over time?*

## Why this matters

Joey runs on a Max 5× rate limit and finite trust. A session that repeats a false alarm twice damages that trust measurably. A session that silently encodes the correction so the next session starts smarter compounds the other direction. The loop below takes under two minutes and is the cheapest possible investment in not-being-stupid-again.

The mechanism is pure curation. No approval. No ceremony. You run it; you move on.

## Trigger

Run the loop whenever any of these fire:

- You alarmed about something that turned out to be fine
- You recommended an action that Joey or a subagent corrected
- A specialist's report disagreed with your prior assertion — and the specialist was right
- Verification (yours or anyone's) showed your hypothesis was wrong, even if you never said it out loud

The trigger fires on small misses, not just loud ones. Small patterns repeat; small corrections compound.

## The 5-step loop

Runs in under two minutes. No approval required.

**1. Name the pattern, not the incident.**
*"I assumed local state equals authoritative state"* is a pattern. *"I thought jcmd_website's MQ was broken"* is an incident. Incidents are forgettable; patterns are teachable. Future-you skims for rules, not stories.

**2. Route the correction to the right home.** One of:

| Scope of the correction | Home |
|---|---|
| This one task only | No persistence — self-correct and move on |
| Every future LT session | `directives/mayor.md` (primed every prime) |
| Every agent on the town | `CLAUDE.md` (primed by everyone) |
| One specialist's domain | That specialist's directive |
| Specific fact a future session should remember | `bd remember "<tagged insight>"` |

The split matters. Directives are primed on every session start — that is where rules belong. `bd remember` is searchable on demand — that is where lookup-only facts belong.

**3. Write the correction as a rule.** Include a brief *why* so the rule can be judged against edge cases. *"Don't confuse jcmd_website MQ with origin/main"* is too narrow and will rot. *"Verify authoritative source before declaring state broken — local signals lag"* generalizes and survives.

**4. Update the running list** when the miss fits an existing class. [`DIAGNOSTIC-DISCIPLINE.md`](./DIAGNOSTIC-DISCIPLINE.md) maintains a running list of false-alarm patterns caught. Every alarm you catch yourself in should feed it.

**5. Do not announce or perform contrition.** Corrections are silent curation. Joey sees the result in future behavior; he does not need to watch the meta. One acknowledgement, then encode and move on.

## Escalation: wrong once → wrong twice → wrong three times

The correction scales with repetition:

- **Wrong once.** Write a rule. That's the whole loop.
- **Wrong twice the same way.** The rule was missing or wrong. Tighten it — more specific trigger, sharper *why*, clearer example.
- **Wrong three times.** Stop trusting your first instinct on this class entirely. **Add a forced verification step** — a concrete command, check, or dispatch that must run before you are allowed to act on instinct. This moves the work from memory (easy to skip) to procedure (hard to skip).

The three-strike rule is load-bearing. It acknowledges that some classes of error are cognitive biases you will not think your way out of. Procedure beats willpower.

## Anti-patterns

Things that feel like self-correction but aren't:

- **Apologizing repeatedly.** One acknowledgement, then encode and move on. The ledger tracks trajectory, not snapshots.
- **Over-generalizing.** Don't turn one miss into sweeping new doctrine. Wrong once = one rule.
- **Silently skipping the loop.** *"I already know this won't happen again"* is the exact thought pattern that makes it happen again. Run the loop even when you're sure.
- **Moving a rule to memory when the real home is a directive.** Memory is searchable but not primed. For *"every session needs this context,"* directives win.
- **Encoding the incident into the rule.** Narrow rules tied to specific names, IDs, or dates rot fast. Write the general pattern.

## Closing signal

After the loop completes, the correction is done. Return to the task you were doing. Do not keep rehearsing the mistake. Stop narrating. Next tick.

## See also

- [`DIAGNOSTIC-DISCIPLINE.md`](./DIAGNOSTIC-DISCIPLINE.md) — a specific instance of Self-Correction, focused on anti-alarmism
- [`ITERATIVE-LEARNING.md`](./ITERATIVE-LEARNING.md) — how encoded corrections propagate across sessions
- [`PERSONAS.md`](./PERSONAS.md) — which personas run this loop (LT and Top especially)
- [`directives/mayor.md`](../directives/mayor.md) — the canonical source for the loop inside LT's directive

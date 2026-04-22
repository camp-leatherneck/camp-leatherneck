---
title: Iterative Learning
audience: Contributors modifying directives; operators wondering where state lives
last-updated: 2026-04-22
author: Scribe
---

# Iterative Learning

Camp Leatherneck grows itself. Every time a session catches a mistake, that correction goes somewhere durable so the next session starts smarter. This doc explains the two state-stores that hold those corrections and when to use which.

The mechanics of *how* corrections get captured live in [`SELF-CORRECTION.md`](./SELF-CORRECTION.md). This doc covers *where* they go.

## Two durable state-stores

The overlay has exactly two places that survive session death:

### 1. Directive files — `directives/*.md`

Primed on every session start. The agent reads them before doing anything else. Always-loaded context.

Use directives for:

- **Rules every session must follow.** Style, voice, scope limits, concurrency constraints, escalation paths.
- **Doctrine.** The Self-Correction Loop. Diagnostic Discipline. Restraint Doctrine.
- **Running lists the agent is expected to scan.** Diagnostic Discipline's false-alarm pattern list is in `mayor.md` for exactly this reason — it must be primed, not looked up.
- **Per-role voice and output format.** Scribe's tone. LT's "Chief of Staff voice." Doc's triage template.

Directives are load-bearing. They are read every time, which also means they compete for context budget. Keep them tight; prune entries that no longer earn their place.

### 2. `bd remember` memories — Dolt-backed, searchable

Not primed. Searchable on demand via `bd remember-search` and friends. Persistent across sessions.

Use `bd remember` for:

- **Lookupable facts.** "Client X's deploy window is Tuesday mornings." "The staging DB password rotates on the 1st."
- **Decisions with a `why`.** ADR-flavored entries a future session might want to re-read when making an adjacent call.
- **Snapshots flagged for re-validation.** Recon findings with a `last-reviewed:` date; go-stale-fast entries the agent should re-check before acting on.
- **Cross-session continuity** for anything too narrow to justify directive space.

Memories are cheap to add and cheap to skip. A directive's value is everyone reads it; a memory's value is it exists when someone searches for it.

## When to use which

Ask: *does every session need this context, or only a session that happens to look for it?*

| Characteristic | Directive | `bd remember` |
|---|---|---|
| Applies every session | Yes | No |
| Specific fact vs general rule | General rule | Specific fact |
| Changes behavior without being searched | Yes | No |
| Size budget | Tight | Loose |
| Decay / staleness risk | Low (you prune on edit) | Medium (add `last-reviewed`) |

The default for a correction from the [Self-Correction Loop](./SELF-CORRECTION.md) is *directive*, not memory. Memory is the fallback for specific facts that don't justify everyone loading them.

## How a new LT session inherits prior corrections

The inheritance chain is automatic:

1. A prior LT session runs the Self-Correction Loop, routes the correction to `directives/mayor.md` (or `CLAUDE.md` for town-wide rules).
2. The edit lands in the repo; `git push` ships it.
3. A new LT session starts. `gt prime` runs. Priming reads `CLAUDE.md`, then the appropriate directive, then any relevant context files.
4. The correction is now in the new session's context. It is primed, not searched. The new session will not repeat the mistake — not because it's been told about it, but because the rule is already loaded before its first reasoning step.

No ceremony. No memory handoff doc. The directive itself is the handoff.

For corrections routed to `bd remember`: the new session does not auto-load them. It finds them when it searches — e.g., when triaging a bead about Client X, a search for "Client X" surfaces the memory. Memories support a pull model, directives support a push model.

## What this means for contributors

When you make the overlay better:

- **Rule that changes how every session behaves?** Edit the directive. Commit. Push. Done.
- **Fact a session might need to look up?** `bd remember`. No directive edit.
- **Both?** Edit the directive with the rule; add the specific incident to `bd remember` only if the incident itself has reusable detail beyond the rule.

Keep directives short. Keep memories searchable. Corrections compound either way.

## See also

- [`SELF-CORRECTION.md`](./SELF-CORRECTION.md) — the loop that produces the corrections this doc stores
- [`DIAGNOSTIC-DISCIPLINE.md`](./DIAGNOSTIC-DISCIPLINE.md) — a running list maintained inside a directive, example of the "primed" pattern
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — where directives sit in the fork
- Upstream [`README.md`](../README.md) — `bd` command reference

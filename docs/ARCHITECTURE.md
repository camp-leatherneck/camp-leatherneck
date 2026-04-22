---
title: Camp Leatherneck Architecture
audience: New contributors, operators, and anyone auditing the fork boundary
last-updated: 2026-04-22
author: Scribe
---

# Camp Leatherneck Architecture

Camp Leatherneck is a **layered fork** of [Gas Town](https://github.com/steveyegge/gastown). The engine is Gas Town, unchanged. Camp Leatherneck sits on top as a persona and doctrine overlay. This doc explains the seam so contributors know which side of it a given change belongs on.

For engine internals (rig lifecycle, bead schema, merge queue, Dolt wiring, tmux session model), read the upstream Gas Town [`README.md`](../README.md) and [`docs/overview.md`](./overview.md). Do not re-document them here.

## The Fork-C model

We use a **Fork-C layered fork**. The rules:

- User-facing surface — binary name, display strings, prime banners, default status-left, role vocabulary — is ours to rename.
- Internal Go identifiers, CLI role-slot names (`mayor`, `deacon`, `refinery`, `witness`, `boot`), package paths, and protocol formats are Gas Town's and stay unchanged.
- This keeps `git merge upstream/main` almost always trivial. Upstream conflicts, when they happen, live in `README.md`, `templates/`, and embedded directive files — not in engine code.

The fork is on GitHub at `camp-leatherneck/camp-leatherneck`. Upstream is `steveyegge/gastown`.

## What Camp Leatherneck inherits from Gas Town

Everything that makes the system work:

- **Rigs** — per-project workspaces with isolated worktrees.
- **Polecats** — standing per-rig agents with persistent identity but ephemeral sessions.
- **Beads** — the `bd` issue tracker and the primary work ledger.
- **Merge queue (MQ)** — Bors-style serialization of polecat branches into `origin/main`.
- **Dolt** — the single-server data plane for beads, mail, and identity (port 3307).
- **tmux orchestration** — every agent runs in a tmux session; lifecycle management via `gt` commands.
- **Role daemons** — `mayor`, `deacon`, `refinery`, `witness`, `boot` daemons, each with a directive and a hook-driven wake cadence.
- **`gt` CLI** — the entire command surface for rig/polecat/bead/mail/convoy management.

If you want to understand how any of that works, read upstream. We did not touch it.

## What Camp Leatherneck adds

The overlay:

- **Marine persona layer** — 23 personas mapped onto Gas Town's role slots. See [`PERSONAS.md`](./PERSONAS.md). `gt mayor` routes to the **LT** persona; `gt deacon` routes to **Top**; `gt refinery` routes to **Gunny**; etc.
- **Two new standing roles** — **Fire Watch** (Pvt, keeps Top alive) and **RTO** (Sgt, maintains `~/Desktop/sitrep.md`). Not present in stock Gas Town.
- **Self-Correction Loop** — the meta-discipline for encoding LT's mistakes into durable rules. See [`SELF-CORRECTION.md`](./SELF-CORRECTION.md).
- **Diagnostic Discipline** — four rules that counter premature-alarm reflexes. See [`DIAGNOSTIC-DISCIPLINE.md`](./DIAGNOSTIC-DISCIPLINE.md).
- **Sitrep generator** — a launchd-scheduled synthesizer that fuses `gt feed`, `bd ready`, mail, convoy, and MQ into a <30-second read.
- **Installer** — `brew install camp-leatherneck/tap/lt` (planned for v0.1.0), plus an `lt migrate` subcommand for existing `~/gt/` installs. See [`MIGRATION.md`](./MIGRATION.md).

## Layer diagram

```
+------------------------------------------------------+
|  User-facing brand                                   |
|    LT, Top, Gunny, Sarge, Fire Watch, RTO            |
|    Marine display strings, prime banners, sitrep     |
|    `lt` binary (alias-first), Sitrep.app bundle      |
+------------------------------------------------------+
|  Camp Leatherneck overlay                            |
|    directives/*.md  (23 personas)                    |
|    Self-Correction Loop + Diagnostic Discipline      |
|    RTO cron + Fire Watch liveness                    |
+------------------------------------------------------+
|  Gas Town engine  (github.com/steveyegge/gastown)    |
|    rigs, polecats, beads, MQ, Dolt, tmux             |
|    `gt` CLI, role daemons, hook system               |
|    internal/cmd/*.go, Go module path unchanged       |
+------------------------------------------------------+
```

Upstream sits beneath us. The overlay never reaches inside the engine — it only reaches above, through directive files, display-string configuration, and new CLI subcommands that call the engine via its public surface.

## Upstream sync policy

We track `github.com/steveyegge/gastown` and pull improvements. Year-1 cadence: monthly minimum; opportunistic when a notable upstream feature lands.

Workflow:

1. `git fetch upstream && git merge upstream/main`
2. Resolve conflicts — expected only in `README.md`, `NOTICE`, embedded directives, and any display-string overrides.
3. Rebuild, smoke-test against a scratch rig.
4. Cut a Camp Leatherneck patch release if the merge is material.

If upstream ships a feature that duplicates something we built in the overlay (e.g., their own sitrep equivalent), we prefer upstream and retire our version. Keep the overlay small.

## See also

- [`PERSONAS.md`](./PERSONAS.md) — full persona roster and tier model
- [`SELF-CORRECTION.md`](./SELF-CORRECTION.md) — the meta-loop for being wrong
- [`DIAGNOSTIC-DISCIPLINE.md`](./DIAGNOSTIC-DISCIPLINE.md) — anti-alarmism rules
- [`ITERATIVE-LEARNING.md`](./ITERATIVE-LEARNING.md) — how the overlay grows itself
- [`MIGRATION.md`](./MIGRATION.md) — moving from `~/gt/` to Camp Leatherneck
- Upstream [`README.md`](../README.md) — Gas Town engine overview
- [`NOTICE`](../NOTICE) — attribution

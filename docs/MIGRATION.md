---
title: Migration from Gas Town to Camp Leatherneck
audience: Existing Gas Town users evaluating the fork
last-updated: 2026-04-22
author: Scribe
status: v0.1.0 (in progress) — design spec + placeholder
---

# Migration

> **Status: v0.1.0 (in progress).** The `lt migrate` subcommand described here does not yet exist as of `2026-04-22`. This doc is a design spec and a placeholder so you know what the migration contract will be. Until the subcommand ships, migration is manual — see the "Manual path" section at the bottom.

Camp Leatherneck is a drop-in replacement for a stock Gas Town install. Migration is designed to be **additive and reversible**: no data format changes, no schema migrations, no bead loss. You keep your state; you gain the persona overlay.

## What migration preserves

Untouched by migration:

- **Bead data.** The `bd` schema is Gas Town's. Your open beads, closed history, mail, memories, and identity all stay.
- **Dolt database.** `~/.dolt-data/` is not moved or rewritten. Port 3307 keeps serving the same rows.
- **Polecat history.** CVs, worktrees, ledgers, branches, per-polecat state on each rig.
- **Directive customizations.** If you've hand-edited `~/gt/directives/*.md`, migration offers to merge your edits with the new Camp Leatherneck directives (three-way merge; conflicts go to you).
- **Rig configuration.** `~/gt/rigs/` stays where it is. Rig topology is unchanged.
- **Installed hooks, launchd plists, cron jobs** — the migration inspects them and warns on anything that references the stock `gt` binary path.

## What migration adds

New after migration:

- **The 23-persona overlay.** LT, Top, Gunny, Sarge replace Mayor, Deacon, Refinery, Witness in display strings and prime banners.
- **Fire Watch** — standing role (`boot` slot), keeps Top alive. Installed as a launchd agent; starts automatically.
- **RTO** — standing role, cron + event-driven sitrep generator. Writes `~/Desktop/sitrep.md`. Installed as a launchd agent.
- **Marine display strings.** Status-left defaults, prime banners, `gt` help text surface the new vocabulary. Internal Go identifiers and CLI role-slot names (`mayor`, `deacon`, etc.) are unchanged — scripts continue to work.
- **Sitrep.app bundle.** Native macOS app wrapper around the sitrep file, optional.
- **`lt` binary alias.** `lt` invokes the Camp Leatherneck-built binary; `gt` stays as a compatibility alias so your muscle memory and existing scripts don't break.

## The `lt migrate` contract (v0.1.0 target)

```bash
lt migrate                    # idempotent; safe to re-run
lt migrate --dry-run          # preview actions, no changes
lt migrate --source ~/gt      # explicit source (default: ~/gt)
lt migrate --rollback         # see "Rollback" below
```

Behavior:

1. **Preflight.** Check for running `gt` daemons (mayor, deacon, witness, refinery, boot). Refuse to migrate with live daemons; ask the user to `gt down` first. Capture a Dolt goroutine dump + `gt dolt status` for rollback safety.
2. **Snapshot.** `git -C ~/gt status` clean check. Tag current state. Cut a Dolt backup to `~/.dolt-data/backups/pre-camp-leatherneck-<timestamp>/`.
3. **Install.** Lay down new directives under `~/gt/directives/` with three-way merge for any user-edited files. Install Fire Watch and RTO launchd plists. Install Marine display-string overrides. Install the `lt` binary and keep `gt` as an alias.
4. **Verify.** Start daemons. Confirm bead counts, polecat lists, MQ state match pre-migration. Confirm sitrep writes. Confirm Fire Watch wakes.
5. **Report.** Summary of what changed and what stayed. Location of backup. Rollback command.

Migration leaves `~/gt/` in place. Camp Leatherneck reads and writes the same directory tree.

## Rollback

Yes. You can go back.

The bead schema and Dolt format are **unchanged**. That's the whole point of the layered fork. To roll back:

```bash
lt migrate --rollback         # (target for v0.1.0 — restores directives + uninstalls overlay daemons)
```

Or, if the subcommand isn't yet available:

1. `lt down` (or `gt down`) — stop all daemons.
2. Uninstall Camp Leatherneck's launchd agents (Fire Watch, RTO).
3. Reinstall stock Gas Town: `brew install gastown` (or your previous install path).
4. Restore the pre-migration directive files from the backup snapshot.
5. `gt up`. Point it at the same `~/gt/`. Your beads, Dolt DB, polecat history are all still there and readable — stock Gas Town opens them unchanged.

There is no data lock-in. The worst case is a few minutes of uninstall-and-reinstall, and you are back on stock Gas Town with full history intact.

## Manual path (pre-v0.1.0)

Until `lt migrate` ships:

1. `git clone https://github.com/camp-leatherneck/camp-leatherneck ~/camp-leatherneck`
2. Stop Gas Town daemons: `gt down`.
3. Diff `~/camp-leatherneck/directives/` against `~/gt/directives/`. Copy over the Camp Leatherneck directives you want (at minimum: `mayor.md`, `deacon.md`, `rto.md`, and any new specialist directives). Preserve any custom edits you've made.
4. Rebuild and install: `cd ~/camp-leatherneck && make install`.
5. Start daemons.

This is manual enough that most users will want to wait for `lt migrate`. Track the v0.1.0 release for the automated path.

## See also

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — why migration is safe (the fork seam)
- [`PERSONAS.md`](./PERSONAS.md) — what you gain (the 23-persona roster)
- [`README.md`](../README.md) — install overview, current status
- [`NOTICE`](../NOTICE) — attribution; your Gas Town install's attribution is unchanged after migration
- Upstream [Gas Town](https://github.com/steveyegge/gastown) — where you came from; where you can go back to

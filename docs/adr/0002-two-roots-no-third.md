# ADR 0002: Two roots — `~/camp-leatherneck` is source, `~/gt` is runtime; no third

**Status:** Accepted
**Date:** 2026-07-31

## Context

Verified 2026-07-31: three directories held directives, scripts, or build
artifacts simultaneously. `~/camp-leatherneck` (git remotes `origin` +
`upstream`) was the real, actively-developed source. `~/gt` (local-only git
history, no remote) was the real, live runtime town —
directives, `settings/config.json`, `rigs.json`, plugins, beads, daemon,
logs. `~/lt` was the frozen residue of an April 22 rename: its
`directives/` was byte-identical to `~/gt/directives/` on every overlapping
file, carrying only the five pre-rename filenames and nothing unique.
Despite holding nothing unique, `~/lt` was live: `com.campleatherneck.rto.plist`
pointed its RTO job at `~/lt/scripts/rto.sh`, which wrote to
`~/Desktop/sitrep.md` — a macOS TCC-blocked destination for launchd-spawned
processes — and had been failing silently for 99 days (`rto.err` grew to
4.4MB, ~71,000 failed attempts) because nothing asserted that RTO should be
reading from `~/gt`.

## Decision

There are exactly two durable roots. `~/camp-leatherneck` is the sole
version-controlled source. `~/gt` is the sole runtime town — the path name
is plumbing, not identity, and is not renamed (see ADR 0004's naming
boundary — an operator types `lt`, not `cd ~/gt`). Any third directory
holding directives, scripts, or roster data is drift, not architecture, and
gets archived then removed, not documented into legitimacy.

`~/lt` was archived to a dated tarball outside both roots
(`~/lt-archived-2026-07-31.tar.gz`, independently verified byte-for-byte
against the live directory before deletion) and deleted 2026-07-31 after
confirming zero live references remained anywhere on the system.

## Consequences

- `lt doctor`'s provenance check (ADR 0005) treats a third root the same
  way it treats any other provenance violation: a hard failure, not a
  warning.
- Nothing may write runtime output to `~/Desktop` — the direct cause of the
  99-day RTO failure. RTO writes to `$HOME/gt/sitrep.md`.
- If a similar rename ever happens again, the old path must be repointed or
  removed in the same change that introduces the new one — not left as a
  live parallel root for launchd, scripts, or directives to drift onto.

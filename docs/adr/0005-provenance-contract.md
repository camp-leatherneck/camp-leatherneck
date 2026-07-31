# ADR 0005: Provenance contract enforced by `lt doctor`

**Status:** Accepted
**Date:** 2026-07-31

## Context

Two failures persisted, silently, for months, for the same underlying
reason: nothing asserted that the system's declared state matched its
actual state. RTO wrote to a TCC-blocked path and failed roughly every 120
seconds for 99 days without anyone noticing, because nothing checked
whether `~/gt/sitrep.md` was actually being produced. The daemon's launchd
plist declared `/Users/joeydeleon/camp-leatherneck/lt` (a gitignored build
artifact) while the actually-running process was `/Users/joeydeleon/.local/bin/lt`
— byte-identical *by coincidence*, because the plist had simply never been
corrected, which is not a control.

Two pre-existing facts shaped how the fix was scoped: `lt doctor` was not
new work — it already existed, alongside three other overlapping health
surfaces (`status`, `vitals`, `health`). And `internal/cmd/doctor.go` was
itself a near-upstream file (`upstream/main` carries independent fixes to
it) — heavily extending it in place would generate recurring merge
conflicts on a file upstream actively maintains, violating the same
upstream-compatibility principle ADR 0001 and 0004 depend on.

## Decision

The provenance identity that must hold: **launchd's declared program ≡ the
running daemon's executable ≡ the `lt` on `PATH` ≡ a known commit in
`~/camp-leatherneck`.** Any mismatch is a hard failure surfaced by
`lt doctor`, not a warning buried in a log — silent failure is a defect
regardless of what it improves.

The logic lives entirely in a new product-layer package,
`internal/provenance/`, covering every dimension: binary (PATH resolution,
real path, sha256, embedded version/commit/build-time), shadowing (any
`lt`/`gt` elsewhere on `PATH`, flagged only when its content differs from
canonical — the exact check that would have caught the Homebrew `gastown`
hazard in ADR 0003 before it became a live incident), source (commit
ancestry and working-tree cleanliness against the actual repo), upstream
(remote-tracking ref freshness for the source repo's configured upstream
remote, added by ADR 0006 — never fetches, and never reports a
commit-distance figure against a ref whose freshness is unknown or expired),
daemon (running, executable matches PATH binary), launchd (every
`com.campleatherneck.*` job, declared program exists, loaded state),
directives (every role resolves per ADR 0004, with `dog`/`crew` correctly
treated as directive-less by design rather than flagged missing), and
roster (flags any stale standalone roster file). `internal/cmd/doctor.go`
is touched by exactly one import and one `Register()` call — no fifth
health command was created; provenance surfaces through the existing
`doctor`.

The check was proven, not just written: run clean against the live system,
then a required directive was deliberately removed for the duration of one
`lt doctor` run, confirmed to flip the check from green to a hard failure,
then restored and reconfirmed green. A guard that has never failed has
never been tested.

## Consequences

- Any future provenance-relevant addition (a new launchd job, a new role,
  a new binary alias) should extend `internal/provenance`, not create a
  fifth health command or a second lookup path. ADR 0006's upstream
  remote-reference freshness dimension is this pattern already exercised:
  zero further edits to `doctor.go`, the whole check added inside
  `internal/provenance`.
- `status`/`vitals`/`health` remain operational dashboards and are not
  extended for provenance reporting — `doctor` is the diagnostic surface.
- The RTO and daemon-plist failures this ADR responds to are structurally
  prevented from recurring silently: both are now provenance-check
  dimensions, not tribal knowledge.

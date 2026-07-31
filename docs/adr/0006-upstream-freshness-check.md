# ADR 0006: Upstream remote-reference freshness in `lt doctor`

**Status:** Accepted
**Date:** 2026-07-31

## Context

The 2026-07-31 Amendment to `CAMP_LEATHERNECK_ARCHITECTURE_CERTIFICATION.md`
(not tracked in this repo — see
`ai-powerhouse/projects/camp-leatherneck/architecture/`) found that a
governance certification had reported
`git rev-list --count HEAD..upstream/main` = 0 against a local
`upstream/main` remote-tracking ref with no confirmed fresh fetch in that
session. That measurement is fail-open: a stale ref silently under-reports
how far behind the fork actually is, and reads identically to genuine
parity. A same-day, timestamp-verified re-measurement showed main **777
commits and 1,177 files** behind. The false "0 commits behind" figure had
already been repeated as settled fact across five separate governance
documents (`docs/adr/0001-productized-downstream-distribution.md`,
`CONTRIBUTING.md`, and three non-tracked `architecture/` docs).

That Amendment recorded four governance gaps rather than silently patching
them. Gap 1, quoted directly: **"`lt doctor` does not verify remote-reference
freshness."** Confirmed by reading `internal/cmd/doctor.go` and
`internal/provenance/provenance.go` — the seven existing provenance
dimensions (Binary, Shadowing, Source, Daemon, Launchd, Directives, Roster)
contained no "Upstream" dimension at all. A fork could drift 777 commits
with `lt doctor` reporting fully green throughout. Gap 2 restated this as
policy: **any future check must fail closed (unknown freshness ≠
assumed-fresh) or it reproduces exactly this defect in code instead of in a
document.**

Two constraints from ADR 0005 (Provenance contract) carry forward directly:
`internal/cmd/doctor.go` is a near-upstream file — any extension belongs in
`internal/provenance/`, touching `doctor.go` with at most one import and one
`Register()` call. That call already exists (`provenance.NewCheck()`,
registered once, in the doctor.go position that predates this ADR) — this
change requires **zero** further edits to `doctor.go`, since it extends the
same `Report`/`Gather`/`Violations`/`Warnings` machinery ADR 0005 already
wired in.

A second constraint is new to this ADR: **no existing doctor check performs
network I/O**, in an ordinary run or under `--fix` (confirmed by reading
every `internal/doctor/*.go` Fix implementation). Silently making `git
fetch` part of an ordinary `lt doctor` run would violate that established
contract and make a diagnostic command mutate repository state and depend
on network reachability — the opposite of what a doctor check should do,
and a regression for offline use.

## Decision

Add an eighth provenance dimension, `Upstream` (`internal/provenance/upstream.go`),
answering: **how current is our local knowledge of the configured upstream
remote's default branch, and is that knowledge trustworthy enough to
support a commit-distance or parity claim?**

`GatherUpstream` never fetches. It only reads evidence already on disk:

1. Is the remote (`upstream`, configurable via `UpstreamRemoteName`)
   configured at all?
2. Does the remote-tracking ref (`upstream/main`, configurable via
   `UpstreamBranchName`) exist locally?
3. When was it last written? The reflog
   (`.git/logs/refs/remotes/upstream/main`) is preferred — every entry
   carries an explicit committer timestamp, appended on every fetch that
   moves or confirms the ref. If no reflog exists (`core.logAllRefUpdates`
   off, or predating that setting), the loose ref file's mtime is used as a
   weaker but still repository-native fallback. If neither exists, freshness
   is reported as unknown — never silently assumed fresh.

This produces one of six explicit states (`UpstreamFreshness`):
`no-remote`, `missing-branch`, `never-fetched`, `stale`, `fresh`, and
`fetch-failed` (reachable only through the explicit, opt-in
`FetchUpstreamAndRecheck` helper below — never through the passive,
doctor-driven `GatherUpstream`).

**The commit-distance figure is the control itself, not a side effect.**
`CommitsBehind` is populated *only* when `Freshness == fresh`. For every
other state it stays `-1` — a distance number against a ref of unverified
or expired currency is exactly the unsupported claim this ADR exists to
prevent, so the code simply never produces one, rather than producing it
and hoping callers remember to check freshness first.

**Threshold, grounded not invented:** `DefaultUpstreamStaleAfter` is 30
days, taken directly from `CONTRIBUTING.md`'s existing "Camp Leatherneck:
Upstream Merge Policy" section, which already commits this project to a
monthly-minimum sync cadence. A ref fetched anywhere within that required
window reads as fresh; one left untouched longer than the policy's own
stated minimum reads as stale. This is a package-level var
(`UpstreamStaleAfter`), overridable — if the governance cadence changes,
this default should change with it, and a test or operator can override it
without recompiling.

**Non-mutating by design, with an explicit escape hatch:** `lt doctor`
calls `GatherUpstream` only, so an ordinary run — including `--fix` — never
touches the network and works fully offline, preserving the existing
contract. `never-fetched` and `stale` surface as `doctor.StatusWarning`
(not `StatusError`): a fork that has merely missed its cadence, or is being
inspected offline, is not broken the way a dead daemon is, and treating it
as a hard failure would punish normal offline use and turn a routine
out-of-cadence state into a build-breaking one. What must never happen —
the actual defect — is silently reading it as parity; that is closed by
never emitting `CommitsBehind` for a non-fresh ref, not by making `lt
doctor` fail loudly on every merely-out-of-cadence clone.

A separate function, `FetchUpstreamAndRecheck`, is the one function in this
package that touches the network — and only when a caller explicitly
invokes it (not wired into `lt doctor` or any other command by this
change). It runs `git fetch <remote> <branch>`, then re-derives
`UpstreamInfo`; a failed fetch reports `fetch-failed` with the underlying
error, never falling back to reporting the stale pre-fetch state as if it
were current. Every non-fresh `UpstreamInfo` also carries a
`RemediationCommand()` — the literal `git fetch` command to run — surfaced
as part of the doctor warning message, so a human has an explicit
fetch-then-recheck path without `lt doctor` performing it silently.

Every `UpstreamInfo` records `ObservedAt` (when the check ran) and
`RemoteName`/`Branch`/`RefCommit` (which ref, resolved to which commit) —
the evidence and as-of time the certification Amendment's gap 4 called for,
so a reported state is never presented without knowing when and against
what it was observed.

## Consequences

- `internal/cmd/doctor.go` is unchanged by this ADR — the existing
  `provenance.NewCheck()` registration already covers this dimension,
  exactly as ADR 0005 intended future provenance extensions to work.
- Gap 1 from the certification Amendment is closed: `internal/provenance/upstream.go`'s
  `GatherUpstream` plus the eight-dimension `Report`/`Violations`/`Warnings`
  machinery mean a stale, never-fetched, missing, or otherwise unverifiable
  `upstream/main` ref can no longer be misread as parity — `lt doctor` now
  surfaces exactly which of those states applies, with its evidence and
  as-of time. This is a freshness-of-measurement guarantee, not a
  drift-amount guarantee: a ref fetched often enough to stay `fresh`
  reports no warning regardless of how many commits behind it actually is.
  `lt doctor` does not detect or bound upstream drift in general — once a
  gap is known to exist, closing it is `hq-35iwf`'s job, not this check's.
- Gap 2 is closed as policy and as code: the check fails closed by
  construction (`CommitsBehind` is `-1` unless `Freshness == fresh`), not by
  convention callers have to remember.
- Gaps 3 (sync cadence ownership — `hq-35iwf`) and the actual 777-commit
  sync are explicitly **not** addressed here; this ADR is scoped to the
  detection control only, per the Amendment's own framing of these as four
  separate, independently trackable gaps.
- Any future change to the upstream sync cadence policy should update
  `DefaultUpstreamStaleAfter`'s justification here, not just its value.

# ADR 0003: One build, one CLI — `lt` canonical, `gt` a generated deprecation shim

**Status:** Accepted
**Date:** 2026-07-31

## Context

Verified 2026-07-31: `gt` on the reference machine was not a Camp Leatherneck
artifact at all. `/opt/homebrew/bin/gt` resolved to Homebrew formula
`gastown 1.1.0` — an independently installed upstream product, built
2026-05-06 (twelve weeks stale relative to this repo), with a different
sha256 than `lt`. It was not a compatibility alias; it was a second,
foreign product mutating the same town root at `~/gt`. Live plugins called
`gt rig list --json`, `gt escalate`, `gt mail send`, `gt rig show`,
`gt stale --json` — all routed to the stale upstream binary. A session
status-log entry from the same day recorded a boot-triage cycle "fixing" a
missing `gt` symlink via `brew link --overwrite gastown`, which actually
*restored* the foreign binary — two owners contending for one name, and the
wrong one kept winning.

## Decision

`lt` is the canonical CLI; there is exactly one. Homebrew's `gastown`
package is uninstalled — it is not a compatibility path, it is a competing
installation. `gt` becomes a **generated deprecation shim on the same
binary**: `make install` creates it as a symlink alongside `lt`, and the
binary detects `argv[0] == "gt"` at runtime, emits a one-line deprecation
notice to **stderr only** (plugins parse `--json` on stdout and must never
see it), then executes normally. `make install` refuses to leave a
shadowing `lt` or `gt` anywhere else on `PATH` — the existing shadow-nuking
hygiene (already covering `~/go/bin`, `~/bin`) was extended to
`/opt/homebrew/bin`, warning rather than silently deleting a Homebrew-owned
file there, since removal of a Homebrew formula belongs to `brew uninstall`.

The non-negotiable invariant: every executable named `gt` or `lt` reachable
on `PATH` must come from the same build and report the same commit via
`lt version`/`gt version --json`. Two behaviorally different binaries
answering to these names is a defect of the highest severity, because it
silently corrupts shared runtime state.

Sequencing matters here specifically: the shim was built, installed, and
verified working (`gt rig list --json | jq .` parses cleanly) *before* the
Homebrew uninstall, not after — the shim is the precondition that makes the
uninstall safe, not a convenience added afterward. This closed a
self-contradiction between two draft planning documents that would
otherwise have left an interval where unattended patrol cycles calling
`gt` found nothing.

## Consequences

- `gt` is a deprecation ramp with an end date, not a permanently supported
  second CLI — retired once call sites migrate to `lt` and `lt doctor`
  reports zero `gt` invocations for a sustained period.
- `lt version --json` (and by extension `lt doctor`) reports commit, dirty
  flag, build time, install path, and sha256 for exactly this reason: so a
  binary-identity mismatch is a command's output, not something inferred
  from symptoms.
- Reinstalling Homebrew's `gastown` package on this machine again would
  recreate this exact hazard; `lt doctor`'s shadowing check is the ongoing
  guard against it.

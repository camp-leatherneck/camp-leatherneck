# ADR 0001: Camp Leatherneck is a productized downstream distribution of Gastown

**Status:** Accepted
**Date:** 2026-07-31

## Context

Camp Leatherneck's relationship to Gas Town (`steveyegge/gastown`) had never been
formally decided. Three models were live candidates: a themed overlay (Gastown
remains the product, Camp Leatherneck is cosmetic), an independent maintained
fork (full terminology and identity replacement, upstream compatibility
abandoned), or a productized downstream distribution (Gastown is the engine,
Camp Leatherneck is a deliberately divergent product built on it).

Verified 2026-07-31: `git rev-list --count HEAD..upstream/main` = 0 (full
parity with upstream), `upstream/main..HEAD` = 45 (45 commits of product
divergence, including PHI containment and routing policy upstream doesn't
have). The CLI is renamed at the Makefile level (`BINARY := lt`), not
aliased. A dedicated Homebrew tap, independent `CHANGELOG.md`/`README.md`,
an install script, and an npm package directory already exist.

> **Correction, 2026-07-31 (later same day).** The `HEAD..upstream/main` = 0
> figure above was measured against a local `upstream/main` ref with no
> confirmed fresh fetch in that session — a fail-open method, and the result
> was false. A fresh, timestamped fetch (`2026-07-31T21:47:10Z`, cross-checked
> live via `git ls-remote upstream main`) showed main 777 commits and 1,177
> files behind. This does not change the Decision below: the relevant
> property for a deliberately divergent downstream distribution is
> **demonstrated mergeability**, not commit-distance parity, and mergeability
> was separately verified the same session by scratch merge (138 of 1,177
> differing files conflicted, all small and localized, rename detection
> intact, `main` unchanged afterward). See
> `CAMP_LEATHERNECK_ARCHITECTURE_CERTIFICATION.md`'s Amendment for full
> evidence, and `hq-35iwf` (bd) for the tracked workstream to actually close
> the 777-commit gap — a separate effort from this decision.

## Decision

Camp Leatherneck is a **productized downstream distribution of Gastown**:
Gastown is the engine (polecats, beads, molecules, hooks, mail, worktrees,
merge flow, dogs, patrols, rigs, daemon, Dolt) and is tracked, not owned.
Camp Leatherneck is the product layer (identity, CLI name, personas, ranks,
directives, doctrine, routing policy, governance, PHI containment, operator
experience) and is deliberately divergent.

The themed-overlay model was rejected as already factually false — it would
require reverting real product work. The independent-fork model was rejected
because the cost (abandoning free upstream engine maintenance, permanent
merge tax from internal renames) is real and permanent while the benefit
(no translation seam between operator-facing names and internal Go package
names) is aesthetic and invisible to the non-technical operator this system
serves.

## Consequences

- Upstream (`steveyegge/gastown`) is tracked on a regular cadence (recommend
  monthly), not treated as a one-time fork point.
- Internal Gastown-native identifiers (`internal/polecat`, `GT_TOWN_ROOT`,
  role IDs `mayor`/`deacon`/`witness`/`refinery`/`polecat`) are permanent and
  never renamed for terminology completeness — see ADR 0004.
- New engine-layer features should be proposed upstream first where
  reasonable, rather than reimplemented in the product layer.
- This decision is revisited only by a subsequent ADR, not by informal
  discussion — see `CAMP_LEATHERNECK_ARCHITECTURE_CONSTITUTION.md` §1.

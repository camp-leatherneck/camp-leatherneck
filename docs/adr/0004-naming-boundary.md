# ADR 0004: Naming boundary — internal role IDs stay Gastown, operator surfaces are USMC

**Status:** Accepted
**Date:** 2026-07-31

## Context

A governance audit had flagged uncertainty over whether the `polecat.md` →
`marine.md` directive rename (part of the earlier USMC rebrand) broke
worker directive lookup. Verified 2026-07-31: `internal/config/directives.go`'s
`roleToFileName()` maps stable internal role identifiers (`mayor`,
`deacon`, `witness`, `refinery`, `polecat`) to USMC directive filenames
(`lt`, `top`, `sarge`, `gunny`, `marine`). All five renamed targets were
present in `~/gt/directives/`; all five legacy names were absent — a clean
5-for-5 rename with no orphans. The rename did not break lookup; the scare
was the seam working correctly, not a defect.

More broadly: the honest cost of keeping Gastown-native internal names
while presenting a USMC product surface is a permanent translation seam —
operators see "LT" while stack traces say "mayor". The alternative
(renaming internals to match) would impose a permanent merge tax on every
future upstream sync, because git must re-detect renames across the whole
tree on each merge, for a confusion that has never actually cost anything —
the operator this system serves is explicitly non-technical and reads
`top.md`, not `internal/deacon/`.

## Decision

**The test:** would an operator ever see this string? Yes → Camp Leatherneck
/ USMC naming. No → leave it Gastown-native.

| Layer | Naming |
|---|---|
| Product name, docs, CLI binary (`lt`), operator-facing output, directive filenames, persona/rank names | USMC / Camp Leatherneck |
| Internal role identifiers, Go package names, environment variables (`GT_TOWN_ROOT`, `GT_RIG`), database/table names, helper binaries (`gt-proxy-*`) | Gastown-native, unchanged |

`roleToFileName()` is the **only** place role-to-file translation may
occur — no second lookup table, no per-call-site special casing, no
duplicate directive files under both old and new names. `lt doctor`
validates, for every known role, that `roleToFileName(role) + ".md"`
resolves at town or rig level, and fails loudly if not — the class of
failure the original audit feared is now detectable by command, not
archaeology.

## Consequences

- No future "let's rename the internals to match" proposal is adopted
  without a new ADR — this decision is final until upstream mergeability
  materially changes.
- Directive front-matter (ADR-adjacent, see the roster work in
  `internal/roster`) carries persona metadata (rank, scope, cardinality)
  alongside the USMC filename, without touching the internal role ID.
- `~/gt` itself is not renamed to `~/lt` or anything else — the path is
  plumbing (`GT_TOWN_ROOT`, the daemon plist, every rig worktree path,
  roughly a dozen existing Claude session histories reference it), not
  identity. The operator types `lt`, not `cd ~/gt`.

# Contributing to Gas Town

Thanks for your interest in contributing! Gas Town is experimental software, and we welcome contributions that help explore these ideas.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Install prerequisites (see README.md)
4. Build and test: `go build -o gt ./cmd/gt && go test ./...`

## Development Workflow

We use a direct-to-main workflow for trusted contributors. For external contributors:

1. Create a feature branch from `main`
2. Make your changes
3. Ensure tests pass: `go test ./...`
4. Submit a pull request

### PR Branch Naming

**Never create PRs from your fork's `main` branch.** Always create a dedicated branch for each PR:

```bash
# Good - dedicated branch per PR
git checkout -b fix/deacon-startup upstream/main
git checkout -b feat/auto-seance upstream/main

# Bad - PR from main accumulates unrelated commits
git checkout main  # Don't PR from here!
```

Why this matters:
- PRs from `main` accumulate ALL commits pushed to your fork
- Multiple contributors pushing to the same fork's `main` creates chaos
- Reviewers can't tell which commits belong to which PR
- You can't have multiple PRs open simultaneously

Branch naming conventions:
- `fix/*` - Bug fixes
- `feat/*` - New features
- `refactor/*` - Code restructuring
- `docs/*` - Documentation only

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and small
- Add comments for non-obvious logic
- Include tests for new functionality

## Design Philosophy

Gas Town follows two core principles that shape every contribution. Understanding
these will save you (and reviewers) time.

### Zero Framework Cognition (ZFC)

**Go provides transport. Agents provide cognition.**

Gas Town's Go code handles plumbing: tmux sessions, message delivery, hooks,
nudges, file transport, and observability primitives (like `bd show --json`).
All reasoning, judgment calls, and decision-making happen in the AI agents via
molecule formulas and role templates.

This means:
- **No hardcoded thresholds in Go.** Don't write `if age > 5*time.Minute`
  to decide if an agent is stuck. Expose the age as data and let the agent decide.
- **No heuristics in Go.** Don't write detection logic that pattern-matches
  agent behavior. Give agents the tools to observe, and let them reason.
- **Formulas over subcommands.** If the feature is "detect X and do Y," it's
  probably a molecule step, not a new `gt` subcommand.

**The test:** Before adding Go code, ask yourself — *"Am I adding transport or
cognition?"* If the answer is cognition, it should be a molecule step or
formula instruction instead.

For the full rationale, see
[Zero Framework Cognition](https://steve-yegge.medium.com/zero-framework-cognition-a-way-to-build-resilient-ai-applications-56b090ed3e69).

### Bitter Lesson Alignment

Gas Town bets on models getting smarter, not on hand-crafted heuristics getting
more elaborate. If an AI agent can observe data and reason about it, we expose
the data (transport) rather than encoding the reasoning (cognition). Today's
clumsy heuristic is tomorrow's technical debt — but a clean observability
primitive ages well.

**Examples:**

| Good (transport) | Bad (cognition in Go) |
|---|---|
| `gt nudge <session> "message"` | Go code deciding *when* to nudge |
| `bd show --json` exposing step status | Go code deciding *what* step status means |
| `tmux has-session` checking liveness | Go code with hardcoded "stuck after N minutes" |

## What to Contribute

Good first contributions:
- Bug fixes with clear reproduction steps
- Documentation improvements
- Test coverage for untested code paths
- Small, focused features

For larger changes, please open an issue first to discuss the approach.

## Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Keep the first line under 72 characters
- Reference issues when applicable: `Fix timeout bug (gt-xxx)`

## Testing

Run the full test suite before submitting:

```bash
go test ./...
```

For specific packages:

```bash
go test ./internal/wisp/...
go test ./cmd/gt/...
```

### Integration Test Guards

Integration tests (tagged `//go:build integration`) require external resources
that may not be available in every environment. Use the helpers in
`internal/testutil` to skip gracefully when prerequisites are missing:

| Helper | When to use |
|--------|-------------|
| `testutil.RequireDoltContainer(t)` | Test needs a running Dolt SQL server (starts a Docker container) |
| `testutil.StartIsolatedDoltContainer(t)` | Test needs its own isolated Dolt instance (per-test container) |
| `testutil.RequireTownEnv(t)` | Test needs a live Gas Town workspace (checks `workspace.FindFromCwd` + `rigs.json`); returns root path |

**`requireDoltServer`** (in `internal/cmd`) is a local wrapper around
`testutil.RequireDoltContainer` used by the `cmd` package's integration tests.

**When to use which guard:**

- Tests that connect to Dolt (create databases, run SQL) →
  `RequireDoltContainer` or `StartIsolatedDoltContainer`
- Tests that need a real Gas Town directory tree (shell out to `gt`/`bd` with
  workspace detection) → `RequireTownEnv`
- Tests that create their own temporary town via `t.TempDir()` → no guard needed
  (they are self-contained)

For packages with many Dolt-dependent tests, prefer adding
`testutil.EnsureDoltContainerForTestMain()` in a `TestMain` function so all
tests in the package share a single container.

## Releasing

Releases are cut from tags of the form `vX.Y.Z`. See [RELEASING.md](RELEASING.md)
for the full workflow. One guardrail to know about:

- `make check-version-tag` verifies the `Version` constant in
  `internal/cmd/version.go` matches the tag at HEAD. The release workflow runs
  this before GoReleaser and fails the release on mismatch. Prevents recurrence
  of [#3459](https://github.com/steveyegge/gastown/issues/3459). Run it locally
  after bumping if you want to catch drift before pushing the tag.

## Camp Leatherneck: Upstream Merge Policy

This repository is `camp-leatherneck/camp-leatherneck`, a productized
downstream distribution of this project (`steveyegge/gastown` is the
`upstream` remote) — see
[`docs/adr/0001-productized-downstream-distribution.md`](docs/adr/0001-productized-downstream-distribution.md)
and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full reasoning.
This section records the policy that keeps upstream merges cheap, per
[`CAMP_LEATHERNECK_ARCHITECTURE_CONSTITUTION.md`](../ai-powerhouse/projects/camp-leatherneck/architecture/CAMP_LEATHERNECK_ARCHITECTURE_CONSTITUTION.md) §8
(governing document, not part of this repo).

**Near-upstream — minimize divergence, merge freely, never rename:**
`internal/polecat/`, beads/molecules machinery, `internal/dolt/`, mail,
worktree and merge flow, daemon, hooks, patrols/dogs,
`cmd/gt-proxy-server`, `cmd/gt-proxy-client`, and files like
`internal/cmd/doctor.go` that upstream actively maintains.

**Intentionally divergent — the product layer:** `cmd/lt/`,
`internal/config/directives.go`, `internal/assets/` (directive templates,
scripts), `internal/phi/`, `internal/provenance/`, `internal/roster/`,
routing/runtime config, `README`, `CHANGELOG`, `docs/`, `Makefile`
naming, the Homebrew tap, `install.sh`.

**The rule that keeps merges cheap:** no file renames, package renames, or
symbol renames in near-upstream areas — ever. Renames are the dominant
cause of merge conflict, because git must re-detect the mapping on every
subsequent merge. New product-layer logic goes in new files; a
near-upstream file that needs product-layer behavior gets the smallest
possible seam (see ADR 0005 for a worked example — one import and one
function call added to `doctor.go`, not a refactor).

**Cadence:** fetch and merge `upstream/main` on a regular schedule
(monthly minimum, opportunistically sooner for a notable upstream fix).
Because the product layer never touches upstream paths, this stays a
low-conflict operation relative to the size of the gap — but **it is not
free, and letting it lapse makes it worse, not neutral.**

**Corrected 2026-07-31 (later same day):** this section previously claimed
"parity was exact (0 commits behind) as of 2026-07-31." That was measured
against a local `upstream/main` ref with no confirmed fresh fetch in that
session — fail-open, and false. A fresh, timestamped fetch the same day
(cross-checked live via `git ls-remote upstream main`) showed main **777
commits and 1,177 files behind**. A scratch merge against that real gap
conflicted in 138 of those 1,177 files — small, localized conflicts,
**not** confined to README/NOTICE/display strings as the example below used
to claim; near-upstream files with a product-layer seam (e.g. files touched
per the "smallest possible seam" rule above) can and did conflict too, at a
scale still fully reviewable by hand. See
`CAMP_LEATHERNECK_ARCHITECTURE_CERTIFICATION.md`'s Amendment for full
evidence. This cadence now has an owner and a tracked trigger — `hq-35iwf`
(bd), assigned to Joey — rather than being a sentence with no mechanism
behind it.

**⚠️ Any `git rev-list --count HEAD..upstream/main`-style count is invalid
evidence unless `git fetch upstream` ran in the same session immediately
before it, and the observation is recorded with a timestamp.** An unfetched
or undated count is not a measurement — it reads identically to a real one
and cannot be told apart from it after the fact, which is exactly how the
false "0 commits behind" claim above got repeated as fact.

```bash
git fetch upstream
# Verify freshness before trusting anything downstream of this fetch:
#   git rev-parse upstream/main  must equal  git ls-remote upstream main
git merge upstream/main
# Conflicts are expected to be small and reviewable, not necessarily
# confined to README/NOTICE/display strings — a large gap can surface
# conflicts anywhere a product-layer seam sits near upstream churn.
```

**Added 2026-07-31 (later same day): `lt doctor` now checks this itself.**
The certification Amendment above recorded, as one of four governance gaps
this defect exposed, that "`lt doctor` does not verify remote-reference
freshness" and that any future check "must fail closed ... or it reproduces
exactly this defect in code instead of in a document." That gap is closed —
see [`docs/adr/0006-upstream-freshness-check.md`](docs/adr/0006-upstream-freshness-check.md).
`lt doctor` now reports whether `upstream/main`'s local ref is fresh, stale
(older than this section's monthly-minimum cadence), never fetched, or
missing entirely, and — the actual point of the control — it never emits a
commit-distance number for any ref it can't first confirm is fresh. It
still never fetches on its own; the warning it prints includes the exact
`git fetch` command to run before trusting any distance claim again. This
does not replace `hq-35iwf` (the tracked effort to actually close the
777-commit gap) — it only makes it impossible for a stale ref to look like
parity again.

## Questions?

Open an issue for questions about contributing. We're happy to help!

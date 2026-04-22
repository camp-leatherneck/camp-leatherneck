# Releasing Camp Leatherneck (`lt`)

This doc covers the end-to-end release procedure for the `lt` binary and
the Homebrew tap Formula update. It's the companion to
`.github/workflows/release.yml` and `.goreleaser.yml`.

## One-time setup: the `TAP_GITHUB_TOKEN` secret

GoReleaser needs to push to `camp-leatherneck/homebrew-tap` when it
updates the Formula. The default `GITHUB_TOKEN` provided by GitHub
Actions is scoped to the current repo only, so a cross-repo PAT is
required.

**This is a human task.** Perform it once, then again whenever the PAT
expires (set a calendar reminder — fine-grained PATs max out at 1 year).

### Generate the PAT

1. Go to <https://github.com/settings/personal-access-tokens/new>
2. Choose **Fine-grained token**.
3. **Resource owner**: `camp-leatherneck`.
4. **Repository access**: *Only select repositories* → choose
   `camp-leatherneck/homebrew-tap`.
5. **Repository permissions**:
   - `Contents`: **Read and write**
   - `Metadata`: **Read-only** (auto-selected)
   - Everything else: leave at **No access**.
6. **Expiration**: 1 year (max). Set a reminder to rotate.
7. Name it something like `camp-leatherneck-tap-release`.
8. Generate, copy the token (`github_pat_...`).

### Register the secret on this repo

```bash
gh secret set TAP_GITHUB_TOKEN --repo camp-leatherneck/camp-leatherneck
# Paste the token when prompted.
```

Confirm:

```bash
gh secret list --repo camp-leatherneck/camp-leatherneck
```

You should see `TAP_GITHUB_TOKEN` alongside the automatic `GITHUB_TOKEN`.

## Cutting a release

1. **Pick a version.** Follow semver. Pre-releases use `-pre.N` suffix
   (e.g. `v0.1.0-pre.1`). GoReleaser auto-flags pre-release tags.
2. **Make sure `main` is green** (CI workflow passes on latest commit).
3. **Update the in-code `Version` constant** if `make check-version-tag`
   enforces it — otherwise skip.
4. **Tag and push:**

   ```bash
   git checkout main
   git pull
   git tag v0.1.0-pre.1
   git push origin v0.1.0-pre.1
   ```

5. **Watch the workflow:**

   ```bash
   gh run watch --repo camp-leatherneck/camp-leatherneck
   ```

   Or open <https://github.com/camp-leatherneck/camp-leatherneck/actions>.

## Post-release verification

After `release.yml` completes, verify:

1. **GitHub Release exists** with 4 tarballs + `checksums.txt`:

   ```bash
   gh release view v0.1.0-pre.1 --repo camp-leatherneck/camp-leatherneck
   ```

   Expect:
   - `camp-leatherneck_0.1.0-pre.1_darwin_amd64.tar.gz`
   - `camp-leatherneck_0.1.0-pre.1_darwin_arm64.tar.gz`
   - `camp-leatherneck_0.1.0-pre.1_linux_amd64.tar.gz`
   - `camp-leatherneck_0.1.0-pre.1_linux_arm64.tar.gz`
   - `checksums.txt`

2. **Tap Formula was updated.** GoReleaser commits to
   `camp-leatherneck/homebrew-tap` on the `main` branch with message
   `lt: release v0.1.0-pre.1`:

   ```bash
   gh api repos/camp-leatherneck/homebrew-tap/commits --jq '.[0].commit.message'
   ```

3. **Homebrew install works:**

   ```bash
   brew untap camp-leatherneck/tap 2>/dev/null || true
   brew tap camp-leatherneck/tap
   brew install lt
   lt --version
   ```

   (Pre-releases may require `brew install --HEAD` or explicit version
   pinning depending on how the tap exposes them.)

## If something fails

- **Workflow fails with `Env.TAP_GITHUB_TOKEN not found`** — the secret
  is missing or expired. Re-run the one-time setup above.
- **Formula push fails with 403** — PAT doesn't have `Contents: write`
  on `homebrew-tap`, or token is for the wrong resource owner.
- **Tag already exists / goreleaser dirty state** — delete the tag
  locally and remotely, fix the underlying commit, re-tag:

  ```bash
  git tag -d v0.1.0-pre.1
  git push origin :refs/tags/v0.1.0-pre.1
  ```

- **Release succeeded but tap commit is missing** — check the
  `Run GoReleaser` step logs for a `brew` block warning. Most likely
  the token was silently empty; rotate and re-run the workflow via
  `workflow_dispatch`.

## Related files

- `.goreleaser.yml` — build, archive, brew, release config
- `.github/workflows/release.yml` — the release workflow
- `.github/workflows/ci.yml` — PR-time validation (includes
  `goreleaser check` so config errors are caught before tagging)
- `install.sh` — curl|bash fallback distribution path

package provenance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// runGit runs git in dir, failing the test on error. Deterministic
// author/committer identity keeps fixtures reproducible across machines.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepoWithCommit creates a git repo at a fresh temp dir with one commit
// on main, and returns its path.
func newRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "f.txt")
	runGit(t, dir, "commit", "-q", "-m", "first")
	return dir
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func TestGatherUpstream_NoRemoteConfigured(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)

	info := GatherUpstream(dir, "upstream", "main", DefaultUpstreamStaleAfter)

	if info.Freshness != FreshnessNoRemote {
		t.Fatalf("Freshness = %q, want %q", info.Freshness, FreshnessNoRemote)
	}
	if info.CommitsBehind != -1 {
		t.Errorf("CommitsBehind = %d, want -1 (no distance claim without a remote)", info.CommitsBehind)
	}
}

func TestGatherUpstream_RemoteConfiguredButNeverFetched_MissingBranch(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)
	upstreamDir := newRepoWithCommit(t)

	runGit(t, dir, "remote", "add", "upstream", upstreamDir)
	// Deliberately no `git fetch` — the ref never gets created.

	info := GatherUpstream(dir, "upstream", "main", DefaultUpstreamStaleAfter)

	if info.Freshness != FreshnessMissingBranch {
		t.Fatalf("Freshness = %q, want %q", info.Freshness, FreshnessMissingBranch)
	}
	if info.RemoteURL != upstreamDir {
		t.Errorf("RemoteURL = %q, want %q", info.RemoteURL, upstreamDir)
	}
	if info.CommitsBehind != -1 {
		t.Errorf("CommitsBehind = %d, want -1 (no ref, no distance claim)", info.CommitsBehind)
	}
}

func TestGatherUpstream_FreshAfterFetch(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)
	upstreamDir := newRepoWithCommit(t)

	runGit(t, dir, "remote", "add", "upstream", upstreamDir)
	runGit(t, dir, "fetch", "-q", "upstream", "main")

	info := GatherUpstream(dir, "upstream", "main", DefaultUpstreamStaleAfter)

	if info.Freshness != FreshnessFresh {
		t.Fatalf("Freshness = %q, want %q (details: %+v)", info.Freshness, FreshnessFresh, info)
	}
	if !info.RefExists {
		t.Error("RefExists = false after a real fetch")
	}
	if info.LastFetchTime == nil {
		t.Fatal("LastFetchTime is nil after a real fetch")
	}
	if time.Since(*info.LastFetchTime) > time.Minute {
		t.Errorf("LastFetchTime = %v, want within the last minute", *info.LastFetchTime)
	}
	if info.EvidenceSource != "reflog" {
		t.Errorf("EvidenceSource = %q, want %q (a real fetch always writes a reflog entry)", info.EvidenceSource, "reflog")
	}
	// upstream and local are at the identical commit (both are one-commit
	// fixtures created independently) — 0 is a TRUE parity reading here,
	// unlike the fail-open bug, because freshness was verified first.
	if info.CommitsBehind != 0 {
		t.Errorf("CommitsBehind = %d, want 0 (identical single-commit fixtures)", info.CommitsBehind)
	}
}

// TestGatherUpstream_StaleRef_NeverReportsDistance is the direct regression
// test for the exact failure mode the 2026-07-31 certification amendment
// found: `git rev-list --count HEAD..upstream/main` returned 0 against a
// local upstream/main ref that had not been freshly fetched, indistinguishable
// from genuine parity. Here, the local ref is fetched once, then the
// upstream remote advances (a commit the local ref never sees), and the
// reflog timestamp is backdated past the staleness threshold. A naive
// `git rev-list --count HEAD..upstream/main` in this exact repo state
// would still report 0 — GatherUpstream must refuse to report any
// CommitsBehind number at all once it knows the ref is stale.
func TestGatherUpstream_StaleRef_NeverReportsDistance(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)
	upstreamDir := newRepoWithCommit(t)

	runGit(t, dir, "remote", "add", "upstream", upstreamDir)
	runGit(t, dir, "fetch", "-q", "upstream", "main")

	// Prove the naive, unsound method really does read as 0 here — this is
	// the exact fail-open measurement the amendment corrected.
	naiveCount := runGit(t, dir, "rev-list", "--count", "HEAD..upstream/main")
	if naiveCount != "0" {
		t.Fatalf("naive HEAD..upstream/main count = %q, want %q for this fixture to test the right scenario", naiveCount, "0")
	}

	// Now the real upstream moves — a commit the stale local ref never sees.
	if err := os.WriteFile(filepath.Join(upstreamDir, "g.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, upstreamDir, "add", "g.txt")
	runGit(t, upstreamDir, "commit", "-q", "-m", "second")

	// Backdate the reflog entry so GatherUpstream sees fetch evidence older
	// than the threshold, without needing to wait in real time.
	backdateReflog(t, dir, "upstream", "main", 48*time.Hour)

	info := GatherUpstream(dir, "upstream", "main", 24*time.Hour)

	if info.Freshness != FreshnessStale {
		t.Fatalf("Freshness = %q, want %q (details: %+v)", info.Freshness, FreshnessStale, info)
	}
	if info.CommitsBehind != -1 {
		t.Errorf("CommitsBehind = %d, want -1 — a stale ref must never report a distance number, even though the naive count reads %q", info.CommitsBehind, naiveCount)
	}
}

func TestGatherUpstream_ReflogMissing_FallsBackToRefFileMtime(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)
	upstreamDir := newRepoWithCommit(t)

	runGit(t, dir, "remote", "add", "upstream", upstreamDir)
	runGit(t, dir, "fetch", "-q", "upstream", "main")

	reflogPath := filepath.Join(dir, ".git", "logs", "refs", "remotes", "upstream", "main")
	if err := os.Remove(reflogPath); err != nil {
		t.Fatalf("removing reflog fixture: %v", err)
	}

	info := GatherUpstream(dir, "upstream", "main", DefaultUpstreamStaleAfter)

	if info.EvidenceSource != "ref-file-mtime" {
		t.Fatalf("EvidenceSource = %q, want %q after removing the reflog", info.EvidenceSource, "ref-file-mtime")
	}
	if info.Freshness != FreshnessFresh {
		t.Fatalf("Freshness = %q, want %q (details: %+v)", info.Freshness, FreshnessFresh, info)
	}
}

// TestGatherUpstream_RefWithNoEvidenceAtAll_NeverFetched simulates a ref
// that exists (e.g. restored from packed-refs, or seeded some way other
// than a fetch this repo recorded) but carries no reflog and no loose ref
// file — no on-disk evidence of when it was written at all. This must
// read as "never fetched", not silently as fresh.
func TestGatherUpstream_RefWithNoEvidenceAtAll_NeverFetched(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)
	upstreamDir := newRepoWithCommit(t)
	upstreamSHA := runGit(t, upstreamDir, "rev-parse", "HEAD")

	runGit(t, dir, "remote", "add", "upstream", upstreamDir)
	runGit(t, dir, "fetch", "-q", "upstream", "main")

	// Strip every trace of "when": delete the reflog, then pack the ref so
	// the loose ref file (and its mtime) is gone too, leaving only
	// packed-refs — which carries no per-ref timestamp.
	if err := os.Remove(filepath.Join(dir, ".git", "logs", "refs", "remotes", "upstream", "main")); err != nil {
		t.Fatalf("removing reflog fixture: %v", err)
	}
	runGit(t, dir, "pack-refs", "--all")
	loosePath := filepath.Join(dir, ".git", "refs", "remotes", "upstream", "main")
	if _, err := os.Stat(loosePath); err == nil {
		t.Fatalf("expected pack-refs to remove the loose ref file at %s", loosePath)
	}

	info := GatherUpstream(dir, "upstream", "main", DefaultUpstreamStaleAfter)

	if info.Freshness != FreshnessNeverFetched {
		t.Fatalf("Freshness = %q, want %q (details: %+v)", info.Freshness, FreshnessNeverFetched, info)
	}
	if !info.RefExists {
		t.Error("RefExists = false, want true — the ref is still resolvable via packed-refs")
	}
	if info.RefCommit != upstreamSHA {
		t.Errorf("RefCommit = %q, want %q", info.RefCommit, upstreamSHA)
	}
	if info.CommitsBehind != -1 {
		t.Errorf("CommitsBehind = %d, want -1 (no fetch evidence, no distance claim)", info.CommitsBehind)
	}
}

func TestFetchUpstreamAndRecheck_UnreachableRemote_ReportsFetchFailed(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)
	runGit(t, dir, "remote", "add", "upstream", "/nonexistent/path/does-not-exist-"+t.Name())

	info := FetchUpstreamAndRecheck(dir, "upstream", "main", DefaultUpstreamStaleAfter)

	if info.Freshness != FreshnessFetchFailed {
		t.Fatalf("Freshness = %q, want %q (details: %+v)", info.Freshness, FreshnessFetchFailed, info)
	}
	if info.Error == "" {
		t.Error("Error is empty, want the underlying git fetch failure message")
	}
	if info.CommitsBehind != -1 {
		t.Errorf("CommitsBehind = %d, want -1 (fetch failed, no distance claim)", info.CommitsBehind)
	}
}

func TestFetchUpstreamAndRecheck_Succeeds_ReportsFresh(t *testing.T) {
	requireGit(t)
	dir := newRepoWithCommit(t)
	upstreamDir := newRepoWithCommit(t)
	runGit(t, dir, "remote", "add", "upstream", upstreamDir)

	info := FetchUpstreamAndRecheck(dir, "upstream", "main", DefaultUpstreamStaleAfter)

	if info.Freshness != FreshnessFresh {
		t.Fatalf("Freshness = %q, want %q (details: %+v)", info.Freshness, FreshnessFresh, info)
	}
}

func TestUpstreamInfo_RemediationCommand(t *testing.T) {
	cases := []struct {
		freshness UpstreamFreshness
		wantEmpty bool
	}{
		{FreshnessNoRemote, true},
		{FreshnessFresh, true},
		{FreshnessMissingBranch, false},
		{FreshnessNeverFetched, false},
		{FreshnessStale, false},
		{FreshnessFetchFailed, false},
	}
	for _, c := range cases {
		u := UpstreamInfo{RemoteName: "upstream", Branch: "main", Freshness: c.freshness}
		got := u.RemediationCommand()
		if (got == "") != c.wantEmpty {
			t.Errorf("Freshness=%q: RemediationCommand() = %q, wantEmpty=%v", c.freshness, got, c.wantEmpty)
		}
	}
}

func TestReport_Warnings_SurfacesStaleUpstream(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	r := &Report{
		SourceRoot: "/x/source",
		Daemon:     runningDaemon(),
		Upstream: UpstreamInfo{
			RemoteName:     "upstream",
			Branch:         "main",
			RefExists:      true,
			LastFetchTime:  &old,
			EvidenceSource: "reflog",
			Freshness:      FreshnessStale,
			StaleAfter:     24 * time.Hour,
			CommitsBehind:  -1,
			ObservedAt:     now,
		},
	}
	w := r.Warnings()
	found := false
	for _, msg := range w {
		if strings.Contains(msg, "upstream/main") && strings.Contains(msg, "exceeds") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning describing the stale upstream ref, got: %v", w)
	}
}

func TestReport_Warnings_FreshUpstream_NoWarning(t *testing.T) {
	r := &Report{
		SourceRoot: "/x/source",
		Daemon:     runningDaemon(),
		Upstream: UpstreamInfo{
			RemoteName: "upstream",
			Branch:     "main",
			RefExists:  true,
			Freshness:  FreshnessFresh,
		},
	}
	w := r.Warnings()
	for _, msg := range w {
		if strings.Contains(msg, "upstream") {
			t.Errorf("expected no upstream warning when Freshness=fresh, got: %q", msg)
		}
	}
}

// backdateReflog rewrites the last line of the reflog for
// refs/remotes/<remote>/<branch> so its recorded timestamp is `age` in the
// past, without needing to actually wait or fake the system clock.
func backdateReflog(t *testing.T, repoDir, remote, branch string, age time.Duration) {
	t.Helper()
	path := filepath.Join(repoDir, ".git", "logs", "refs", "remotes", remote, branch)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading reflog %s: %v", path, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	last := lines[len(lines)-1]
	tabIdx := strings.Index(last, "\t")
	if tabIdx == -1 {
		t.Fatalf("reflog line has no message separator: %q", last)
	}
	header, msg := last[:tabIdx], last[tabIdx:]
	fields := strings.Fields(header)
	if len(fields) < 2 {
		t.Fatalf("could not parse reflog header: %q", header)
	}
	backdated := time.Now().Add(-age).Unix()
	fields[len(fields)-2] = strconv.FormatInt(backdated, 10)
	lines[len(lines)-1] = strings.Join(fields, " ") + msg
	out := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatalf("writing backdated reflog: %v", err)
	}
}

// Upstream freshness: closes the gap the 2026-07-31 architecture
// certification amendment found by reading internal/cmd/doctor.go and
// internal/provenance/provenance.go — "lt doctor does not verify
// remote-reference freshness" and "upstream freshness checks currently fail
// open." A governance certification had reported
// `git rev-list --count HEAD..upstream/main` = 0 against a local
// `upstream/main` ref with no confirmed fresh fetch in that session. That
// measurement is fail-open: a stale ref under-reports distance and reads
// identically to genuine parity, which is exactly how a two-week-old
// number got repeated as "verified" fact across five documents. See
// docs/adr/0006-upstream-freshness-check.md for the full design rationale.
package provenance

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// UpstreamFreshness classifies what lt doctor actually knows about a
// remote-tracking ref's currency, so a stale or never-refreshed ref can
// never be misread as "at parity" with the real remote.
type UpstreamFreshness string

const (
	// FreshnessNoRemote: the named remote isn't configured in this repo at all.
	FreshnessNoRemote UpstreamFreshness = "no-remote"
	// FreshnessMissingBranch: the remote is configured, but <remote>/<branch>
	// has no local remote-tracking ref (never fetched, or wrong branch name).
	FreshnessMissingBranch UpstreamFreshness = "missing-branch"
	// FreshnessNeverFetched: the ref exists, but this repo has no recorded
	// evidence — neither reflog nor loose-ref mtime — of when it was last
	// written. Never assumed fresh; this is the state a fail-open check
	// would silently treat as parity.
	FreshnessNeverFetched UpstreamFreshness = "never-fetched"
	// FreshnessStale: evidence exists, but it is older than the accepted
	// threshold (see UpstreamStaleAfter / DefaultUpstreamStaleAfter).
	FreshnessStale UpstreamFreshness = "stale"
	// FreshnessFresh: evidence exists and is within the accepted threshold.
	// The only state in which CommitsBehind is trustworthy enough to compute.
	FreshnessFresh UpstreamFreshness = "fresh"
	// FreshnessFetchFailed: an explicit fetch-then-recheck was attempted
	// (never during an ordinary `lt doctor` run — see GatherUpstream vs.
	// FetchUpstreamAndRecheck below) and the fetch itself failed, e.g. the
	// machine is offline or the remote is unreachable.
	FreshnessFetchFailed UpstreamFreshness = "fetch-failed"
)

// UpstreamInfo answers: how current is our local knowledge of the
// configured upstream remote's default branch, and is that knowledge
// trustworthy enough to support a commit-distance or parity claim.
//
// GatherUpstream never fetches — it only reads evidence already on disk
// (reflog, ref files) — so it is safe to run offline and never mutates
// repository state during an ordinary `lt doctor` run. CommitsBehind is
// therefore only populated when Freshness == FreshnessFresh: for every
// other state, a distance number would be exactly the unsupported claim
// this check exists to prevent, so none is produced.
type UpstreamInfo struct {
	RemoteName     string
	Branch         string
	RemoteURL      string
	RefExists      bool
	RefCommit      string     // resolved SHA of <remote>/<branch>, when RefExists
	LastFetchTime  *time.Time // when the local ref was last written; nil if unknown
	EvidenceSource string     // "reflog" | "ref-file-mtime" | "" (no evidence found)
	Freshness      UpstreamFreshness
	StaleAfter     time.Duration // threshold used to produce Freshness
	CommitsBehind  int           // -1 unless Freshness == FreshnessFresh
	ObservedAt     time.Time     // when this gather ran — the "as of" timestamp
	Error          string
}

// UpstreamRemoteName and UpstreamBranchName identify the remote/branch this
// check validates. Overridable package vars (not constants) so tests can
// point at fixture repos without needing a remote named exactly "upstream",
// mirroring the SourceRoot override pattern in doctor_check.go.
var (
	UpstreamRemoteName = "upstream"
	UpstreamBranchName = "main"
)

// DefaultUpstreamStaleAfter is the maximum age of local fetch evidence
// before a remote-tracking ref is treated as stale rather than fresh.
//
// Grounded in this repo's own existing governance rather than invented:
// CONTRIBUTING.md's "Camp Leatherneck: Upstream Merge Policy" section
// commits to a monthly-minimum upstream sync cadence. 30 days *is* that
// cadence — a ref fetched anywhere within the required monthly window
// reads as fresh, and one left untouched longer than the policy's own
// stated minimum reads as stale. If that governance cadence ever changes,
// this default should change with it (see
// docs/adr/0006-upstream-freshness-check.md).
const DefaultUpstreamStaleAfter = 30 * 24 * time.Hour

// UpstreamStaleAfter is the effective threshold GatherUpstream uses when
// callers don't pass an explicit one. Override for tests or operators;
// defaults to DefaultUpstreamStaleAfter.
var UpstreamStaleAfter = DefaultUpstreamStaleAfter

func gitRun(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...) //nolint:gosec // args are fixed literals plus repo-internal names, not external input
	cmd.Dir = repoPath
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// GatherUpstream reports what is known about remote/branch's remote-tracking
// ref in the repo at repoPath, using only evidence already on disk. It never
// invokes `git fetch` and therefore never touches the network or mutates
// repository state — safe to call from an ordinary, offline `lt doctor` run.
func GatherUpstream(repoPath, remote, branch string, staleAfter time.Duration) UpstreamInfo {
	info := UpstreamInfo{
		RemoteName:    remote,
		Branch:        branch,
		StaleAfter:    staleAfter,
		CommitsBehind: -1,
		ObservedAt:    time.Now(),
	}

	if repoPath == "" {
		info.Freshness = FreshnessNoRemote
		info.Error = "source repo path not provided"
		return info
	}

	remotesOut, err := gitRun(repoPath, "remote")
	if err != nil {
		info.Freshness = FreshnessNoRemote
		info.Error = "could not list git remotes: " + err.Error()
		return info
	}
	configured := false
	for _, r := range strings.Fields(remotesOut) {
		if r == remote {
			configured = true
			break
		}
	}
	if !configured {
		info.Freshness = FreshnessNoRemote
		return info
	}
	if url, err := gitRun(repoPath, "remote", "get-url", remote); err == nil {
		info.RemoteURL = url
	}

	refName := remote + "/" + branch
	sha, err := gitRun(repoPath, "rev-parse", "--verify", refName)
	if err != nil || sha == "" {
		info.Freshness = FreshnessMissingBranch
		return info
	}
	info.RefExists = true
	info.RefCommit = sha

	if t, source, ok := lastRefWriteTime(repoPath, remote, branch); ok {
		info.LastFetchTime = &t
		info.EvidenceSource = source
	}

	if info.LastFetchTime == nil {
		info.Freshness = FreshnessNeverFetched
		return info
	}

	age := info.ObservedAt.Sub(*info.LastFetchTime)
	if age > staleAfter {
		info.Freshness = FreshnessStale
		return info
	}
	info.Freshness = FreshnessFresh

	// Only computed once freshness is established — see the type doc above.
	if out, err := gitRun(repoPath, "rev-list", "--count", "HEAD.."+refName); err == nil {
		if n, err := strconv.Atoi(out); err == nil {
			info.CommitsBehind = n
		}
	}

	return info
}

// lastRefWriteTime finds the most reliable on-disk evidence of when
// refs/remotes/<remote>/<branch> was last written.
//
// The reflog is preferred: every entry carries an explicit committer
// timestamp, and git appends one on every fetch that moves or confirms the
// ref (a fast-forward, or even a no-op fetch that just re-confirms the same
// SHA). If no reflog exists (core.logAllRefUpdates off, or the entry
// predates that setting), the loose ref file's mtime is used as a weaker
// but still repository-native fallback — remote-tracking refs are written
// only by fetch or explicit update-ref, not by unrelated operations like
// `git gc`. If neither exists, the caller must treat this as unknown
// (FreshnessNeverFetched), never silently assume freshness.
func lastRefWriteTime(repoPath, remote, branch string) (t time.Time, source string, ok bool) {
	if reflogPath, err := gitPathAbs(repoPath, "logs/refs/remotes/"+remote+"/"+branch); err == nil {
		if ts, err := lastReflogTimestamp(reflogPath); err == nil {
			return ts, "reflog", true
		}
	}

	if refPath, err := gitPathAbs(repoPath, "refs/remotes/"+remote+"/"+branch); err == nil {
		if fi, err := os.Stat(refPath); err == nil {
			return fi.ModTime(), "ref-file-mtime", true
		}
	}

	return time.Time{}, "", false
}

// gitPathAbs resolves a path relative to repoPath's actual git directory
// (which may not be "<repoPath>/.git" — worktrees and GIT_DIR overrides
// both break that assumption), via `git rev-parse --git-path`.
func gitPathAbs(repoPath, relPath string) (string, error) {
	out, err := gitRun(repoPath, "rev-parse", "--git-path", relPath)
	if err != nil || out == "" {
		return "", fmt.Errorf("could not resolve git-path %q: %w", relPath, err)
	}
	if filepath.IsAbs(out) {
		return out, nil
	}
	return filepath.Join(repoPath, out), nil
}

// lastReflogTimestamp parses the last non-empty line of a git reflog file
// and returns its recorded timestamp. Reflog line format:
//
//	<old-sha> <new-sha> <name> <email> <unix-ts> <tz>\t<message>
func lastReflogTimestamp(path string) (time.Time, error) {
	f, err := os.Open(path) //nolint:gosec // path resolved via `git rev-parse --git-path`, not user input
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()

	var lastLine string
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		if line := scanner.Text(); strings.TrimSpace(line) != "" {
			lastLine = line
		}
	}
	if err := scanner.Err(); err != nil {
		return time.Time{}, err
	}
	if lastLine == "" {
		return time.Time{}, fmt.Errorf("empty reflog at %s", path)
	}

	header := lastLine
	if tabIdx := strings.Index(lastLine, "\t"); tabIdx != -1 {
		header = lastLine[:tabIdx]
	}
	fields := strings.Fields(header)
	if len(fields) < 2 {
		return time.Time{}, fmt.Errorf("could not parse reflog line: %q", lastLine)
	}
	// Timestamp is always the second-to-last field — the last is the tz
	// offset, and everything before it is old-sha/new-sha/name/email
	// (name and email may themselves contain spaces).
	tsStr := fields[len(fields)-2]
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse reflog timestamp %q: %w", tsStr, err)
	}
	return time.Unix(ts, 0), nil
}

// FetchUpstreamAndRecheck performs the one network-touching operation this
// package offers, and only when a caller explicitly asks for it — never
// from an ordinary `lt doctor` run, per the established non-mutating
// doctor contract (no existing check performs network I/O, in --fix or
// otherwise; see docs/adr/0006-upstream-freshness-check.md). It runs
// `git fetch <remote> <branch>`, then re-derives UpstreamInfo from the
// result. If the fetch fails (offline, revoked credentials, unreachable
// host), the returned info reports FreshnessFetchFailed with the
// underlying error — it does not fall back to reporting the stale
// pre-fetch state as if it were fresh.
func FetchUpstreamAndRecheck(repoPath, remote, branch string, staleAfter time.Duration) UpstreamInfo {
	if _, err := gitRun(repoPath, "fetch", remote, branch); err != nil {
		return UpstreamInfo{
			RemoteName:    remote,
			Branch:        branch,
			StaleAfter:    staleAfter,
			CommitsBehind: -1,
			ObservedAt:    time.Now(),
			Freshness:     FreshnessFetchFailed,
			Error:         err.Error(),
		}
	}
	return GatherUpstream(repoPath, remote, branch, staleAfter)
}

// upstreamWarnings translates the Upstream dimension into doctor-facing
// warning strings. Freshness problems surface as warnings, not violations:
// a fork that has merely missed its fetch cadence (or is being inspected
// offline) is not broken the way a dead daemon or a missing directive is,
// and a hard failure here would punish normal offline use. What must never
// happen — the defect this check exists to close — is a stale or
// never-fetched ref being silently read as "at parity"; every non-fresh
// state below is reported explicitly, with its evidence and as-of time,
// instead.
func (r *Report) upstreamWarnings() []string {
	u := r.Upstream
	ref := u.RemoteName + "/" + u.Branch
	asOf := u.ObservedAt.UTC().Format(time.RFC3339)

	switch u.Freshness {
	case FreshnessNoRemote:
		return []string{fmt.Sprintf(
			"no %q remote configured in %s — upstream drift and mergeability cannot be assessed (as of %s)",
			u.RemoteName, r.SourceRoot, asOf,
		)}
	case FreshnessMissingBranch:
		return []string{fmt.Sprintf(
			"%s remote is configured but %s has no local ref — never fetched, or wrong branch name; run `%s` (as of %s)",
			u.RemoteName, ref, u.RemediationCommand(), asOf,
		)}
	case FreshnessNeverFetched:
		return []string{fmt.Sprintf(
			"%s exists but has no recorded fetch evidence (no reflog, no ref-file mtime) — any commit-distance claim against it is unsupported; run `%s` (as of %s)",
			ref, u.RemediationCommand(), asOf,
		)}
	case FreshnessStale:
		age := u.ObservedAt.Sub(*u.LastFetchTime)
		return []string{fmt.Sprintf(
			"%s last refreshed %s ago via %s (exceeds the %s policy threshold) — commit-distance claims against it are stale, not evidence; run `%s` (as of %s)",
			ref, age.Round(time.Hour), u.EvidenceSource, u.StaleAfter, u.RemediationCommand(), asOf,
		)}
	case FreshnessFetchFailed:
		return []string{fmt.Sprintf(
			"fetch of %s failed: %s — freshness could not be re-verified (as of %s)",
			ref, u.Error, asOf,
		)}
	case FreshnessFresh:
		return nil
	default:
		return nil
	}
}

// RemediationCommand returns the exact command a human should run to
// resolve a non-fresh UpstreamInfo, or "" if none is needed (FreshnessFresh)
// or possible (FreshnessNoRemote — adding a remote is a repo-config
// decision, not a fetch).
func (u UpstreamInfo) RemediationCommand() string {
	switch u.Freshness {
	case FreshnessMissingBranch, FreshnessNeverFetched, FreshnessStale, FreshnessFetchFailed:
		return fmt.Sprintf("git -C <source-repo> fetch %s %s", u.RemoteName, u.Branch)
	default:
		return ""
	}
}

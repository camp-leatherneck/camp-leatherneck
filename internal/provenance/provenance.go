// Package provenance answers "what is live?" for a Camp Leatherneck
// installation in one call, per CAMP_LEATHERNECK_ARCHITECTURE_CONSTITUTION.md
// §7 and its C2 certification ruling: this logic lives in a new
// product-layer file, not in internal/cmd/doctor.go (a near-upstream file
// upstream actively maintains), which is touched only by a one-line
// registration.
//
// The provenance identity this package checks:
//
//	launchd's declared program ≡ the running daemon's executable ≡ the lt
//	on PATH ≡ a known commit in ~/camp-leatherneck
//
// Any mismatch is a hard failure, not a warning buried in a log — that
// silence is exactly how the 99-day RTO failure and the stale Homebrew
// gt both persisted undetected.
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// BinaryInfo describes one named executable found on PATH (e.g. "lt" or "gt").
type BinaryInfo struct {
	Name        string // "lt" or "gt"
	PathOnPATH  string // first match from PATH lookup (exec.LookPath)
	RealPath    string // symlink-resolved target
	SHA256      string
	Version     string
	Commit      string
	BuildTime   string
	LookupError string // non-empty if the binary wasn't found on PATH at all
}

// ShadowBinary is a second, distinct executable found for the same name
// somewhere else on PATH — the exact hazard class that let Homebrew
// gastown silently contend with this build for the name "gt".
type ShadowBinary struct {
	Name             string
	Path             string
	SHA256           string
	MatchesCanonical bool
}

// SourceInfo answers: does the running binary's commit actually exist in
// the source repo, and is the source repo's working tree clean?
type SourceInfo struct {
	RepoPath          string
	Commit            string
	CommitFoundInRepo bool
	Relationship      string // "head", "ancestor-of-head", "descendant-of-head", "diverged", "unknown"
	WorkingTreeClean  bool
	Error             string
}

// DaemonInfo describes the actually-running daemon process, if any.
type DaemonInfo struct {
	Running           bool
	PID               int
	ExecutablePath    string // real (symlink-resolved) path of the running process's binary
	MatchesPATHBinary bool
	WorkingDirectory  string // best-effort; empty if not determinable
	Error             string
}

// LaunchdJob describes one com.campleatherneck.* LaunchAgent.
type LaunchdJob struct {
	Label           string
	PlistPath       string
	PlistExists     bool
	DeclaredProgram string
	ProgramExists   bool // does the file at DeclaredProgram actually exist
	Loaded          bool // present in `launchctl list`
	LastExitStatus  string
	StatusError     string
}

// RoleDirectiveStatus is the resolution result for one role.
type RoleDirectiveStatus struct {
	Role          string
	FileName      string
	Present       bool
	DirectiveLess bool // true for roles like "dog"/"crew" that have no directive by design (C4)
}

// RosterFinding flags a standalone roster file found outside version control.
type RosterFinding struct {
	Path   string
	Reason string
}

// Report is the full provenance snapshot.
type Report struct {
	GeneratedAt time.Time
	TownRoot    string
	SourceRoot  string

	Binaries   []BinaryInfo   // canonical lt (and gt, if present) as resolved via PATH
	Shadows    []ShadowBinary // additional lt/gt found elsewhere on PATH
	Source     SourceInfo
	Upstream   UpstreamInfo // freshness of the source repo's upstream remote-tracking ref
	Daemon     DaemonInfo
	Launchd    []LaunchdJob
	Directives []RoleDirectiveStatus
	Roster     []RosterFinding
}

// sha256File returns the hex-encoded sha256 of the file at path.
func sha256File(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // provenance reporting reads its own binary/known paths
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// resolveReal resolves symlinks; returns the input path unchanged on error.
func resolveReal(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

// GatherBinaries resolves "lt" and "gt" via PATH (exec.LookPath), each
// with their real path, sha256, and embedded version info obtained by
// running `<binary> version --json`.
func GatherBinaries() []BinaryInfo {
	names := []string{"lt", "gt"}
	var out []BinaryInfo
	for _, name := range names {
		info := BinaryInfo{Name: name}
		p, err := exec.LookPath(name)
		if err != nil {
			info.LookupError = err.Error()
			out = append(out, info)
			continue
		}
		info.PathOnPATH = p
		info.RealPath = resolveReal(p)
		if sum, err := sha256File(info.RealPath); err == nil {
			info.SHA256 = sum
		}
		if v, c, bt, err := runVersionJSON(p); err == nil {
			info.Version, info.Commit, info.BuildTime = v, c, bt
		}
		out = append(out, info)
	}
	return out
}

// runVersionJSON shells out to `<path> version --json` and extracts the
// fields provenance needs, without importing internal/cmd (which would be
// an import cycle — internal/cmd imports this package's checks, not the
// reverse). This is the intended boundary: provenance treats "the binary"
// as an external artifact to interrogate, the same way an operator would.
func runVersionJSON(path string) (version, commit, buildTime string, err error) {
	out, err := exec.Command(path, "version", "--json").Output() //nolint:gosec // path is from exec.LookPath / known install locations
	if err != nil {
		return "", "", "", err
	}
	// Minimal, dependency-free field extraction (avoids importing
	// internal/cmd's versionInfo type, preserving the layering above).
	get := func(field string) string {
		marker := `"` + field + `": "`
		idx := strings.Index(string(out), marker)
		if idx == -1 {
			return ""
		}
		rest := string(out)[idx+len(marker):]
		end := strings.Index(rest, `"`)
		if end == -1 {
			return ""
		}
		return rest[:end]
	}
	return get("version"), get("commit"), get("build_time"), nil
}

// GatherShadows scans every directory on PATH for additional "lt"/"gt"
// executables beyond the first (canonical) match GatherBinaries found, and
// reports whether each one's content matches the canonical binary. This is
// the check that would have caught Homebrew gastown occupying
// /opt/homebrew/bin/gt before it ever became a live incident.
func GatherShadows(canonical []BinaryInfo) []ShadowBinary {
	canonicalReal := map[string]string{}
	for _, b := range canonical {
		if b.RealPath != "" {
			canonicalReal[b.Name] = b.RealPath
		}
	}

	pathEnv := os.Getenv("PATH")
	dirs := strings.Split(pathEnv, string(os.PathListSeparator))
	seen := map[string]bool{}
	var shadows []ShadowBinary

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		for _, name := range []string{"lt", "gt"} {
			candidate := filepath.Join(dir, name)
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			fi, err := os.Lstat(candidate)
			if err != nil || fi.IsDir() {
				continue
			}
			real := resolveReal(candidate)
			want, ok := canonicalReal[name]
			if ok && real == want {
				continue // this IS the canonical binary, not a shadow
			}
			sb := ShadowBinary{Name: name, Path: candidate}
			if sum, err := sha256File(real); err == nil {
				sb.SHA256 = sum
			}
			sb.MatchesCanonical = ok && sb.SHA256 != "" && func() bool {
				wantSum, err := sha256File(want)
				return err == nil && wantSum == sb.SHA256
			}()
			shadows = append(shadows, sb)
		}
	}
	return shadows
}

// GatherSource checks whether commit exists in the repo at repoPath, and
// its ancestry relationship to HEAD, and whether the working tree is clean.
func GatherSource(repoPath, commit string) SourceInfo {
	info := SourceInfo{RepoPath: repoPath, Commit: commit}
	if repoPath == "" {
		info.Error = "source repo path not provided"
		return info
	}
	if commit == "" {
		info.Error = "no commit to check (binary reported no commit)"
		return info
	}

	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}

	if _, err := run("cat-file", "-e", commit); err != nil {
		info.CommitFoundInRepo = false
		info.Relationship = "unknown"
		info.Error = "commit not found in " + repoPath
		return info
	}
	info.CommitFoundInRepo = true

	head, err := run("rev-parse", "HEAD")
	if err != nil {
		info.Relationship = "unknown"
		info.Error = "could not resolve HEAD: " + err.Error()
		return info
	}
	resolvedCommit, _ := run("rev-parse", commit)
	if resolvedCommit == head {
		info.Relationship = "head"
	} else if _, err := run("merge-base", "--is-ancestor", resolvedCommit, "HEAD"); err == nil {
		info.Relationship = "ancestor-of-head"
	} else if _, err := run("merge-base", "--is-ancestor", "HEAD", resolvedCommit); err == nil {
		info.Relationship = "descendant-of-head"
	} else {
		info.Relationship = "diverged"
	}

	status, err := run("status", "--porcelain")
	if err == nil {
		info.WorkingTreeClean = status == ""
	}

	return info
}

// GatherDaemon inspects the running lt daemon process, if any, comparing
// its real executable path against the canonical PATH binary.
func GatherDaemon(pathBinaryRealPath string) DaemonInfo {
	info := DaemonInfo{}

	out, err := exec.Command("pgrep", "-fl", "lt daemon run").Output()
	if err != nil {
		info.Running = false
		return info
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		info.Running = false
		return info
	}
	// First matching process; format is "<pid> <cmd...>"
	fields := strings.SplitN(strings.TrimSpace(lines[0]), " ", 2)
	if len(fields) < 1 {
		info.Error = "could not parse pgrep output: " + lines[0]
		return info
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		info.Error = "could not parse PID from pgrep output: " + lines[0]
		return info
	}
	info.Running = true
	info.PID = pid

	// macOS/BSD: resolve the executable path for this PID via ps.
	psOut, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err == nil {
		execPath := strings.TrimSpace(string(psOut))
		info.ExecutablePath = resolveReal(execPath)
		info.MatchesPATHBinary = pathBinaryRealPath != "" && info.ExecutablePath == pathBinaryRealPath
	}

	return info
}

// GatherLaunchd inspects every ~/Library/LaunchAgents/com.campleatherneck.*
// plist: does the declared program path actually exist, and is the job
// currently loaded.
func GatherLaunchd() []LaunchdJob {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	loadedLabels := loadedLaunchdLabels()

	var jobs []LaunchdJob
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "com.campleatherneck.") || !strings.HasSuffix(name, ".plist") {
			continue
		}
		label := strings.TrimSuffix(name, ".plist")
		job := LaunchdJob{Label: label, PlistPath: filepath.Join(dir, name), PlistExists: true}

		program, err := plistProgramArguments0(job.PlistPath)
		if err != nil {
			job.StatusError = err.Error()
		} else {
			job.DeclaredProgram = program
			if _, err := os.Stat(program); err == nil {
				job.ProgramExists = true
			}
		}

		job.Loaded = loadedLabels[label]
		jobs = append(jobs, job)
	}
	return jobs
}

// loadedLaunchdLabels returns the set of job labels currently known to
// launchd (via `launchctl list`), regardless of running state.
func loadedLaunchdLabels() map[string]bool {
	out, err := exec.Command("launchctl", "list").Output()
	result := map[string]bool{}
	if err != nil {
		return result
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && strings.HasPrefix(fields[2], "com.campleatherneck.") {
			result[fields[2]] = true
		}
	}
	return result
}

// plistProgramArguments0 extracts the first <string> under
// <key>ProgramArguments</key> from a plist file, without a full plist
// parser — the format here is fixed and simple (Camp Leatherneck writes
// these plists itself), and avoiding a new dependency keeps this package
// small per Constitution §7's "no broad metadata framework" spirit.
func plistProgramArguments0(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is enumerated from ~/Library/LaunchAgents, not user input
	if err != nil {
		return "", err
	}
	content := string(data)
	idx := strings.Index(content, "<key>ProgramArguments</key>")
	if idx == -1 {
		return "", fmt.Errorf("no ProgramArguments key found in %s", path)
	}
	rest := content[idx:]
	arrStart := strings.Index(rest, "<array>")
	if arrStart == -1 {
		return "", fmt.Errorf("no <array> after ProgramArguments in %s", path)
	}
	rest = rest[arrStart:]
	strStart := strings.Index(rest, "<string>")
	if strStart == -1 {
		return "", fmt.Errorf("no <string> entries in ProgramArguments array in %s", path)
	}
	rest = rest[strStart+len("<string>"):]
	strEnd := strings.Index(rest, "</string>")
	if strEnd == -1 {
		return "", fmt.Errorf("unterminated <string> in %s", path)
	}
	return strings.TrimSpace(rest[:strEnd]), nil
}

// coreRoleFiles mirrors internal/config.roleToFileName for the roles that
// map to a Marine-named directive file. Duplicated rather than imported
// because roleToFileName is unexported (it's an internal implementation
// detail of LoadRoleDirective, not a public API of internal/config).
// TestGatherDirectives_KnownCoreRolesResolveCorrectly asserts each of these
// five mappings against expected values, so drift between this map and the
// real roleToFileName would show up as a directive resolving to the wrong
// file — a real, user-visible failure, not just a hidden divergence.
var coreRoleFiles = map[string]string{
	"mayor":    "lt",
	"deacon":   "top",
	"witness":  "sarge",
	"refinery": "gunny",
	"polecat":  "marine",
}

// directiveLessRoles are roster entities with no directive file by design
// (C4 certification ruling) — their absence from directives/ must not be
// flagged as an error.
var directiveLessRoles = map[string]bool{
	"dog":  true,
	"crew": true,
}

// specialistRoles are the ephemeral specialists, which have no entry in
// config.AllRoles() (they're not persistent Gastown roles) but do have a
// directive file each, per ~/gt/directives/lt.md's specialist table.
var specialistRoles = []string{
	"recon", "snoop", "doc", "housemouse", "gunbunny", "boxkicker",
	"groundguide", "brush", "scribe", "sapper", "marksman", "coach",
	"qrf", "rto", "firewatch",
}

// GatherDirectives resolves every known role (core Gastown roles from
// config.AllRoles(), plus the ephemeral specialists) against townRoot's
// directive set.
func GatherDirectives(townRoot string, allRoles []string) []RoleDirectiveStatus {
	var out []RoleDirectiveStatus

	check := func(role, fileName string, directiveLess bool) RoleDirectiveStatus {
		status := RoleDirectiveStatus{Role: role, FileName: fileName, DirectiveLess: directiveLess}
		if directiveLess {
			return status
		}
		path := filepath.Join(townRoot, "directives", fileName+".md")
		if _, err := os.Stat(path); err == nil {
			status.Present = true
		}
		return status
	}

	for _, role := range allRoles {
		if directiveLessRoles[role] {
			out = append(out, check(role, role, true))
			continue
		}
		fileName, ok := coreRoleFiles[role]
		if !ok {
			fileName = role // unmapped role: file name equals role name
		}
		out = append(out, check(role, fileName, false))
	}

	for _, role := range specialistRoles {
		out = append(out, check(role, role, false))
	}

	return out
}

// GatherRoster flags standalone roster files that could be mistaken for an
// authority — currently just the historical Desktop CSV (Constitution §6).
// This is deliberately narrow; full `lt roster` generation is Phase 4.
func GatherRoster() []RosterFinding {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var findings []RosterFinding
	csvPath := filepath.Join(home, "Desktop", "camp_leatherneck_roster.csv")
	if _, err := os.Stat(csvPath); err == nil {
		findings = append(findings, RosterFinding{
			Path:   csvPath,
			Reason: "unversioned standalone roster file — not the authority; lt roster (Phase 4) will supersede it",
		})
	}
	return findings
}

package provenance

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSHA256File_Deterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(path, []byte("provenance"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	if a != b || len(a) != 64 {
		t.Fatalf("sha256File(%q) = %q, %q — want deterministic 64-char hex", path, a, b)
	}
}

func TestPlistProgramArguments0_ParsesRealPlistShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.plist")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.campleatherneck.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/joeydeleon/.local/bin/lt</string>
        <string>daemon</string>
        <string>run</string>
    </array>
</dict>
</plist>
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := plistProgramArguments0(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/Users/joeydeleon/.local/bin/lt" {
		t.Errorf("plistProgramArguments0() = %q, want %q", got, "/Users/joeydeleon/.local/bin/lt")
	}
}

func TestPlistProgramArguments0_MissingKey_Errors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-program.plist")
	if err := os.WriteFile(path, []byte("<plist></plist>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := plistProgramArguments0(path); err == nil {
		t.Fatal("expected an error for a plist with no ProgramArguments key")
	}
}

// TestGatherDirectives_KnownCoreRolesResolveCorrectly proves the role→file
// mapping this package duplicates from internal/config.roleToFileName
// (deliberately, to avoid an import-cycle risk — see the comment on
// coreRoleFiles) actually matches, using a real directive tree fixture.
func TestGatherDirectives_KnownCoreRolesResolveCorrectly(t *testing.T) {
	townRoot := t.TempDir()
	dir := filepath.Join(townRoot, "directives")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"lt.md", "top.md", "sarge.md", "gunny.md", "marine.md", "recon.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("# directive"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	roles := []string{"mayor", "deacon", "witness", "refinery", "polecat", "dog", "crew"}
	results := GatherDirectives(townRoot, roles)

	byRole := map[string]RoleDirectiveStatus{}
	for _, r := range results {
		byRole[r.Role] = r
	}

	cases := []struct {
		role          string
		wantFileName  string
		wantPresent   bool
		wantDirectiveLess bool
	}{
		{"mayor", "lt", true, false},
		{"deacon", "top", true, false},
		{"witness", "sarge", true, false},
		{"refinery", "gunny", true, false},
		{"polecat", "marine", true, false},
		{"dog", "dog", false, true},   // C4: directive-less by design
		{"crew", "crew", false, true}, // C4: directive-less by design
	}
	for _, c := range cases {
		got, ok := byRole[c.role]
		if !ok {
			t.Fatalf("no result for role %q", c.role)
		}
		if got.DirectiveLess != c.wantDirectiveLess {
			t.Errorf("role %q: DirectiveLess = %v, want %v", c.role, got.DirectiveLess, c.wantDirectiveLess)
		}
		if !c.wantDirectiveLess && got.FileName != c.wantFileName {
			t.Errorf("role %q: FileName = %q, want %q", c.role, got.FileName, c.wantFileName)
		}
		if !c.wantDirectiveLess && got.Present != c.wantPresent {
			t.Errorf("role %q: Present = %v, want %v", c.role, got.Present, c.wantPresent)
		}
	}

	// Specialist resolves too (part of the always-appended specialist list).
	foundRecon := false
	for _, r := range results {
		if r.Role == "recon" {
			foundRecon = true
			if !r.Present {
				t.Error("recon directive should be present in the fixture")
			}
		}
	}
	if !foundRecon {
		t.Fatal("expected 'recon' (a specialist) in GatherDirectives results")
	}
}

// TestGatherDirectives_MissingCoreDirective_FlaggedNotPresent is the direct
// regression test for the failure class this whole check exists to catch —
// a role that's supposed to have a directive but doesn't.
func TestGatherDirectives_MissingCoreDirective_FlaggedNotPresent(t *testing.T) {
	townRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(townRoot, "directives"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Deliberately create NO directive files.
	results := GatherDirectives(townRoot, []string{"mayor"})
	// GatherDirectives always appends the specialist roles regardless of
	// allRoles (they're not part of config.AllRoles()) — so this is
	// 1 core role + len(specialistRoles), not just 1.
	if len(results) != 1+len(specialistRoles) {
		t.Fatalf("expected %d results (1 core + %d specialists), got %d", 1+len(specialistRoles), len(specialistRoles), len(results))
	}
	if results[0].Role != "mayor" || results[0].Present {
		t.Errorf("expected mayor's directive to be reported absent when the file doesn't exist, got %+v", results[0])
	}
}

// runningDaemon is a Daemon fixture that isolates tests for other
// dimensions from the (correct, independently-tested) "daemon not
// running" violation.
func runningDaemon() DaemonInfo {
	return DaemonInfo{Running: true, ExecutablePath: "/x", MatchesPATHBinary: true}
}

func TestReport_Violations_FlagsMissingDirective(t *testing.T) {
	r := &Report{
		Daemon: runningDaemon(),
		Directives: []RoleDirectiveStatus{
			{Role: "mayor", FileName: "lt", Present: false, DirectiveLess: false},
			{Role: "dog", FileName: "dog", Present: false, DirectiveLess: true},
		},
	}
	v := r.Violations()
	if len(v) != 1 {
		t.Fatalf("expected exactly 1 violation (missing mayor directive, dog is directive-less), got %d: %v", len(v), v)
	}
}

func TestReport_Violations_FlagsShadowMismatch(t *testing.T) {
	r := &Report{
		Daemon: runningDaemon(),
		Shadows: []ShadowBinary{
			{Name: "gt", Path: "/opt/homebrew/bin/gt", MatchesCanonical: false},
		},
	}
	v := r.Violations()
	if len(v) != 1 {
		t.Fatalf("expected exactly 1 violation for a non-matching shadow binary, got %d: %v", len(v), v)
	}
}

func TestReport_Violations_DoesNotFlagMatchingShadow(t *testing.T) {
	// A "shadow" that IS byte-identical to canonical (e.g. a second
	// symlink to the same file under a different PATH dir) is not a
	// violation — only a genuinely different binary is.
	r := &Report{
		Daemon: runningDaemon(),
		Shadows: []ShadowBinary{
			{Name: "lt", Path: "/opt/homebrew/bin/lt", MatchesCanonical: true},
		},
	}
	v := r.Violations()
	if len(v) != 0 {
		t.Fatalf("expected no violations for a matching shadow, got %d: %v", len(v), v)
	}
}

func TestReport_Violations_FlagsDaemonNotRunning(t *testing.T) {
	r := &Report{Daemon: DaemonInfo{Running: false}}
	v := r.Violations()
	found := false
	for _, x := range v {
		if x == "daemon is not running" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'daemon is not running' violation, got: %v", v)
	}
}

func TestReport_Violations_FlagsLaunchdMissingProgram(t *testing.T) {
	r := &Report{
		Daemon: runningDaemon(),
		Launchd: []LaunchdJob{
			{Label: "com.campleatherneck.daemon", DeclaredProgram: "/does/not/exist", ProgramExists: false},
		},
	}
	v := r.Violations()
	if len(v) != 1 {
		t.Fatalf("expected 1 violation for a launchd job declaring a nonexistent program, got %d: %v", len(v), v)
	}
}

// TestGatherSource_AgainstRealGitFixture proves ancestry/dirty detection
// against a real (throwaway) git repo, not a mock.
func TestGatherSource_AgainstRealGitFixture(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "f.txt")
	run("commit", "-q", "-m", "first")

	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headCmd.Dir = dir
	headOut, err := headCmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	head := string(headOut)
	head = head[:len(head)-1] // trim newline

	info := GatherSource(dir, head)
	if !info.CommitFoundInRepo {
		t.Fatal("expected commit to be found in the fixture repo")
	}
	if info.Relationship != "head" {
		t.Errorf("Relationship = %q, want %q", info.Relationship, "head")
	}
	if !info.WorkingTreeClean {
		t.Error("expected a freshly committed working tree to be clean")
	}

	// Now dirty the tree and re-check.
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("v2-uncommitted"), 0o644); err != nil {
		t.Fatal(err)
	}
	info2 := GatherSource(dir, head)
	if info2.WorkingTreeClean {
		t.Error("expected a modified working tree to be reported dirty")
	}
}

func TestGatherSource_UnknownCommit_ReportsNotFound(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	info := GatherSource(dir, "0000000000000000000000000000000000000000")
	if info.CommitFoundInRepo {
		t.Error("expected a nonexistent commit to be reported not-found")
	}
}

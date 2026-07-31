package version

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetRepoRoot_FindsCampLeatherneckSourceRoot is the regression test for
// a real bug found 2026-07-31 (LT_IMPLEMENTATION_CONTRACT.md Phase 3 item
// 8): GetRepoRoot's candidate list never included ~/camp-leatherneck, the
// actual canonical source root — only legacy ~/gt/gastown-style paths. This
// meant `lt stale` (and the rebuild-gt plugin, which depends on it) could
// only succeed by accident, when run from inside the source repo and
// falling through to the CWD git-toplevel fallback.
func TestGetRepoRoot_FindsCampLeatherneckSourceRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GT_ROOT", "") // isolate from a real GT_ROOT in the test environment

	srcDir := filepath.Join(home, "camp-leatherneck", "cmd", "lt")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := GetRepoRoot()
	if err != nil {
		t.Fatalf("GetRepoRoot() error = %v, want to find %s", err, filepath.Join(home, "camp-leatherneck"))
	}
	want := filepath.Join(home, "camp-leatherneck")
	if got != want {
		t.Errorf("GetRepoRoot() = %q, want %q", got, want)
	}
}

// TestGetRepoRoot_PrefersCampLeatherneckOverLegacyPath proves the new
// candidate is checked first when both exist, matching the Constitution's
// "canonical source root" framing rather than treating it as just another
// equally-weighted fallback.
func TestGetRepoRoot_PrefersCampLeatherneckOverLegacyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GT_ROOT", "")

	for _, rel := range []string{
		filepath.Join("camp-leatherneck", "cmd", "lt"),
		filepath.Join("gt", "gastown", "cmd", "lt"),
	} {
		dir := filepath.Join(home, rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := GetRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "camp-leatherneck")
	if got != want {
		t.Errorf("GetRepoRoot() = %q, want %q (camp-leatherneck should win over the legacy gt/gastown path)", got, want)
	}
}

func TestShortCommit(t *testing.T) {
	tests := []struct {
		name   string
		hash   string
		expect string
	}{
		{"full SHA", "abcdef1234567890abcdef1234567890abcdef12", "abcdef123456"},
		{"exactly 12", "abcdef123456", "abcdef123456"},
		{"short hash", "abcdef", "abcdef"},
		{"empty", "", ""},
		{"13 chars", "abcdef1234567", "abcdef123456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortCommit(tt.hash)
			if got != tt.expect {
				t.Errorf("ShortCommit(%q) = %q, want %q", tt.hash, got, tt.expect)
			}
		})
	}
}

func TestCommitsMatch(t *testing.T) {
	tests := []struct {
		name   string
		a, b   string
		expect bool
	}{
		{"identical full", "abcdef1234567890", "abcdef1234567890", true},
		{"prefix match short-long", "abcdef1234567", "abcdef1234567890abcd", true},
		{"prefix match long-short", "abcdef1234567890abcd", "abcdef1234567", true},
		{"no match", "abcdef1234567", "1234567abcdef", false},
		{"too short a", "abc", "abcdef1234567", false},
		{"too short b", "abcdef1234567", "abc", false},
		{"both too short", "abc", "abc", false},
		{"exactly 7 chars match", "abcdefg", "abcdefg", true},
		{"exactly 7 chars no match", "abcdefg", "abcdefh", false},
		{"6 chars too short", "abcdef", "abcdef", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commitsMatch(tt.a, tt.b)
			if got != tt.expect {
				t.Errorf("commitsMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expect)
			}
		})
	}
}

func TestSetCommit(t *testing.T) {
	original := Commit
	defer func() { Commit = original }()

	SetCommit("abc123def456")
	if Commit != "abc123def456" {
		t.Errorf("SetCommit did not set Commit; got %q", Commit)
	}
}

func TestIsBuildBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   bool
	}{
		{"main", true},
		{"master", true},
		{"carry/operational", true},
		{"carry/staging", true},
		{"carry/", true},
		{"fix/something", false},
		{"feat/new-thing", false},
		{"develop", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			if got := isBuildBranch(tt.branch); got != tt.want {
				t.Errorf("isBuildBranch(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

func TestCheckStaleBinary_NoCommit(t *testing.T) {
	original := Commit
	defer func() { Commit = original }()

	Commit = ""
	// Force resolveCommitHash to return empty by clearing Commit
	// (vcs.revision from build info may still be set, so this test
	// verifies the error path when no commit is available)
	info := CheckStaleBinary(t.TempDir())
	if info == nil {
		t.Fatal("CheckStaleBinary returned nil")
	}
	// Either we get an error (no commit) or we get a valid result from build info
	// Both are acceptable outcomes
	if info.BinaryCommit == "" && info.Error == nil {
		t.Error("expected error when binary commit is empty")
	}
}

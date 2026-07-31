package docaudit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindRepoRoot(cwd)
	if err != nil {
		t.Fatalf("could not find repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolved root %q has no go.mod", root)
	}
}

// TestDesignDocsDoNotClaimNonexistentCode is the actual regression guard.
// It runs against the real docs/design/ tree, so it enforces the invariant
// on every `go test ./...` run (part of the normal CI test pass — no new
// pipeline, no new tooling, just a test like any other).
func TestDesignDocsDoNotClaimNonexistentCode(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindRepoRoot(cwd)
	if err != nil {
		t.Fatalf("could not find repo root: %v", err)
	}
	violations, err := CheckDesignDocs(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	for _, v := range violations {
		t.Errorf("%s: checked box claims %q is done, but that path does not exist. "+
			"Either the code was never merged (uncheck the box / correct the status) "+
			"or the path is stale (fix the path).", v.DocPath, v.ClaimedPath)
	}
}

// TestCheckDesignDocs_DetectsKnownFixtureViolation proves the checker
// actually catches the failure shape it exists to catch — using a
// synthetic fixture, not relying on real docs staying broken (they
// shouldn't be, that's the point).
func TestCheckDesignDocs_DetectsKnownFixtureViolation(t *testing.T) {
	root := t.TempDir()
	designDir := filepath.Join(root, "docs", "design")
	if err := os.MkdirAll(designDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := "# Fixture\n\n- [x] Create `internal/models/database.go` — does not exist in this fixture\n"
	if err := os.WriteFile(filepath.Join(designDir, "fixture.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := CheckDesignDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].ClaimedPath != "internal/models/database.go" {
		t.Fatalf("unexpected claimed path: %q", violations[0].ClaimedPath)
	}

	// Now prove a real, existing path is NOT flagged (no false positives).
	doc2 := "# Fixture 2\n\n- [x] Create `go.mod` — this one really exists\n"
	if err := os.WriteFile(filepath.Join(designDir, "fixture2.md"), []byte(doc2), 0o644); err != nil {
		t.Fatal(err)
	}
	violations2, err := CheckDesignDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations2 {
		if v.DocPath == filepath.Join("docs", "design", "fixture2.md") {
			t.Fatalf("false positive on an existing path: %+v", v)
		}
	}
}

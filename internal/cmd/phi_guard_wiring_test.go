package cmd

import (
	"os"
	"strings"
	"testing"
)

// This test is a deliberate substitute for full mutation testing (no
// mutation-testing tool is part of this module's toolchain). It proves the
// specific, narrow claim the PHI containment contract depends on: the
// guard call is actually wired into both dispatch entry points, in source.
// If a future edit deletes the phiGuardForBead(...) call from either site
// — accidentally or as a "helpful" refactor — this test fails immediately,
// which is the practical goal of the "tests must prove that removing or
// bypassing the guard causes failure" requirement.
//
// It intentionally does NOT re-verify Classify/Guard logic (see
// internal/phi/phi_test.go for that) — only that the call sites exist.
func TestPHIGuardIsWiredIntoBothDispatchEntryPoints(t *testing.T) {
	cases := []struct {
		file        string
		mustContain string
	}{
		{"sling.go", "phiGuardForBead(townRoot, info)"},
		{"sling_dispatch.go", "phiGuardForBead(townRoot, info)"},
	}
	for _, c := range cases {
		data, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		if !strings.Contains(string(data), c.mustContain) {
			t.Fatalf("%s no longer calls %q — the PHI containment gate has been removed or bypassed at this dispatch entry point", c.file, c.mustContain)
		}
	}
}

// TestPHIGuardRunsBeforeForceOverrideLogic guards against a specific
// regression shape: someone moves the guard call to run AFTER the --force
// re-dispatch branching, which would let --force reach an agent session
// before classification succeeds. We check textual ordering within each
// file as a proxy for control-flow ordering, since both call sites are
// linear (no early branches) between bead-info resolution and the guard.
func TestPHIGuardRunsBeforeForceOverrideLogic(t *testing.T) {
	cases := []struct {
		file        string
		guardMarker string
		forceMarker string
	}{
		{"sling.go", "phiGuardForBead(townRoot, info)", `info.Status == "pinned" || info.Status == "hooked"`},
		{"sling_dispatch.go", "phiGuardForBead(townRoot, info)", `info.Status == "pinned" || info.Status == "hooked"`},
	}
	for _, c := range cases {
		data, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		content := string(data)
		guardIdx := strings.Index(content, c.guardMarker)
		forceIdx := strings.Index(content, c.forceMarker)
		if guardIdx == -1 || forceIdx == -1 {
			t.Fatalf("%s: expected markers not found (guardIdx=%d forceIdx=%d)", c.file, guardIdx, forceIdx)
		}
		if guardIdx > forceIdx {
			t.Fatalf("%s: PHI guard call appears AFTER force-override logic — it must run first so --force cannot bypass classification", c.file)
		}
	}
}

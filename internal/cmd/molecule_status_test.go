package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/camp-leatherneck/camp-leatherneck/internal/beads"
)

func TestOutputMoleculeStatus_StandaloneFormulaShowsVars(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir tempDir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	status := MoleculeStatusInfo{
		HasWork:         true,
		PinnedBead:      &beads.Issue{ID: "gt-wisp-xyz", Title: "Standalone formula work"},
		AttachedFormula: "mol-release",
		AttachedVars:    []string{"version=1.2.3", "channel=stable"},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputMoleculeStatus(status)

	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = oldStdout
	output := buf.String()

	if !strings.Contains(output, "📐 Formula: mol-release") {
		t.Fatalf("expected formula in output, got:\n%s", output)
	}
	if !strings.Contains(output, "--var version=1.2.3") || !strings.Contains(output, "--var channel=stable") {
		t.Fatalf("expected formula vars in output, got:\n%s", output)
	}
}

func TestOutputMoleculeStatus_FormulaWispShowsWorkflowContext(t *testing.T) {
	status := MoleculeStatusInfo{
		HasWork:         true,
		PinnedBead:      &beads.Issue{ID: "tool-wisp-demo", Title: "demo-hello"},
		AttachedFormula: "demo-hello",
		Progress: &MoleculeProgressInfo{
			RootID:     "tool-wisp-demo",
			RootTitle:  "demo-hello",
			TotalSteps: 3,
			DoneSteps:  0,
			ReadySteps: []string{"tool-wisp-step-1"},
		},
		NextAction: "Show the workflow steps: lt prime or bd mol current tool-wisp-demo",
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputMoleculeStatus(status)

	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = oldStdout
	output := buf.String()

	if !strings.Contains(output, "📐 Formula: demo-hello") {
		t.Fatalf("expected formula line in output, got:\n%s", output)
	}
	if strings.Contains(output, "No molecule attached") {
		t.Fatalf("formula wisp should not be rendered as naked work, got:\n%s", output)
	}
	if strings.Contains(output, "Attach a molecule to start work") {
		t.Fatalf("formula wisp should not suggest lt mol attach, got:\n%s", output)
	}
	if !strings.Contains(output, "Show the workflow steps: lt prime or bd mol current tool-wisp-demo") {
		t.Fatalf("expected workflow next action, got:\n%s", output)
	}
}

// TestAssigneeIdentityVariants pins the set of assignee-string forms that
// mayor/deacon may be persisted under (see hq-kllvf: resolveSelfTarget
// writes "mayor/"/"deacon/", the named-target sling path writes "mayor"/
// "deacon"). Non-town-level identities must not be broadened.
func TestAssigneeIdentityVariants(t *testing.T) {
	tests := []struct {
		target string
		want   []string
	}{
		{"mayor", []string{"mayor", "mayor/"}},
		{"mayor/", []string{"mayor", "mayor/"}},
		{"deacon", []string{"deacon", "deacon/"}},
		{"deacon/", []string{"deacon", "deacon/"}},
		{"gastown/polecats/toast", []string{"gastown/polecats/toast"}},
		{"gastown/crew/max", []string{"gastown/crew/max"}},
		{"deacon/boot", []string{"deacon/boot"}},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := assigneeIdentityVariants(tt.target)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("assigneeIdentityVariants(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}

// TestListByAssigneeAnyForm_CrossFormatMatch reproduces hq-wisp-oflnu /
// hq-kllvf: a bead dispatched via the named-target sling path persists
// Assignee="deacon" (no slash, per session.AgentIdentity.Address()), but
// self-status ('lt hook' / 'lt hook --json') queried for "deacon/" (per
// buildAgentIdentity) and got has_work:false despite the bead being
// correctly HOOKED and assigned. listByAssigneeAnyForm must find it either
// way, and must not falsely match unrelated agents.
func TestListByAssigneeAnyForm_CrossFormatMatch(t *testing.T) {
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skip("bd not installed, skipping test")
	}

	tmpDir := t.TempDir()
	initCmd := exec.Command("bd", "init", "--prefix", "test", "--quiet")
	initCmd.Dir = tmpDir
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("bd init: %v\n%s", err, output)
	}

	b := beads.New(tmpDir)

	mustHook := func(t *testing.T, title, assignee string) string {
		t.Helper()
		issue, err := b.Create(beads.CreateOptions{Title: title, Priority: 2})
		if err != nil {
			t.Fatalf("create bead %q: %v", title, err)
		}
		status := beads.StatusHooked
		if err := b.Update(issue.ID, beads.UpdateOptions{
			Status:   &status,
			Assignee: &assignee,
		}); err != nil {
			t.Fatalf("hook bead %q to %q: %v", issue.ID, assignee, err)
		}
		return issue.ID
	}

	// Sling-path write: no trailing slash.
	slingBead := mustHook(t, "patrol wisp via named-target sling", "deacon")
	// Self-attach-path write: trailing slash.
	selfAttachBead := mustHook(t, "hook via bare self-attach", "mayor/")
	// An unrelated agent must never be matched by the mayor/deacon variant expansion.
	otherBead := mustHook(t, "unrelated polecat work", "gastown/polecats/toast")

	t.Run("self-status query (deacon/) finds sling-path bead (deacon)", func(t *testing.T) {
		got := listByAssigneeAnyForm(b, beads.StatusHooked, "deacon/")
		if len(got) != 1 || got[0].ID != slingBead {
			t.Fatalf("expected to find %s, got %+v", slingBead, got)
		}
	})

	t.Run("named-target query (mayor) finds self-attach-path bead (mayor/)", func(t *testing.T) {
		got := listByAssigneeAnyForm(b, beads.StatusHooked, "mayor")
		if len(got) != 1 || got[0].ID != selfAttachBead {
			t.Fatalf("expected to find %s, got %+v", selfAttachBead, got)
		}
	})

	t.Run("exact-form query still finds its own bead", func(t *testing.T) {
		got := listByAssigneeAnyForm(b, beads.StatusHooked, "deacon")
		if len(got) != 1 || got[0].ID != slingBead {
			t.Fatalf("expected to find %s, got %+v", slingBead, got)
		}
	})

	t.Run("non-town-level target is not broadened", func(t *testing.T) {
		got := listByAssigneeAnyForm(b, beads.StatusHooked, "gastown/polecats/toast")
		if len(got) != 1 || got[0].ID != otherBead {
			t.Fatalf("expected to find only %s, got %+v", otherBead, got)
		}

		// A near-miss variant must not spuriously match the polecat's bead.
		if got := listByAssigneeAnyForm(b, beads.StatusHooked, "gastown/polecats/toast/"); len(got) != 0 {
			t.Fatalf("expected no match for trailing-slash variant of a non-town-level identity, got %+v", got)
		}
	})

	t.Run("no work for an idle agent", func(t *testing.T) {
		if got := listByAssigneeAnyForm(b, beads.StatusHooked, "deacon/boot"); len(got) != 0 {
			t.Fatalf("expected no hooked work for deacon/boot, got %+v", got)
		}
	})
}

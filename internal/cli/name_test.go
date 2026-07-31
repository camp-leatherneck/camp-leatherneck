package cli

import (
	"os"
	"sync"
	"testing"
)

func TestName_DefaultIsLt(t *testing.T) {
	// Reset singleton for test isolation
	nameOnce = sync.Once{}
	name = ""
	t.Setenv("GT_COMMAND", "")

	got := Name()
	if got != "lt" {
		t.Errorf("Name() = %q, want %q", got, "lt")
	}
}

func TestName_RespectsGT_COMMAND(t *testing.T) {
	nameOnce = sync.Once{}
	name = ""
	t.Setenv("GT_COMMAND", "gastown")

	got := Name()
	if got != "gastown" {
		t.Errorf("Name() = %q, want %q", got, "gastown")
	}
}

// TestIsInvokedAsGT_ExactBasenameOnly guards against the shim firing on any
// path merely containing "gt" — it must match the exact basename, not a
// substring (a real bug caught during manual testing: a symlink named
// "gt-test-symlink" did not trigger the notice, which is correct, but is
// easy to get wrong with a naive strings.Contains check).
func TestIsInvokedAsGT_ExactBasenameOnly(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	cases := []struct {
		arg0 string
		want bool
	}{
		{"/usr/local/bin/gt", true},
		{"gt", true},
		{"/usr/local/bin/lt", false},
		{"lt", false},
		{"/tmp/gt-test-symlink", false},
		{"/tmp/gtsomething", false},
		{"", false},
	}
	for _, c := range cases {
		os.Args = []string{c.arg0}
		if got := IsInvokedAsGT(); got != c.want {
			t.Errorf("IsInvokedAsGT() with os.Args[0]=%q = %v, want %v", c.arg0, got, c.want)
		}
	}
}

func TestIsInvokedAsGT_EmptyArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{}
	if IsInvokedAsGT() {
		t.Error("IsInvokedAsGT() with empty os.Args should be false, not panic or true")
	}
}

// TestName_FallsBackToGTWhenInvokedAsGT proves the shim keeps help/usage
// text consistent with how the user actually typed the command, without a
// second binary or a hand-maintained copy — Name() derives it from argv[0].
func TestName_FallsBackToGTWhenInvokedAsGT(t *testing.T) {
	nameOnce = sync.Once{}
	name = ""
	t.Setenv("GT_COMMAND", "")

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"/usr/local/bin/gt", "version"}

	got := Name()
	if got != "gt" {
		t.Errorf("Name() = %q, want %q when invoked as gt", got, "gt")
	}
}

// TestName_GTCommandEnvWinsOverArgv0 proves explicit GT_COMMAND still takes
// precedence over argv[0] detection — an operator who sets GT_COMMAND
// explicitly should not be overridden by how the binary happens to be
// symlinked.
func TestName_GTCommandEnvWinsOverArgv0(t *testing.T) {
	nameOnce = sync.Once{}
	name = ""
	t.Setenv("GT_COMMAND", "custom-name")

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"/usr/local/bin/gt", "version"}

	got := Name()
	if got != "custom-name" {
		t.Errorf("Name() = %q, want %q (GT_COMMAND should win over argv[0])", got, "custom-name")
	}
}

func TestName_OnceSemantics(t *testing.T) {
	nameOnce = sync.Once{}
	name = ""
	t.Setenv("GT_COMMAND", "first")

	first := Name()
	if first != "first" {
		t.Fatalf("Name() = %q, want %q", first, "first")
	}

	// Changing env after first call should have no effect (sync.Once)
	t.Setenv("GT_COMMAND", "second")
	second := Name()
	if second != "first" {
		t.Errorf("Name() returned %q after env change, want %q (sync.Once should cache)", second, "first")
	}
}

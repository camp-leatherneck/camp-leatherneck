// Package cli provides CLI configuration utilities.
package cli

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	name     string
	nameOnce sync.Once
)

// Name returns the Camp Leatherneck CLI command name.
// Defaults to "lt", but can be overridden with GT_COMMAND env var, or by
// invoking the same binary via its "gt" shim name (see IsInvokedAsGT) —
// this keeps help/usage text consistent with how the user actually typed
// the command, without requiring a second binary or a hand-maintained copy.
func Name() string {
	nameOnce.Do(func() {
		name = os.Getenv("GT_COMMAND")
		if name == "" && IsInvokedAsGT() {
			name = "gt"
		}
		if name == "" {
			name = "lt"
		}
	})
	return name
}

// IsInvokedAsGT reports whether this process was invoked via argv[0] == "gt"
// (the generated deprecation shim — see `make install` and Execute() in
// internal/cmd/root.go). Camp Leatherneck's canonical CLI is `lt`; `gt` is
// the same binary and the same commit, installed as a second filename
// pointing at it, kept only for compatibility with existing call sites and
// muscle memory. There is no separate `gt` binary or codebase — checking
// argv[0] is how one binary answers to two names without a hand-maintained
// second copy that could drift.
func IsInvokedAsGT() bool {
	if len(os.Args) == 0 {
		return false
	}
	return filepath.Base(os.Args[0]) == "gt"
}

// Package docaudit is a narrow, deliberately small guard against one
// specific documentation-drift failure mode found during the Camp
// Leatherneck architectural redesign (hq-lhy1, 2026-07-31):
// docs/design/model-aware-molecules.md marked a checklist item
// "- [x] Create `internal/models/database.go`" done when the file never
// existed. This package does not attempt general documentation linting —
// it checks exactly one thing: a checked box in docs/design/*.md that
// names a backtick-quoted repo-relative path must point at something that
// exists. See docaudit_test.go for the enforcement.
package docaudit

import (
	"os"
	"path/filepath"
	"regexp"
)

// checkedBoxPathPattern matches a completed checklist item that names a
// single backtick-quoted repo-relative path, e.g.:
//
//	- [x] Create `internal/models/database.go` — static benchmarks...
//
// Deliberately narrow: only fires on the exact shape that caused the known
// incident (checked box + one backtick path near the start of the line),
// not on every backtick span in a document.
var checkedBoxPathPattern = regexp.MustCompile("(?m)^-\\s*\\[x\\]\\s*(?:Create|Add|Implement|Build)\\s*`([^`]+)`")

// FindRepoRoot walks up from dir looking for go.mod.
func FindRepoRoot(dir string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// Violation is one checked-box claim whose named path does not exist.
type Violation struct {
	DocPath   string
	ClaimedPath string
}

// CheckDesignDocs scans docRoot/docs/design/*.md (non-recursive at the top
// level, but subdirectories are included) for checked-box claims naming a
// path that doesn't exist under repoRoot. Returns all violations found.
func CheckDesignDocs(repoRoot string) ([]Violation, error) {
	designDir := filepath.Join(repoRoot, "docs", "design")
	var violations []Violation
	err := filepath.Walk(designDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := checkedBoxPathPattern.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			claimed := m[1]
			if _, err := os.Stat(filepath.Join(repoRoot, claimed)); os.IsNotExist(err) {
				rel, _ := filepath.Rel(repoRoot, path)
				violations = append(violations, Violation{DocPath: rel, ClaimedPath: claimed})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return violations, nil
}

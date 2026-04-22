package git_test

import (
	"github.com/camp-leatherneck/camp-leatherneck/internal/beads"
	"github.com/camp-leatherneck/camp-leatherneck/internal/git"
)

// Compile-time assertion: Git must satisfy BranchChecker.
var _ beads.BranchChecker = (*git.Git)(nil)

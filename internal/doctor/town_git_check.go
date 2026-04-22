package doctor

import (
	"os"
	"path/filepath"
)

// TownGitCheck verifies that the HQ root directory is under version control.
// Having the town harness in git is optional but recommended for:
// - Backing up personal Camp Leatherneck configuration and operating history
// - Tracking mail and coordination beads
// - Easier federation across machines
type TownGitCheck struct {
	BaseCheck
}

// NewTownGitCheck creates a new town git version control check.
func NewTownGitCheck() *TownGitCheck {
	return &TownGitCheck{
		BaseCheck: BaseCheck{
			CheckName:        "town-git",
			CheckDescription: "Verify HQ root is under version control",
			CheckCategory:    CategoryCore,
		},
	}
}

// Run checks if the HQ root has a .git directory.
func (c *TownGitCheck) Run(ctx *CheckContext) *CheckResult {
	gitDir := filepath.Join(ctx.TownRoot, ".git")
	info, err := os.Stat(gitDir)

	if os.IsNotExist(err) {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: "HQ root is not under version control",
			Details: []string{
				"Your town harness contains personal configuration and operating history",
				"Version control makes it easier to backup and federate across machines",
			},
			FixHint: "Run 'git init' in your HQ root to initialize a repository",
		}
	}

	if err != nil {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusError,
			Message: "Failed to check town git status: " + err.Error(),
		}
	}

	// Verify it's actually a directory (not a file named .git)
	if !info.IsDir() {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: "HQ root .git is not a directory",
			Details: []string{
				"Expected .git to be a directory, but it's a file",
				"This may indicate a git worktree or submodule configuration",
			},
		}
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusOK,
		Message: "HQ root is under version control",
	}
}

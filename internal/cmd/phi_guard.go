package cmd

import (
	"github.com/camp-leatherneck/camp-leatherneck/internal/phi"
)

// phiGuardForBead enforces the PHI containment contract before any bead
// content is handed to an agent session. It must be called unconditionally
// at every dispatch entry point, immediately after bead info is resolved
// and before any --force/--agent override logic runs — those flags govern
// re-dispatch and runtime selection, never PHI classification.
//
// See ~/gt/directives/lt.md "PHI Containment Contract" for the policy this
// enforces, and internal/phi for the classification logic and tests.
func phiGuardForBead(townRoot string, info *beadInfo) error {
	policy, err := phi.LoadPolicy(townRoot)
	if err != nil {
		// Fail closed on a broken policy file too — an operator error in
		// the override should never silently widen what's allowed through.
		return err
	}
	item := phi.WorkItem{
		Title:       info.Title,
		Description: info.Description,
		Labels:      info.Labels,
	}
	return phi.Guard(item, policy)
}

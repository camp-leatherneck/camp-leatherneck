package provenance

import (
	"fmt"
	"os"

	"github.com/camp-leatherneck/camp-leatherneck/internal/config"
	"github.com/camp-leatherneck/camp-leatherneck/internal/doctor"
)

// SourceRoot is the camp-leatherneck source checkout path this check
// validates the running binary's commit against. Overridable for tests;
// defaults to the conventional location.
var SourceRoot = ""

func defaultSourceRoot() string {
	if SourceRoot != "" {
		return SourceRoot
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/camp-leatherneck"
}

// Check implements doctor.Check. This is the entire footprint in
// internal/cmd/doctor.go per C2: one import, one Register call.
type Check struct {
	doctor.BaseCheck
	lastReport *Report
}

// NewCheck creates the provenance doctor check.
func NewCheck() *Check {
	return &Check{
		BaseCheck: doctor.BaseCheck{
			CheckName:        "provenance",
			CheckDescription: "Verify launchd declared program == running daemon == PATH lt == known commit (Constitution §7)",
			CheckCategory:    doctor.CategoryCore,
		},
	}
}

func (c *Check) Run(ctx *doctor.CheckContext) *doctor.CheckResult {
	roles := config.AllRoles()
	report := Gather(ctx.TownRoot, defaultSourceRoot(), roles)
	c.lastReport = report

	violations := report.Violations()
	warnings := report.Warnings()

	result := &doctor.CheckResult{
		Name:     c.CheckName,
		Category: c.CheckCategory,
	}

	switch {
	case len(violations) > 0:
		result.Status = doctor.StatusError
		result.Message = fmt.Sprintf("%d provenance violation(s)", len(violations))
		result.Details = append(append([]string{}, violations...), warnings...)
	case len(warnings) > 0:
		result.Status = doctor.StatusWarning
		result.Message = fmt.Sprintf("%d provenance warning(s)", len(warnings))
		result.Details = warnings
	default:
		result.Status = doctor.StatusOK
		result.Message = "binary, daemon, launchd, and directive provenance all consistent"
	}

	return result
}

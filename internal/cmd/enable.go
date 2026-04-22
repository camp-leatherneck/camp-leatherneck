// ABOUTME: Command to enable Camp Leatherneck system-wide.
// ABOUTME: Sets the global state to enabled for all agentic coding tools.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/camp-leatherneck/camp-leatherneck/internal/state"
	"github.com/camp-leatherneck/camp-leatherneck/internal/style"
)

var enableCmd = &cobra.Command{
	Use:     "enable",
	GroupID: GroupConfig,
	Short:   "Enable Camp Leatherneck system-wide",
	Long: `Enable Camp Leatherneck for all agentic coding tools.

When enabled:
  - Shell hooks set GT_TOWN_ROOT and GT_RIG environment variables
  - Claude Code SessionStart hooks run 'lt prime' for context
  - Git repos are auto-registered as rigs (configurable)

Use 'lt disable' to turn off. Use 'lt status' to check state.

Environment overrides:
  GASTOWN_DISABLED=1  - Disable for current session only
  GASTOWN_ENABLED=1   - Enable for current session only`,
	RunE: runEnable,
}

func init() {
	rootCmd.AddCommand(enableCmd)
}

func runEnable(cmd *cobra.Command, args []string) error {
	if err := state.Enable(Version); err != nil {
		return fmt.Errorf("enabling Camp Leatherneck: %w", err)
	}

	fmt.Printf("%s Camp Leatherneck enabled\n", style.Success.Render("✓"))
	fmt.Println()
	fmt.Println("Camp Leatherneck will now:")
	fmt.Println("  • Inject context into Claude Code sessions")
	fmt.Println("  • Set GT_TOWN_ROOT and GT_RIG environment variables")
	fmt.Println("  • Auto-register git repos as rigs (if configured)")
	fmt.Println()
	fmt.Printf("Use %s to disable, %s to check status\n",
		style.Dim.Render("lt disable"),
		style.Dim.Render("lt status"))

	return nil
}

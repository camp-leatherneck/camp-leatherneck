package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/camp-leatherneck/camp-leatherneck/internal/config"
	"github.com/camp-leatherneck/camp-leatherneck/internal/roster"
)

var (
	rosterCSV  bool
	rosterJSON bool
)

var rosterCmd = &cobra.Command{
	Use:         "roster",
	GroupID:     GroupDiag,
	Annotations: map[string]string{AnnotationPolecatSafe: "true"},
	Short:       "Show the Camp Leatherneck roster (rank, persona, scope, cardinality)",
	Long: `Render the roster of every known role: persona name, USMC rank,
scope (town/rig/ephemeral), cardinality (singular/multi), and how it's
spawned.

The roster is generated on demand from two versioned sources (Architecture
Constitution §6, Certification C4) — directive front-matter for roles that
have a directive file, and a small explicit list for roles that don't
(Dog, Crew). There is no separate roster file to go stale; this command IS
the roster.

If either source contradicts the other, or a known role has no entry in
either source, this command exits with an error rather than silently
printing an incomplete roster — that silent-omission failure mode is
exactly what the old, unversioned Desktop CSV roster had.

Markdown output by default. Use --csv or --json for machine-readable output.`,
	RunE: runRoster,
}

func init() {
	rosterCmd.Flags().BoolVar(&rosterCSV, "csv", false, "Output as CSV")
	rosterCmd.Flags().BoolVar(&rosterJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(rosterCmd)
}

func runRoster(cmd *cobra.Command, args []string) error {
	if rosterCSV && rosterJSON {
		return fmt.Errorf("--csv and --json are mutually exclusive")
	}

	report := roster.Build(config.AllRoles())
	if !report.OK() {
		fmt.Fprintln(os.Stderr, "lt roster: refusing to print an incomplete or contradictory roster:")
		for _, e := range report.Errors {
			fmt.Fprintln(os.Stderr, "  - "+e)
		}
		return fmt.Errorf("%d roster error(s)", len(report.Errors))
	}

	switch {
	case rosterJSON:
		return printRosterJSON(report)
	case rosterCSV:
		return printRosterCSV(report)
	default:
		printRosterMarkdown(report)
		return nil
	}
}

func printRosterMarkdown(report *roster.Report) {
	fmt.Println("# Camp Leatherneck Roster")
	fmt.Println()
	fmt.Println("| Role | Persona | Rank | Scope | Cardinality | Spawned by |")
	fmt.Println("|---|---|---|---|---|---|")
	for _, e := range report.Entities {
		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
			e.Role, e.Persona, e.Rank, e.Scope, e.Cardinality, e.SpawnedBy)
	}
}

func printRosterCSV(report *roster.Report) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	header := []string{"role", "persona", "rank", "scope", "cardinality", "spawned_by", "directive_file", "source"}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, e := range report.Entities {
		row := []string{e.Role, e.Persona, e.Rank, e.Scope, e.Cardinality, e.SpawnedBy, e.DirectiveFile, e.Source}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func printRosterJSON(report *roster.Report) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report.Entities)
}

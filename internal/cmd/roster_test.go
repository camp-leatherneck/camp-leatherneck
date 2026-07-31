package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/camp-leatherneck/camp-leatherneck/internal/roster"
)

func TestRunRoster_MutuallyExclusiveFlags(t *testing.T) {
	origCSV, origJSON := rosterCSV, rosterJSON
	defer func() { rosterCSV, rosterJSON = origCSV, origJSON }()

	rosterCSV = true
	rosterJSON = true
	err := runRoster(nil, nil)
	if err == nil {
		t.Fatal("expected an error when both --csv and --json are set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPrintRosterMarkdown_ProducesATable(t *testing.T) {
	report := &roster.Report{
		Entities: []roster.Entity{
			{Role: "mayor", Persona: "LT", Rank: "O-1", Scope: "town", Cardinality: "singular", SpawnedBy: "gt prime"},
		},
	}
	out := captureStdout(t, func() { printRosterMarkdown(report) })
	if !strings.Contains(out, "| mayor | LT | O-1 | town | singular | gt prime |") {
		t.Errorf("markdown output missing expected row:\n%s", out)
	}
	if !strings.Contains(out, "# Camp Leatherneck Roster") {
		t.Errorf("markdown output missing header:\n%s", out)
	}
}

func TestPrintRosterCSV_RoundTrips(t *testing.T) {
	report := &roster.Report{
		Entities: []roster.Entity{
			{Role: "mayor", Persona: "LT", Rank: "O-1", Scope: "town", Cardinality: "singular", SpawnedBy: "gt prime", DirectiveFile: "lt", Source: "directive"},
			{Role: "dog", Persona: "Dog", Rank: "MWD", Scope: "town", Cardinality: "multi", SpawnedBy: "gt deacon", Source: "explicit"},
		},
	}
	out := captureStdout(t, func() {
		if err := printRosterCSV(report); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Fatalf("expected 3 CSV lines (header + 2 rows), got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "role,persona,rank,scope,cardinality,spawned_by,directive_file,source") {
		t.Errorf("unexpected CSV header: %q", lines[0])
	}
}

func TestPrintRosterJSON_ValidJSON(t *testing.T) {
	report := &roster.Report{
		Entities: []roster.Entity{
			{Role: "mayor", Persona: "LT"},
		},
	}
	out := captureStdout(t, func() {
		if err := printRosterJSON(report); err != nil {
			t.Fatal(err)
		}
	})
	var entities []roster.Entity
	if err := json.Unmarshal([]byte(out), &entities); err != nil {
		t.Fatalf("printRosterJSON produced invalid JSON: %v\noutput: %s", err, out)
	}
	if len(entities) != 1 || entities[0].Role != "mayor" {
		t.Errorf("unexpected decoded entities: %+v", entities)
	}
}

// TestRunRoster_RealBuild_Succeeds is the integration proof that, with
// front-matter now present on all 20 directive templates plus the two
// explicit entities, the real roster builds cleanly end to end — no
// missing roles, no contradictions.
func TestRunRoster_RealBuild_Succeeds(t *testing.T) {
	origCSV, origJSON := rosterCSV, rosterJSON
	defer func() { rosterCSV, rosterJSON = origCSV, origJSON }()
	rosterCSV, rosterJSON = false, false

	out := captureStdout(t, func() {
		if err := runRoster(nil, nil); err != nil {
			t.Fatalf("runRoster() failed against the real embedded directive set: %v", err)
		}
	})
	if !strings.Contains(out, "mayor") {
		t.Errorf("expected 'mayor' role in real roster output:\n%s", out)
	}
	if !strings.Contains(out, "dog") || !strings.Contains(out, "crew") {
		t.Errorf("expected 'dog' and 'crew' (directive-less, C4) in real roster output:\n%s", out)
	}
}

package roster

import (
	"strings"
	"testing"
)

func TestParseFrontMatter_NoFrontMatter(t *testing.T) {
	_, has, err := parseFrontMatter("# LT\n\nYou are LT.\n")
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected hasFrontMatter=false for content with no front-matter block")
	}
}

func TestParseFrontMatter_ValidBlock(t *testing.T) {
	content := "+++\n" +
		"rank = \"LT (O-1)\"\n" +
		"persona = \"lt\"\n" +
		"scope = \"town\"\n" +
		"cardinality = \"singular\"\n" +
		"spawned_by = \"gt prime\"\n" +
		"+++\n" +
		"# LT\n\nbody\n"
	fm, has, err := parseFrontMatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected hasFrontMatter=true")
	}
	if fm.Rank != "LT (O-1)" || fm.Persona != "lt" || fm.Scope != "town" || fm.Cardinality != "singular" || fm.SpawnedBy != "gt prime" {
		t.Errorf("parsed front-matter = %+v, fields don't match input", fm)
	}
}

func TestParseFrontMatter_MalformedTOML_Errors(t *testing.T) {
	content := "+++\nrank = not valid toml [[[\n+++\nbody\n"
	_, has, err := parseFrontMatter(content)
	if !has {
		t.Fatal("expected hasFrontMatter=true even though parsing fails — the block was present, just malformed")
	}
	if err == nil {
		t.Fatal("expected an error for malformed TOML")
	}
}

func TestBuild_ExplicitEntitiesAlwaysPresent(t *testing.T) {
	// Regardless of directive front-matter state, dog and crew must always
	// resolve — this is the exact C4 failure class this package prevents.
	report := Build([]string{"dog", "crew"})
	byRole := map[string]Entity{}
	for _, e := range report.Entities {
		byRole[e.Role] = e
	}
	if _, ok := byRole["dog"]; !ok {
		t.Error("expected 'dog' to always be present in the roster (C4: directive-less entity)")
	}
	if _, ok := byRole["crew"]; !ok {
		t.Error("expected 'crew' to always be present in the roster (C4: directive-less entity)")
	}
	if byRole["dog"].Source != "explicit" || byRole["crew"].Source != "explicit" {
		t.Errorf("expected dog/crew Source=explicit, got dog=%q crew=%q", byRole["dog"].Source, byRole["crew"].Source)
	}
}

func TestBuild_MissingRoleFailsLoudly(t *testing.T) {
	// A role that exists in neither source must produce a loud error, not
	// a silently incomplete roster — the exact failure mode of the old
	// Desktop CSV, reproduced on purpose here to prove the guard.
	report := Build([]string{"a-role-that-will-never-exist"})
	if report.OK() {
		t.Fatal("expected Build to report an error for an unresolvable role, got none")
	}
	found := false
	for _, e := range report.Errors {
		if strings.Contains(e, "a-role-that-will-never-exist") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error mentioning the missing role, got: %v", report.Errors)
	}
}

func TestBuild_DoesNotFlagDirectiveLessRolesAsMissing(t *testing.T) {
	// dog/crew are resolved via explicitEntities, not directive front-matter
	// — Build must not additionally complain about them lacking a directive.
	report := Build([]string{"dog", "crew"})
	for _, e := range report.Errors {
		if strings.Contains(e, "dog") || strings.Contains(e, "crew") {
			t.Errorf("unexpected error mentioning a directive-less role: %q", e)
		}
	}
}

// TestBuild_RealEmbeddedDirectives exercises Build against the actual
// embedded directive set (assets.DirectivesFS), not a fixture — this is
// the integration proof that the roster reflects the real, shipped source
// templates.
func TestBuild_RealEmbeddedDirectives(t *testing.T) {
	report := Build([]string{"mayor", "deacon", "witness", "refinery", "polecat", "dog", "crew"})
	// dog and crew must always resolve regardless of directive state.
	byRole := map[string]Entity{}
	for _, e := range report.Entities {
		byRole[e.Role] = e
	}
	if _, ok := byRole["dog"]; !ok {
		t.Error("expected 'dog' in the real roster build")
	}
	if _, ok := byRole["crew"]; !ok {
		t.Error("expected 'crew' in the real roster build")
	}
}

func TestEntity_DirectiveLessRolesMap_MatchesExplicitEntities(t *testing.T) {
	for _, e := range explicitEntities {
		if !directiveLessRoles[e.Role] {
			t.Errorf("explicitEntities contains role %q but directiveLessRoles doesn't list it — these two must stay in sync", e.Role)
		}
	}
	for role := range directiveLessRoles {
		found := false
		for _, e := range explicitEntities {
			if e.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("directiveLessRoles lists role %q but explicitEntities has no matching entry", role)
		}
	}
}

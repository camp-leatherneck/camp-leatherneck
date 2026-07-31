// Package roster implements the Camp Leatherneck roster authority per
// Constitution §6 and its C4 certification ruling: two versioned sources —
// directive front-matter for entities that have a directive, and a small
// explicit list for entities that don't (Dog, Crew) — merged deterministically,
// failing loudly on duplicates, conflicts, missing required entities, or
// invalid references. Neither source lives outside version control, and
// nothing silently omits an entity, which is exactly the failure mode the
// stale, unversioned Desktop CSV roster had.
package roster

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/camp-leatherneck/camp-leatherneck/internal/assets"
)

// Entity is one roster entry — a persona or role, whether or not it has a
// directive file.
type Entity struct {
	Role          string // internal role identifier (mayor, deacon, recon, dog, crew, ...)
	Persona       string // display persona name from front-matter, or Role if absent
	Rank          string
	Scope         string // town | rig | ephemeral
	Cardinality   string // singular | multi
	SpawnedBy     string
	DirectiveFile string // "" for directive-less entities
	Source        string // "directive" | "explicit"
}

// frontMatter is the TOML shape parsed from a directive's front-matter block.
type frontMatter struct {
	Rank        string `toml:"rank"`
	Persona     string `toml:"persona"`
	Scope       string `toml:"scope"`
	Cardinality string `toml:"cardinality"`
	SpawnedBy   string `toml:"spawned_by"`
}

// validScopes and validCardinalities gate front-matter values — a typo here
// should fail loudly at roster-build time, not surface as a silently wrong
// roster entry.
var validScopes = map[string]bool{"town": true, "rig": true, "ephemeral": true}
var validCardinalities = map[string]bool{"singular": true, "multi": true}

// explicitEntities is the second roster source (C4): entities with no
// directive file. Kept as a small Go literal, not a separate file — per
// the certification's "keep this small enough to replace later" guidance,
// two entries doesn't justify a second embedded asset or config format.
var explicitEntities = []Entity{
	{
		Role: "dog", Persona: "Dog", Rank: "MWD (attached)",
		Scope: "town", Cardinality: "multi", SpawnedBy: "gt deacon (patrol dispatch)",
		Source: "explicit",
	},
	{
		Role: "crew", Persona: "Crew", Rank: "n/a (long-lived contributor)",
		Scope: "rig", Cardinality: "multi", SpawnedBy: "gt crew add",
		Source: "explicit",
	},
}

// coreRoleFiles mirrors internal/config's unexported roleToFileName for the
// roles that map to a Marine-named directive file. See the identical
// comment in internal/provenance/provenance.go for why this is duplicated
// rather than imported (roleToFileName is unexported), and how drift is
// caught (TestBuildReport_CoreRolesMatchExpectedFiles below).
var coreRoleFiles = map[string]string{
	"mayor":    "lt",
	"deacon":   "top",
	"witness":  "sarge",
	"refinery": "gunny",
	"polecat":  "marine",
}

// directiveLessRoles are core Gastown roles (from config.AllRoles()) that
// are legitimately directive-less — their metadata comes only from
// explicitEntities, never from a missing directive file being treated as
// an error.
var directiveLessRoles = map[string]bool{"dog": true, "crew": true}

// Report is the outcome of building the roster: the merged entities, plus
// any errors that should fail the build loudly rather than produce a
// silently-incomplete roster.
type Report struct {
	Entities []Entity
	Errors   []string
}

// OK reports whether the roster built cleanly.
func (r *Report) OK() bool {
	return len(r.Errors) == 0
}

// Build constructs the full roster from both sources: front-matter parsed
// from the embedded directive templates (assets.DirectivesFS), and the
// explicit directive-less list. allRoles is normally config.AllRoles() —
// passed as a parameter so tests can control it precisely.
func Build(allRoles []string) *Report {
	report := &Report{}

	fromDirectives, directiveErrors := buildFromDirectives()
	report.Errors = append(report.Errors, directiveErrors...)

	seen := map[string]Entity{}
	var order []string
	addEntity := func(e Entity) {
		if existing, dup := seen[e.Role]; dup {
			report.Errors = append(report.Errors, fmt.Sprintf(
				"role %q defined more than once (sources: %s and %s) — conflicting roster entries",
				e.Role, existing.Source, e.Source))
			return
		}
		seen[e.Role] = e
		order = append(order, e.Role)
	}

	for _, e := range fromDirectives {
		addEntity(e)
	}
	for _, e := range explicitEntities {
		addEntity(e)
	}

	// Verify every core role (config.AllRoles()) resolved to exactly one
	// entity, from exactly one source, with no silent omission. This is
	// the direct fix for the failure class this package exists to prevent:
	// a roster built only from directive front-matter would drop Dog and
	// Crew without warning.
	for _, role := range allRoles {
		if _, ok := seen[role]; !ok {
			report.Errors = append(report.Errors,
				fmt.Sprintf("role %q (from config.AllRoles()) has no roster entry in either source — invalid reference", role))
		}
	}

	sort.Strings(order)
	for _, role := range order {
		report.Entities = append(report.Entities, seen[role])
	}

	return report
}

// buildFromDirectives parses front-matter from every embedded directive
// template. A directive with no front-matter contributes no roster entity
// (it's not an error — not every ephemeral specialist necessarily needs
// roster metadata yet, and this must not regress §C3's "front-matter is
// optional" property into "front-matter is mandatory").
func buildFromDirectives() ([]Entity, []string) {
	var entities []Entity
	var errs []string

	entries, err := assets.DirectivesFS.ReadDir("directives")
	if err != nil {
		return nil, []string{"reading embedded directives: " + err.Error()}
	}

	fileToRole := map[string]string{}
	for role, file := range coreRoleFiles {
		fileToRole[file] = role
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := assets.DirectivesFS.ReadFile("directives/" + entry.Name())
		if err != nil {
			errs = append(errs, fmt.Sprintf("reading %s: %v", entry.Name(), err))
			continue
		}

		fm, hasFrontMatter, err := parseFrontMatter(string(data))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: malformed front-matter: %v", entry.Name(), err))
			continue
		}
		if !hasFrontMatter {
			continue
		}

		baseName := strings.TrimSuffix(entry.Name(), ".md")
		role := baseName
		if r, ok := fileToRole[baseName]; ok {
			role = r
		}

		if fm.Scope != "" && !validScopes[fm.Scope] {
			errs = append(errs, fmt.Sprintf("%s: invalid scope %q (want town, rig, or ephemeral)", entry.Name(), fm.Scope))
			continue
		}
		if fm.Cardinality != "" && !validCardinalities[fm.Cardinality] {
			errs = append(errs, fmt.Sprintf("%s: invalid cardinality %q (want singular or multi)", entry.Name(), fm.Cardinality))
			continue
		}

		persona := fm.Persona
		if persona == "" {
			persona = baseName
		}

		entities = append(entities, Entity{
			Role:          role,
			Persona:       persona,
			Rank:          fm.Rank,
			Scope:         fm.Scope,
			Cardinality:   fm.Cardinality,
			SpawnedBy:     fm.SpawnedBy,
			DirectiveFile: baseName,
			Source:        "directive",
		})
	}

	return entities, errs
}

// parseFrontMatter extracts and parses a leading TOML front-matter block.
// Returns hasFrontMatter=false (no error) when the content has none — this
// mirrors internal/config.stripFrontMatter's detection logic exactly, since
// the two must agree on what counts as front-matter.
func parseFrontMatter(content string) (fm frontMatter, hasFrontMatter bool, err error) {
	trimmed := strings.TrimLeft(content, "\r\n\t ")
	const delim = "+++"
	if !strings.HasPrefix(trimmed, delim) {
		return fm, false, nil
	}
	afterOpen := trimmed[len(delim):]
	nl := strings.IndexByte(afterOpen, '\n')
	if nl == -1 {
		return fm, false, nil
	}
	if strings.TrimRight(afterOpen[:nl], "\r \t") != "" {
		return fm, false, nil // "+++something", not a real delimiter
	}
	afterOpenLine := afterOpen[nl+1:]

	var block string
	if strings.HasPrefix(afterOpenLine, delim) {
		block = "" // empty front-matter block
	} else {
		closeIdx := strings.Index(afterOpenLine, "\n"+delim)
		if closeIdx == -1 {
			return fm, false, nil // unclosed — treat as absent, matches stripFrontMatter
		}
		block = afterOpenLine[:closeIdx]
	}

	if err := toml.Unmarshal([]byte(block), &fm); err != nil {
		return fm, true, err
	}
	return fm, true, nil
}

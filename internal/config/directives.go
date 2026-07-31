package config

import (
	"os"
	"path/filepath"
	"strings"
)

// roleToFileName maps role identifiers to directive filenames.
// This enables renaming directive files to use Marine rank names while
// keeping role identifiers stable (mayor, deacon, etc.) in the codebase.
func roleToFileName(role string) string {
	// Map old CLI/internal role names to new Marine rank filenames
	switch role {
	case "mayor":
		return "lt"
	case "deacon":
		return "top"
	case "witness":
		return "sarge"
	case "refinery":
		return "gunny"
	case "polecat":
		return "marine"
	// Unchanged roles
	case "firewatch", "rto":
		return role
	default:
		// Default: use role name as-is for ephemeral specialists
		return role
	}
}

// LoadRoleDirective loads role directive content from the directive file layout.
// Resolution order:
//  1. HQ-level: <townRoot>/directives/<fileName>.md
//  2. Rig-level:  <townRoot>/<rigName>/directives/<fileName>.md
//
// The fileName is determined by mapping the role identifier to its directive filename.
// If both exist, they are concatenated (town first, then rig) separated by a
// newline, giving rig-level content the last word. If only one exists, that
// content is returned. Returns empty string if no directive files exist.
//
// Invalid or unreadable paths are treated as absent (no error).
//
// Front-matter (a leading TOML block delimited by "+++" lines, used to carry
// roster metadata — see internal/roster) is stripped before the content is
// returned. Per the Architecture Certification C3 ruling: front-matter must
// be invisible to the agent reading this directive, not merely non-crashing.
// A directive with no front-matter is returned byte-identical to how it was
// before front-matter existed as a concept — stripFrontMatter is a no-op
// when the leading "+++" marker isn't present.
func LoadRoleDirective(role, townRoot, rigName string) string {
	var parts []string

	fileName := roleToFileName(role)

	// HQ-level directive
	townPath := filepath.Join(townRoot, "directives", fileName+".md")
	if content, err := os.ReadFile(townPath); err == nil { //nolint:gosec // G304: path is from trusted config
		if s := strings.TrimSpace(stripFrontMatter(string(content))); s != "" {
			parts = append(parts, s)
		}
	}

	// Rig-level directive (wins by appearing last)
	if rigName != "" {
		rigPath := filepath.Join(townRoot, rigName, "directives", fileName+".md")
		if content, err := os.ReadFile(rigPath); err == nil { //nolint:gosec // G304: path is from trusted config
			if s := strings.TrimSpace(stripFrontMatter(string(content))); s != "" {
				parts = append(parts, s)
			}
		}
	}

	return strings.Join(parts, "\n")
}

// frontMatterDelim is the delimiter line for the TOML front-matter block
// this package recognizes. Matches the convention already used by plugin.md
// files elsewhere in this repo (e.g. plugins/*/plugin.md).
const frontMatterDelim = "+++"

// stripFrontMatter removes a leading "+++\n...\n+++" TOML front-matter
// block from directive content, if present. Content with no front-matter
// is returned completely unchanged — this is what makes front-matter
// optional and backward-compatible: a directive written before front-matter
// existed, or one an operator chooses never to add it to, behaves exactly
// as it always did.
//
// Malformed front-matter (an opening "+++" with no matching close) is
// treated as absent rather than eating the rest of the file — a directive
// that happens to start a line with "+++" for some other reason (e.g. a
// markdown horizontal-rule-like flourish) degrades to "front-matter not
// found," not silent content loss.
func stripFrontMatter(content string) string {
	trimmedLeading := strings.TrimLeft(content, "\r\n\t ")
	if !strings.HasPrefix(trimmedLeading, frontMatterDelim) {
		return content
	}
	afterOpen := trimmedLeading[len(frontMatterDelim):]
	// The rest of the opening delimiter's line must be empty (allow
	// trailing whitespace) — "+++" must be alone on its line, not e.g.
	// "+++foo" or a markdown "***" divider that happens to share a prefix.
	newlineIdx := strings.IndexByte(afterOpen, '\n')
	if newlineIdx == -1 {
		return content
	}
	restOfOpenLine := strings.TrimRight(afterOpen[:newlineIdx], "\r \t")
	if restOfOpenLine != "" {
		return content
	}
	afterOpenLine := afterOpen[newlineIdx+1:]

	closeIdx := strings.Index(afterOpenLine, "\n"+frontMatterDelim)
	var closeMarkerLen int
	if strings.HasPrefix(afterOpenLine, frontMatterDelim) {
		// Front-matter block with zero content lines: "+++\n+++\nbody".
		closeIdx = 0
		closeMarkerLen = len(frontMatterDelim)
	} else if closeIdx == -1 {
		return content // no closing delimiter found — treat as no front-matter
	} else {
		closeMarkerLen = 1 + len(frontMatterDelim) // the leading "\n" plus "+++"
	}

	afterClose := afterOpenLine[closeIdx+closeMarkerLen:]
	// Consume the rest of the closing delimiter's line too.
	if nl := strings.IndexByte(afterClose, '\n'); nl != -1 {
		return afterClose[nl+1:]
	}
	return ""
}

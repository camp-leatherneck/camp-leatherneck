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
func LoadRoleDirective(role, townRoot, rigName string) string {
	var parts []string

	fileName := roleToFileName(role)

	// HQ-level directive
	townPath := filepath.Join(townRoot, "directives", fileName+".md")
	if content, err := os.ReadFile(townPath); err == nil { //nolint:gosec // G304: path is from trusted config
		if s := strings.TrimSpace(string(content)); s != "" {
			parts = append(parts, s)
		}
	}

	// Rig-level directive (wins by appearing last)
	if rigName != "" {
		rigPath := filepath.Join(townRoot, rigName, "directives", fileName+".md")
		if content, err := os.ReadFile(rigPath); err == nil { //nolint:gosec // G304: path is from trusted config
			if s := strings.TrimSpace(string(content)); s != "" {
				parts = append(parts, s)
			}
		}
	}

	return strings.Join(parts, "\n")
}

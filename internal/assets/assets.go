// Package assets embeds the Camp Leatherneck overlay runtime assets —
// Marine-flavored role directives, the RTO sitrep script, and the launchd
// plist template for scheduling RTO — into the lt binary so `lt install`
// can bootstrap a fresh machine's ~/lt/ tree without requiring network
// access or a separate asset download.
//
// Source layout (siblings of this file):
//
//	internal/assets/directives/*.md                20 role directives
//	internal/assets/scripts/rto.sh                 RTO sitrep generator
//	internal/assets/launchagents/*.tmpl            launchd plist template(s)
//
// Paths inside the plist template use the literal placeholder {{HOME}};
// callers are expected to substitute the current user's $HOME before writing.
//
// Sitrep.app bundle is intentionally NOT bundled — it lives on the
// installing user's Desktop as a local artifact. Future bead hq-3a8 (and
// related work) will address first-class app distribution once a code-
// signing identity is available.
package assets

import "embed"

// DirectivesFS holds the 20 canonical role directive markdown files.
//
//go:embed directives/*.md
var DirectivesFS embed.FS

// ScriptsFS holds runtime shell scripts (rto.sh, etc.).
//
//go:embed scripts/*.sh
var ScriptsFS embed.FS

// LaunchAgentsFS holds launchd plist templates. Files carry a .tmpl
// extension and contain {{HOME}} placeholders that the installer rewrites.
//
//go:embed launchagents/*.tmpl
var LaunchAgentsFS embed.FS

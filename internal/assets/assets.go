// Package assets embeds the Camp Leatherneck overlay runtime assets —
// Marine-flavored role directives, the RTO sitrep script, the launchd plist
// template for scheduling RTO, and the Sitrep.app observation-post bundle —
// into the lt binary so `lt install` can bootstrap a fresh machine's ~/lt/
// tree without requiring network access or a separate asset download.
//
// Source layout (siblings of this file):
//
//	internal/assets/directives/*.md                20 role directives
//	internal/assets/scripts/rto.sh                 RTO sitrep generator
//	internal/assets/launchagents/*.tmpl            launchd plist template(s)
//	internal/assets/apps/Sitrep.app/...            .app bundle
//
// Paths inside the plist template use the literal placeholder {{HOME}};
// callers are expected to substitute the current user's $HOME before writing.
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

// SitrepAppFS holds the Sitrep.app bundle, including nested
// Contents/MacOS/Sitrep and Contents/Resources/AppIcon.icns. The embed
// directive with `all:` ensures files beginning with `_` or `.` (none
// expected today, but future-proofing) are still embedded.
//
//go:embed all:apps/Sitrep.app
var SitrepAppFS embed.FS

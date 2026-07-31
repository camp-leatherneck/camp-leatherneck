package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/camp-leatherneck/camp-leatherneck/internal/version"
)

// Version information - set at build time via ldflags
var (
	Version = "1.0.0"
	// Build can be set via ldflags at compile time
	Build = "dev"
	// Commit and Branch - the git revision the binary was built from (optional ldflag)
	Commit = ""
	Branch = ""
	// BuiltProperly is set to "1" by `make build`. If empty, the binary was built
	// with raw `go build` and is likely unsigned (will be killed on macOS).
	BuiltProperly = ""
	// BuildTime is set by `make build`'s ldflags (BUILD_TIME := date -u ...).
	// Was previously targeted by the Makefile's -X flag but never declared here,
	// so every build silently discarded it — found and fixed as part of the
	// Camp Leatherneck topology normalization (LT_IMPLEMENTATION_CONTRACT.md
	// Phase 2 item 6: `lt version --json` must report build time).
	BuildTime = ""
)

var versionVerbose bool
var versionShort bool
var versionJSON bool

// versionInfo is the provenance record `lt version --json` emits.
// Required by LT_IMPLEMENTATION_CONTRACT.md Phase 2 item 6, and consumed
// as one input into `lt doctor`'s provenance report (Phase 3 / C2) —
// added here, in the existing version command, rather than as a new
// command surface.
type versionInfo struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	Branch      string `json:"branch,omitempty"`
	Build       string `json:"build"`
	Dirty       bool   `json:"dirty"`
	BuildTime   string `json:"build_time,omitempty"`
	InstallPath string `json:"install_path"`
	SHA256      string `json:"sha256,omitempty"`
	GoVersion   string `json:"go_version"`
}

var versionCmd = &cobra.Command{
	Use:         "version",
	GroupID:     GroupDiag,
	Annotations: map[string]string{AnnotationPolecatSafe: "true"},
	Short:       "Print version information",
	Long: `Print the lt version, build type, git branch, and commit hash.

Output includes the semantic version, whether this is a dev or release build,
and the git revision the binary was built from (if available).

--json emits full provenance: commit, dirty flag, build time, the real path
of the running executable, and its sha256 — the data lt doctor uses to
answer "is this the binary I think it is?".`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionJSON {
			runVersionJSON()
			return
		}

		if versionShort {
			fmt.Printf("%s-%s\n", Version, Build)
			return
		}

		commit := resolveCommitHash()
		branch := resolveBranch()

		if commit != "" && branch != "" {
			fmt.Printf("lt version %s (%s: %s@%s)\n", Version, Build, branch, version.ShortCommit(commit))
		} else if commit != "" {
			fmt.Printf("lt version %s (%s: %s)\n", Version, Build, version.ShortCommit(commit))
		} else {
			fmt.Printf("lt version %s (%s)\n", Version, Build)
		}

		if versionVerbose {
			fmt.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
			fmt.Printf("Go version: %s\n", runtime.Version())
		}
	},
}

func runVersionJSON() {
	info := buildVersionInfo()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(info)
}

// buildVersionInfo is the pure (no I/O to stdout) half of `lt version --json`,
// separated out so it's testable without capturing process stdout.
func buildVersionInfo() versionInfo {
	info := versionInfo{
		Version:   Version,
		Commit:    resolveCommitHash(),
		Branch:    resolveBranch(),
		Build:     Build,
		Dirty:     strings.Contains(Version, "-dirty"),
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
	}

	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			info.InstallPath = resolved
		} else {
			info.InstallPath = exe
		}
		if sum, err := sha256File(info.InstallPath); err == nil {
			info.SHA256 = sum
		}
	}

	return info
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&versionVerbose, "verbose", "v", false, "Show extended version info including timestamp")
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "Output only the version number (e.g., 0.5.0-362)")
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "Output full provenance as JSON: commit, dirty flag, build time, install path, sha256")

	// Pass the build-time commit to the version package for stale binary checks
	if Commit != "" {
		version.SetCommit(Commit)
	}
}

func resolveCommitHash() string {
	if Commit != "" {
		return Commit
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				return setting.Value
			}
		}
	}

	return ""
}

func resolveBranch() string {
	if Branch != "" {
		return Branch
	}

	// Try to get branch from build info (build-time VCS detection)
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.branch" && setting.Value != "" {
				return setting.Value
			}
		}
	}

	// Fallback: try to get branch from git at runtime
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = "."
	if output, err := cmd.Output(); err == nil {
		if branch := strings.TrimSpace(string(output)); branch != "" && branch != "HEAD" {
			return branch
		}
	}

	return ""
}

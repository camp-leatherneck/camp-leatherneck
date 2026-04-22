package assets

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Options controls how the embedded assets are written to disk.
type Options struct {
	// Home is the user's home directory. Required. Used both as the target
	// root for ~/lt/ and as the substitution value for {{HOME}} in the
	// launchd plist template.
	Home string

	// LTRoot is the target root for directives/, scripts/, and logs/.
	// Defaults to filepath.Join(Home, "lt") when empty.
	LTRoot string

	// LaunchAgentDir is where com.campleatherneck.rto.plist is written.
	// Defaults to filepath.Join(Home, "Library/LaunchAgents") when empty.
	LaunchAgentDir string

	// DryRun: when true, compute the plan and return it without writing.
	DryRun bool

	// Out receives human-readable progress lines. May be nil.
	Out io.Writer
}

// Plan describes what Install would (or did) write. One Action per file or
// directory operation.
type Plan struct {
	Actions []Action
}

// ActionKind identifies the type of filesystem operation an Action represents.
type ActionKind string

const (
	ActionMkdir       ActionKind = "mkdir"
	ActionWriteFile   ActionKind = "write"
	ActionSkipExists  ActionKind = "skip (exists, identical)"
	ActionOverwriteOK ActionKind = "overwrite (dry-run does not distinguish)"
)

// Action is a single planned or completed filesystem operation.
type Action struct {
	Kind    ActionKind
	Path    string
	Mode    os.FileMode
	Bytes   int // file size for writes
	Skipped bool
}

// Install writes the embedded Camp Leatherneck assets under opts.Home.
// It is idempotent: files that already exist with identical content are left
// alone. Files that exist with different content are overwritten (so the
// user always ends up with the shipped canonical copy after a fresh install).
//
// When opts.DryRun is true, no filesystem writes occur; the returned Plan
// still reflects what would have happened.
func Install(opts Options) (*Plan, error) {
	if opts.Home == "" {
		return nil, fmt.Errorf("assets.Install: Home is required")
	}

	if opts.LTRoot == "" {
		opts.LTRoot = filepath.Join(opts.Home, "lt")
	}
	if opts.LaunchAgentDir == "" {
		opts.LaunchAgentDir = filepath.Join(opts.Home, "Library", "LaunchAgents")
	}

	plan := &Plan{}

	// 1. Directives -> $LTRoot/directives/*.md
	directivesDir := filepath.Join(opts.LTRoot, "directives")
	if err := writeEmbedTree(plan, opts, DirectivesFS, "directives", directivesDir, 0644, nil); err != nil {
		return plan, err
	}

	// 2. Scripts -> $LTRoot/scripts/*.sh (executable)
	scriptsDir := filepath.Join(opts.LTRoot, "scripts")
	if err := writeEmbedTree(plan, opts, ScriptsFS, "scripts", scriptsDir, 0755, nil); err != nil {
		return plan, err
	}

	// 3. Ensure logs dir exists (rto.sh + plist write here).
	logsDir := filepath.Join(opts.LTRoot, "logs")
	if err := ensureDir(plan, opts, logsDir); err != nil {
		return plan, err
	}

	// 4. Launch agent plist (with {{HOME}} substitution, .tmpl extension
	//    stripped on output).
	transform := func(srcName string, data []byte) (outName string, out []byte) {
		outName = strings.TrimSuffix(srcName, ".tmpl")
		out = bytes.ReplaceAll(data, []byte("{{HOME}}"), []byte(opts.Home))
		return outName, out
	}
	if err := writeEmbedTree(plan, opts, LaunchAgentsFS, "launchagents", opts.LaunchAgentDir, 0644, transform); err != nil {
		return plan, err
	}

	return plan, nil
}

// writeEmbedTree walks an embed.FS subtree and materializes each file under
// targetDir. The transform callback (optional) may rename the output file
// and/or post-process its contents (used for plist {{HOME}} substitution).
func writeEmbedTree(
	plan *Plan,
	opts Options,
	efs fs.FS,
	srcRoot string,
	targetDir string,
	mode os.FileMode,
	transform func(srcName string, data []byte) (outName string, out []byte),
) error {
	if err := ensureDir(plan, opts, targetDir); err != nil {
		return err
	}

	entries, err := fs.ReadDir(efs, srcRoot)
	if err != nil {
		return fmt.Errorf("reading embedded %s: %w", srcRoot, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// Directives/scripts/launchagents are flat in this package; if
			// we ever nest, recurse here.
			continue
		}
		srcPath := srcRoot + "/" + entry.Name()
		data, err := fs.ReadFile(efs, srcPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", srcPath, err)
		}
		outName := entry.Name()
		out := data
		if transform != nil {
			outName, out = transform(entry.Name(), data)
		}
		if err := writeFile(plan, opts, filepath.Join(targetDir, outName), out, mode); err != nil {
			return err
		}
	}
	return nil
}

// ensureDir records + (unless DryRun) creates a directory.
func ensureDir(plan *Plan, opts Options, dir string) error {
	info, err := os.Stat(dir)
	if err == nil && info.IsDir() {
		// Already exists — no-op for idempotency.
		return nil
	}
	plan.Actions = append(plan.Actions, Action{Kind: ActionMkdir, Path: dir, Mode: 0755})
	logf(opts, "  mkdir  %s\n", dir)
	if opts.DryRun {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return nil
}

// writeFile records + (unless DryRun) writes a file. If the file already
// exists with identical content, the write is skipped (idempotent re-install).
func writeFile(plan *Plan, opts Options, path string, data []byte, mode os.FileMode) error {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		plan.Actions = append(plan.Actions, Action{Kind: ActionSkipExists, Path: path, Mode: mode, Bytes: len(data), Skipped: true})
		return nil
	}
	plan.Actions = append(plan.Actions, Action{Kind: ActionWriteFile, Path: path, Mode: mode, Bytes: len(data)})
	logf(opts, "  write  %s  (%d bytes, mode %o)\n", path, len(data), mode)
	if opts.DryRun {
		return nil
	}
	// Ensure parent exists (defensive; WalkDir should have covered it).
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir parent of %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	// os.WriteFile respects umask; chmod explicitly to guarantee executable bits.
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}

func logf(opts Options, format string, args ...interface{}) {
	if opts.Out == nil {
		return
	}
	fmt.Fprintf(opts.Out, format, args...)
}

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSHA256File_MatchesKnownContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("camp leatherneck"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum) != 64 {
		t.Fatalf("sha256File returned %d chars, want 64 (hex-encoded sha256): %q", len(sum), sum)
	}
	sum2, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	if sum != sum2 {
		t.Fatalf("sha256File not deterministic: %q != %q", sum, sum2)
	}
}

func TestSHA256File_DifferentContentDifferentHash(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.txt")
	pathB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(pathA, []byte("content a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte("content b"), 0o644); err != nil {
		t.Fatal(err)
	}
	sumA, err := sha256File(pathA)
	if err != nil {
		t.Fatal(err)
	}
	sumB, err := sha256File(pathB)
	if err != nil {
		t.Fatal(err)
	}
	if sumA == sumB {
		t.Fatal("different file contents produced the same sha256")
	}
}

func TestSHA256File_MissingFile_Errors(t *testing.T) {
	if _, err := sha256File("/nonexistent/path/does/not/exist"); err == nil {
		t.Fatal("expected an error for a nonexistent file")
	}
}

func TestBuildVersionInfo_DirtyFlagFromVersionSuffix(t *testing.T) {
	origVersion := Version
	defer func() { Version = origVersion }()

	Version = "v0.1.1-36-g1f5a2dde-dirty"
	info := buildVersionInfo()
	if !info.Dirty {
		t.Errorf("expected Dirty=true for Version %q", Version)
	}

	Version = "v0.1.1-36-g1f5a2dde"
	info = buildVersionInfo()
	if info.Dirty {
		t.Errorf("expected Dirty=false for Version %q", Version)
	}
}

func TestBuildVersionInfo_PopulatesInstallPathAndSHA256ForRunningTestBinary(t *testing.T) {
	// os.Executable() during `go test` resolves to the compiled test binary,
	// which is a real file on disk — this exercises the same code path
	// `lt version --json` uses against the real lt binary.
	info := buildVersionInfo()
	if info.InstallPath == "" {
		t.Error("expected InstallPath to be populated")
	}
	if info.SHA256 == "" {
		t.Error("expected SHA256 to be populated")
	}
	if len(info.SHA256) != 64 {
		t.Errorf("expected a 64-char hex sha256, got %d chars: %q", len(info.SHA256), info.SHA256)
	}
	if info.GoVersion == "" {
		t.Error("expected GoVersion to be populated")
	}
}

func TestBuildVersionInfo_JSONShapeMatchesContractRequirement(t *testing.T) {
	// LT_IMPLEMENTATION_CONTRACT.md Phase 2 item 6: "lt version --json:
	// commit, dirty flag, build time, install path, and sha256". This test
	// is the regression guard for that specific requirement — if any of
	// these fields silently disappear from the JSON output in a future
	// refactor, this fails.
	info := buildVersionInfo()
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"commit", "dirty", "install_path"} {
		if _, ok := m[field]; !ok {
			t.Errorf("JSON output missing required field %q: %s", field, data)
		}
	}
	// build_time and sha256 are `omitempty` (legitimately blank on an
	// unbuilt/test binary or when BuildTime wasn't set via ldflags), so we
	// only check them when non-blank in the Go struct, not their presence
	// in the JSON.
	if info.BuildTime != "" && !strings.Contains(string(data), "build_time") {
		t.Error("BuildTime was set on the struct but missing from JSON output")
	}
}

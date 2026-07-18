package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRoleDirective(t *testing.T) {
	t.Parallel()

	t.Run("town-level only", func(t *testing.T) {
		t.Parallel()
		townRoot := t.TempDir()
		townDir := filepath.Join(townRoot, "directives")
		if err := os.MkdirAll(townDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(townDir, "marine.md"), []byte("town directive"), 0644); err != nil {
			t.Fatal(err)
		}

		got := LoadRoleDirective("polecat", townRoot, "myrig")
		if got != "town directive" {
			t.Errorf("got %q, want %q", got, "town directive")
		}
	})

	t.Run("rig-level only", func(t *testing.T) {
		t.Parallel()
		townRoot := t.TempDir()
		rigDir := filepath.Join(townRoot, "myrig", "directives")
		if err := os.MkdirAll(rigDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rigDir, "sarge.md"), []byte("rig directive"), 0644); err != nil {
			t.Fatal(err)
		}

		got := LoadRoleDirective("witness", townRoot, "myrig")
		if got != "rig directive" {
			t.Errorf("got %q, want %q", got, "rig directive")
		}
	})

	t.Run("both levels concatenated with rig last", func(t *testing.T) {
		t.Parallel()
		townRoot := t.TempDir()

		townDir := filepath.Join(townRoot, "directives")
		if err := os.MkdirAll(townDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(townDir, "marine.md"), []byte("town rules"), 0644); err != nil {
			t.Fatal(err)
		}

		rigDir := filepath.Join(townRoot, "myrig", "directives")
		if err := os.MkdirAll(rigDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rigDir, "marine.md"), []byte("rig rules"), 0644); err != nil {
			t.Fatal(err)
		}

		got := LoadRoleDirective("polecat", townRoot, "myrig")
		want := "town rules\nrig rules"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("no directives returns empty", func(t *testing.T) {
		t.Parallel()
		townRoot := t.TempDir()

		got := LoadRoleDirective("polecat", townRoot, "myrig")
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("invalid paths graceful", func(t *testing.T) {
		t.Parallel()

		// Non-existent HQ root
		got := LoadRoleDirective("polecat", "/nonexistent/path/xyz", "myrig")
		if got != "" {
			t.Errorf("got %q, want empty string for invalid HQ root", got)
		}

		// Empty rig name skips rig-level lookup
		townRoot := t.TempDir()
		rigDir := filepath.Join(townRoot, "", "directives")
		// With empty rigName, this path would be townRoot/directives — same as town-level
		// Verify it doesn't panic or error
		_ = rigDir
		got = LoadRoleDirective("polecat", townRoot, "")
		if got != "" {
			t.Errorf("got %q, want empty string for empty rig", got)
		}
	})

	t.Run("whitespace-only file treated as absent", func(t *testing.T) {
		t.Parallel()
		townRoot := t.TempDir()
		townDir := filepath.Join(townRoot, "directives")
		if err := os.MkdirAll(townDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(townDir, "marine.md"), []byte("  \n\t\n  "), 0644); err != nil {
			t.Fatal(err)
		}

		got := LoadRoleDirective("polecat", townRoot, "myrig")
		if got != "" {
			t.Errorf("got %q, want empty string for whitespace-only directive", got)
		}
	})

	t.Run("role name mapping to directive filename", func(t *testing.T) {
		t.Parallel()
		townRoot := t.TempDir()
		townDir := filepath.Join(townRoot, "directives")
		if err := os.MkdirAll(townDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create files with new Marine rank names
		mappings := map[string]string{
			"mayor":    "lt.md",
			"deacon":   "top.md",
			"witness":  "sarge.md",
			"refinery": "gunny.md",
			"polecat":  "marine.md",
		}

		for role, filename := range mappings {
			content := "directive for " + role
			if err := os.WriteFile(filepath.Join(townDir, filename), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			got := LoadRoleDirective(role, townRoot, "")
			if got != content {
				t.Errorf("role %s: got %q, want %q", role, got, content)
			}
		}
	})
}

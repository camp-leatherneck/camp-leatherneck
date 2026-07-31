package config

import (
	"os"
	"path/filepath"
	"strings"
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

// TestStripFrontMatter is the unit-level test for the C3 certification
// requirement: front-matter must be invisible to the agent reading a
// directive, not merely non-crashing.
func TestStripFrontMatter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no front-matter is unchanged",
			input: "# LT\n\nYou are LT.\n",
			want:  "# LT\n\nYou are LT.\n",
		},
		{
			name:  "empty content is unchanged",
			input: "",
			want:  "",
		},
		{
			name: "basic front-matter stripped",
			input: "+++\n" +
				"rank = \"LT\"\n" +
				"persona = \"lt\"\n" +
				"+++\n" +
				"# LT\n\nYou are LT.\n",
			want: "# LT\n\nYou are LT.\n",
		},
		{
			name: "front-matter with leading blank line before it",
			input: "\n+++\n" +
				"rank = \"LT\"\n" +
				"+++\n" +
				"body\n",
			want: "body\n",
		},
		{
			name:  "empty front-matter block",
			input: "+++\n+++\nbody\n",
			want:  "body\n",
		},
		{
			name:  "unclosed front-matter treated as absent (content unchanged)",
			input: "+++\nrank = \"LT\"\nno closing delimiter\n",
			want:  "+++\nrank = \"LT\"\nno closing delimiter\n",
		},
		{
			name:  "+++ not alone on its line is not front-matter",
			input: "+++ this is a markdown thing, not a delimiter\nbody\n",
			want:  "+++ this is a markdown thing, not a delimiter\nbody\n",
		},
		{
			name: "front-matter immediately followed by EOF (no trailing body)",
			input: "+++\n" +
				"rank = \"LT\"\n" +
				"+++\n",
			want: "",
		},
		{
			name: "multi-field front-matter, realistic shape",
			input: "+++\n" +
				"rank = \"LT (O-1)\"\n" +
				"persona = \"lt\"\n" +
				"scope = \"town\"\n" +
				"cardinality = \"singular\"\n" +
				"spawned_by = \"gt prime\"\n" +
				"+++\n" +
				"# LT — Chief of Staff\n\nYou are LT.\n",
			want: "# LT — Chief of Staff\n\nYou are LT.\n",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := stripFrontMatter(c.input)
			if got != c.want {
				t.Errorf("stripFrontMatter(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

// TestLoadRoleDirective_FrontMatterByteIdentical is the direct proof for the
// certification's stated proof requirement: "a persona's loaded directive
// text, before and after front-matter is added, must be byte-identical."
func TestLoadRoleDirective_FrontMatterByteIdentical(t *testing.T) {
	t.Parallel()

	body := "# LT — Chief of Staff\n\nYou are LT. Your CLI role slot is Mayor.\n\n## Section\n\nSome doctrine here.\n"

	// Before: no front-matter.
	before := t.TempDir()
	beforeDir := filepath.Join(before, "directives")
	if err := os.MkdirAll(beforeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beforeDir, "lt.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	beforeResult := LoadRoleDirective("mayor", before, "")

	// After: the exact same body, with front-matter prepended.
	after := t.TempDir()
	afterDir := filepath.Join(after, "directives")
	if err := os.MkdirAll(afterDir, 0755); err != nil {
		t.Fatal(err)
	}
	withFrontMatter := "+++\n" +
		"rank = \"LT (O-1)\"\n" +
		"persona = \"lt\"\n" +
		"scope = \"town\"\n" +
		"cardinality = \"singular\"\n" +
		"spawned_by = \"gt prime\"\n" +
		"+++\n" + body
	if err := os.WriteFile(filepath.Join(afterDir, "lt.md"), []byte(withFrontMatter), 0644); err != nil {
		t.Fatal(err)
	}
	afterResult := LoadRoleDirective("mayor", after, "")

	if beforeResult != afterResult {
		t.Fatalf("directive content is NOT byte-identical after adding front-matter.\nbefore: %q\nafter:  %q", beforeResult, afterResult)
	}
	if beforeResult != strings.TrimSpace(body) {
		t.Fatalf("sanity check failed: before-result %q does not match the expected trimmed body %q", beforeResult, strings.TrimSpace(body))
	}
}

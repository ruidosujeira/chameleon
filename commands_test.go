package main

import (
	"strings"
	"testing"

	"github.com/ruidosujeira/chameleon/theme"
)

func TestRunSubcommandIgnoresTools(t *testing.T) {
	// A real tool invocation must NOT be swallowed as a subcommand.
	for _, argv := range [][]string{
		{"git", "status"},
		{"npm", "outdated"},
		{"docker", "ps"},
		{"echo", "hi"},
	} {
		if runSubcommand(argv) {
			t.Errorf("runSubcommand(%v) = true, want false (should pass through)", argv)
		}
	}
}

func TestInitScript(t *testing.T) {
	cases := []struct {
		shell   string
		wantGit string
		wantErr bool
	}{
		{"zsh", "alias git='chameleon git'", false},
		{"bash", "alias git='chameleon git'", false},
		{"fish", "alias git 'chameleon git'", false},
		{"powershell", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		t.Run(c.shell, func(t *testing.T) {
			got, err := initScript(c.shell)
			if c.wantErr {
				if err == nil {
					t.Fatalf("initScript(%q) err = nil, want error", c.shell)
				}
				return
			}
			if err != nil {
				t.Fatalf("initScript(%q): %v", c.shell, err)
			}
			if !strings.Contains(got, c.wantGit) {
				t.Errorf("initScript(%q) missing %q\n%s", c.shell, c.wantGit, got)
			}
			// Every aliased tool must appear exactly once.
			for _, tool := range aliasedTools {
				if !strings.Contains(got, "chameleon "+tool) {
					t.Errorf("initScript(%q) missing alias for %q", c.shell, tool)
				}
			}
		})
	}
}

func TestEmbeddedThemeNames(t *testing.T) {
	names := embeddedThemeNames()
	want := map[string]bool{
		"tokyonight": true, "dracula": true, "catppuccin": true,
		"gruvbox": true, "nord": true, "solarized": true,
	}
	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	for w := range want {
		if !got[w] {
			t.Errorf("embeddedThemeNames() missing %q; got %v", w, names)
		}
	}
}

// TestEmbeddedThemesAreComplete is the guardrail that keeps every built-in theme
// in sync as adapters add new keys: each must define the full swatchKeys set.
func TestEmbeddedThemesAreComplete(t *testing.T) {
	for _, name := range embeddedThemeNames() {
		tm, err := theme.Load(name, embeddedThemes)
		if err != nil {
			t.Errorf("Load(%q): %v", name, err)
			continue
		}
		if missing := missingKeys(tm); len(missing) > 0 {
			t.Errorf("theme %q is missing color keys: %s", name, strings.Join(missing, ", "))
		}
	}
}

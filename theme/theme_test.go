package theme

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/ruidosujeira/chameleon/style"
)

// embeddedFixture is a stand-in for the //go:embed themes filesystem that
// package main passes in production.
func embeddedFixture(name, body string) fstest.MapFS {
	return fstest.MapFS{
		"themes/" + name + ".toml": &fstest.MapFile{Data: []byte(body)},
	}
}

const sampleTheme = `
name = "sample"

[colors]
prompt = "#9ece6a"
branch = "#bb9af7"

[glyphs]
prompt = "❯"
branch = "⎇"

[layout]
indent = "  "
label_width = 10
`

func TestLoadFromEmbedded(t *testing.T) {
	// Neutralize the user-theme layer so resolution falls through to embedded:
	// an empty XDG dir means step 2 misses, and the unique name means the
	// dev-convenience ./themes/<name>.toml (step 3) can't exist either.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const name = "chameleontest-embedded"

	tm, err := Load(name, embeddedFixture(name, sampleTheme))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if tm.Name != "sample" {
		t.Errorf("Name = %q, want %q", tm.Name, "sample")
	}
	// Hex strings must be resolved to style.Color at load time.
	if got, want := tm.Colors["prompt"], (style.Color{R: 0x9e, G: 0xce, B: 0x6a}); got != want {
		t.Errorf("Colors[prompt] = %v, want %v", got, want)
	}
	if tm.Glyphs["branch"] != "⎇" {
		t.Errorf("Glyphs[branch] = %q, want %q", tm.Glyphs["branch"], "⎇")
	}
	if tm.Layout.Indent != "  " || tm.Layout.LabelWidth != 10 {
		t.Errorf("Layout = {%q, %d}, want {%q, %d}", tm.Layout.Indent, tm.Layout.LabelWidth, "  ", 10)
	}
	// Source records where the theme came from — surfaced by CHAMELEON_DEBUG.
	if want := "embedded:themes/" + name + ".toml"; tm.Source != want {
		t.Errorf("Source = %q, want %q", tm.Source, want)
	}
}

func TestStyles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const name = "chameleontest-styles"
	body := `
name = "styled"

[colors]
branch = "#bb9af7"

[styles]
branch = "bold"
path = "dim italic"
command = "underline,bold"
`
	tm, err := Load(name, embeddedFixture(name, body))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Style() merges the per-key color with the parsed attributes.
	branch := tm.Style("branch")
	if !branch.Bold || !branch.HasFG {
		t.Errorf("Style(branch) = %+v, want Bold and HasFG", branch)
	}
	if branch.FG != (style.Color{R: 0xbb, G: 0x9a, B: 0xf7}) {
		t.Errorf("Style(branch).FG = %v, want the branch color", branch.FG)
	}

	// Attributes parse from space- and comma-separated words.
	if path := tm.Style("path"); !path.Dim || !path.Italic {
		t.Errorf("Style(path) = %+v, want Dim and Italic", path)
	}
	if cmd := tm.Style("command"); !cmd.Underline || !cmd.Bold {
		t.Errorf("Style(command) = %+v, want Underline and Bold", cmd)
	}

	// A key with no color and no style is the zero Style (HasFG false).
	if s := tm.Style("nonexistent"); s.HasFG || s.Bold || s.Dim {
		t.Errorf("Style(nonexistent) = %+v, want zero", s)
	}
}

func TestLoadMissingThemeErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Embedded FS holds a DIFFERENT theme, so the requested name is absent from
	// every source and Load must surface an error rather than silently degrade.
	embedded := embeddedFixture("present", sampleTheme)
	if _, err := Load("chameleontest-absent", embedded); err == nil {
		t.Fatal("expected an error for a theme missing from all sources, got nil")
	}
}

func TestUserThemeBeatsEmbedded(t *testing.T) {
	// Precedence: a personal theme under XDG config (step 2) wins over the
	// embedded built-in (step 4) for the same name.
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	const name = "chameleontest-precedence"
	dir := filepath.Join(xdg, "chameleon", "themes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	userBody := `name = "from-user"` + "\n[colors]\nprompt = \"#ffffff\"\n"
	if err := os.WriteFile(filepath.Join(dir, name+".toml"), []byte(userBody), 0o644); err != nil {
		t.Fatal(err)
	}

	embedded := embeddedFixture(name, `name = "from-embedded"`+"\n")
	tm, err := Load(name, embedded)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tm.Name != "from-user" {
		t.Errorf("Name = %q, want %q (user theme should win over embedded)", tm.Name, "from-user")
	}
}

func TestNameFromEnv(t *testing.T) {
	t.Setenv("CHAMELEON_THEME", "dracula")
	if got := Name(); got != "dracula" {
		t.Errorf("Name() = %q, want %q", got, "dracula")
	}
	t.Setenv("CHAMELEON_THEME", "")
	if got := Name(); got != "tokyonight" {
		t.Errorf("Name() default = %q, want %q", got, "tokyonight")
	}
}

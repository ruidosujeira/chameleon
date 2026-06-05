package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// version is the build version, overridable at link time:
//
//	go build -ldflags "-X main.version=v1.2.3"
var version = "dev"

// runSubcommand handles Chameleon's OWN subcommands (theme tooling, shell
// integration, diagnostics) as opposed to tool passthrough. It returns true
// when argv named a subcommand and it has been handled (the caller then exits).
func runSubcommand(argv []string) bool {
	switch argv[0] {
	case "version", "--version", "-v":
		fmt.Println("chameleon", version)
	case "help", "--help", "-h":
		fmt.Print(usage)
	case "themes":
		cmdThemes(argv[1:])
	case "init":
		cmdInit(argv[1:])
	case "doctor":
		cmdDoctor()
	default:
		return false
	}
	return true
}

const usage = `chameleon — a semantic reformatter for terminal output

Usage:
  chameleon <tool> <args>    re-render a supported tool (git, npm, docker, …)
  chameleon themes [name]    list themes, or preview one as a color swatch
  chameleon init <shell>     print shell aliases (zsh | bash | fish)
  chameleon doctor           report installed tools, theme, and color support
  chameleon version          print the version

Any command without an adapter runs raw, untouched.
Theme: $CHAMELEON_THEME (default tokyonight). $NO_COLOR / $CHAMELEON_FORCE_COLOR honored.
`

// --- themes ---------------------------------------------------------------

// cmdThemes lists the embedded themes (no arg) or previews one as a swatch.
func cmdThemes(args []string) {
	r := &style.Renderer{Enabled: true, Profile: style.TrueColor}

	if len(args) == 0 {
		fmt.Println("Available themes (resolved name in bold is the active one):")
		active := theme.Name()
		for _, name := range embeddedThemeNames() {
			marker := "  "
			if name == active {
				marker = "❯ "
			}
			fmt.Printf("%s%s\n", marker, name)
		}
		fmt.Printf("\nPreview one with:  chameleon themes %s\n", active)
		return
	}

	name := args[0]
	t, err := theme.Load(name, embeddedThemes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chameleon:", err)
		os.Exit(1)
	}
	fmt.Printf("%s  (%s)\n", t.Name, t.Source)
	fmt.Print(swatch(t, r))
	if missing := missingKeys(t); len(missing) > 0 {
		fmt.Printf("\n⚠ missing keys (will fall back): %s\n", strings.Join(missing, ", "))
	}
}

// swatchKeys is the full set of color keys an adapter may reference, in a
// reading order grouped by purpose. Used for both the swatch and validation.
var swatchKeys = []string{
	"prompt", "command", "branch", "ahead", "behind", "staged", "modified", "untracked", "path", "dim",
	"added", "deleted", "renamed", "copied", "typechange", "conflict",
	"major", "minor", "patch", "update",
	"ok", "warn", "bad", "unknown",
}

// swatch renders a theme as one line per key: a color block, the key name, its
// glyph (if any), and the hex — so you can eyeball a theme before using it.
func swatch(t *theme.Theme, r *style.Renderer) string {
	var sb strings.Builder
	for _, k := range swatchKeys {
		c, ok := t.Colors[k]
		if !ok {
			continue
		}
		sb.WriteString("  ")
		sb.WriteString(r.Paint(c, "███"))
		sb.WriteString(" ")
		sb.WriteString(style.PadRight(k, 12))
		sb.WriteString(style.PadRight(t.Glyphs[k], 3))
		sb.WriteString(fmt.Sprintf("#%02x%02x%02x\n", c.R, c.G, c.B))
	}
	return sb.String()
}

// missingKeys reports color keys from swatchKeys the theme doesn't define. They
// still render (adapters fall back), but a complete theme defines them all.
func missingKeys(t *theme.Theme) []string {
	var missing []string
	for _, k := range swatchKeys {
		if _, ok := t.Colors[k]; !ok {
			missing = append(missing, k)
		}
	}
	return missing
}

// embeddedThemeNames lists the built-in theme names (sans .toml), sorted.
func embeddedThemeNames() []string {
	entries, err := fs.ReadDir(embeddedThemes, "themes")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if n := strings.TrimSuffix(e.Name(), ".toml"); n != e.Name() {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

// --- init -----------------------------------------------------------------

// aliasedTools are the commands worth wrapping with `chameleon`. Wrapping is
// safe: a non-adapter subcommand (e.g. `git log`) falls through to raw passthrough.
var aliasedTools = []string{
	"git", "npm", "pip", "pip3", "cargo", "brew", "go", "gem",
	"kubectl", "docker", "gh", "systemctl",
}

// cmdInit prints shell aliases that route supported tools through chameleon.
func cmdInit(args []string) {
	shell := ""
	if len(args) > 0 {
		shell = args[0]
	}
	script, err := initScript(shell)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chameleon:", err)
		os.Exit(1)
	}
	fmt.Print(script)
}

// initScript returns the alias block for a shell. Pure, so it is unit-tested.
func initScript(shell string) (string, error) {
	var sb strings.Builder
	switch shell {
	case "zsh", "bash":
		sb.WriteString("# chameleon shell integration — add to your shell rc:\n")
		fmt.Fprintf(&sb, "#   eval \"$(chameleon init %s)\"\n", shell)
		sb.WriteString("# Only the supported subcommands are re-rendered; everything else runs raw.\n")
		for _, tool := range aliasedTools {
			fmt.Fprintf(&sb, "alias %s='chameleon %s'\n", tool, tool)
		}
	case "fish":
		sb.WriteString("# chameleon shell integration — add to ~/.config/fish/config.fish:\n")
		sb.WriteString("#   chameleon init fish | source\n")
		for _, tool := range aliasedTools {
			fmt.Fprintf(&sb, "alias %s 'chameleon %s'\n", tool, tool)
		}
	default:
		return "", fmt.Errorf("init: unknown shell %q (want zsh, bash, or fish)", shell)
	}
	return sb.String(), nil
}

// --- doctor ---------------------------------------------------------------

// cmdDoctor reports the active theme, color support, and which adapter tools are
// installed — a quick "is my setup working?" check.
func cmdDoctor() {
	fmt.Println("chameleon", version)

	t, err := theme.Load(theme.Name(), embeddedThemes)
	if err != nil {
		fmt.Println("theme: ERROR —", err)
	} else {
		fmt.Printf("theme: %s  (%s)\n", t.Name, t.Source)
	}

	r := style.New()
	fmt.Printf("color: enabled=%t  profile=%s\n", r.Enabled, profileName(r.Profile))

	fmt.Println("tools on PATH:")
	for _, tool := range aliasedTools {
		mark := "✗"
		if _, err := exec.LookPath(tool); err == nil {
			mark = "✓"
		}
		fmt.Printf("  %s %s\n", mark, tool)
	}
}

// profileName is the human label for a color depth.
func profileName(p style.Profile) string {
	switch p {
	case style.Ansi256:
		return "256-color"
	case style.Ansi16:
		return "16-color"
	default:
		return "truecolor"
	}
}

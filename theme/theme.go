// Package theme loads the single shared visual theme that every adapter draws
// from. One theme, one identity: git, npm, docker and friends all come out
// looking like siblings.
package theme

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/ruidosujeira/chameleon/style"
)

// Theme is the in-memory, ready-to-use form of a .toml theme: hex strings have
// already been resolved to style.Color values at load time.
type Theme struct {
	Name   string
	Colors map[string]style.Color
	Glyphs map[string]string
	Layout struct {
		Indent     string
		LabelWidth int
	}
}

// rawTheme mirrors the on-disk TOML shape, where colors are still hex strings.
type rawTheme struct {
	Name   string            `toml:"name"`
	Colors map[string]string `toml:"colors"`
	Glyphs map[string]string `toml:"glyphs"`
	Layout struct {
		Indent     string `toml:"indent"`
		LabelWidth int    `toml:"label_width"`
	} `toml:"layout"`
}

// Load resolves and decodes the theme named `name`, turning every hex color
// into a style.Color up front. `embedded` is the built-in theme filesystem
// (see the //go:embed in package main) and is the guaranteed fallback.
//
// Resolution order — the FIRST source that exists wins:
//
//  1. .chameleon.toml in the CWD or any parent up to the git root — the team's
//     versionable theme; a full theme file that overrides everything.
//  2. $XDG_CONFIG_HOME/chameleon/themes/<name>.toml (defaults to
//     ~/.config/...) — the user's personal theme.
//  3. ./themes/<name>.toml relative to the CWD — dev convenience when working
//     inside the repo.
//  4. embedded themes/<name>.toml — built-in fallback, so the binary works
//     from ANY directory with no files alongside it.
func Load(name string, embedded fs.FS) (*Theme, error) {
	data, source, err := resolve(name, embedded)
	if err != nil {
		return nil, err
	}

	var raw rawTheme
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, fmt.Errorf("parse theme %s: %w", source, err)
	}

	t := &Theme{
		Name:   raw.Name,
		Colors: make(map[string]style.Color, len(raw.Colors)),
		Glyphs: raw.Glyphs,
	}
	for k, hex := range raw.Colors {
		t.Colors[k] = style.Hex(hex)
	}
	t.Layout.Indent = raw.Layout.Indent
	t.Layout.LabelWidth = raw.Layout.LabelWidth

	return t, nil
}

// resolve returns the raw bytes of the winning theme source, following the
// precedence documented on Load.
func resolve(name string, embedded fs.FS) (data []byte, source string, err error) {
	// 1. Project override: .chameleon.toml walking up to the git root.
	if cwd, e := os.Getwd(); e == nil {
		if p := findProjectTheme(cwd); p != "" {
			b, e := os.ReadFile(p)
			if e != nil {
				return nil, "", fmt.Errorf("read %s: %w", p, e)
			}
			return b, p, nil
		}
	}

	// 2. User theme under XDG config.
	if dir := userThemeDir(); dir != "" {
		p := filepath.Join(dir, name+".toml")
		if b, e := os.ReadFile(p); e == nil {
			return b, p, nil
		}
	}

	// 3. Dev convenience: ./themes/<name>.toml relative to the CWD.
	if p := filepath.Join("themes", name+".toml"); fileExists(p) {
		if b, e := os.ReadFile(p); e == nil {
			return b, p, nil
		}
	}

	// 4. Embedded built-in — the guaranteed fallback.
	p := "themes/" + name + ".toml"
	b, e := fs.ReadFile(embedded, p)
	if e != nil {
		return nil, "", fmt.Errorf("theme %q not found in any source (embedded: %w)", name, e)
	}
	return b, "embedded:" + p, nil
}

// findProjectTheme walks up from `start` looking for a .chameleon.toml,
// stopping at (and including) the git root so the search never escapes the
// repository. Returns "" if none is found.
func findProjectTheme(start string) string {
	dir := start
	for {
		if candidate := filepath.Join(dir, ".chameleon.toml"); fileExists(candidate) {
			return candidate
		}
		// The git root is the ceiling: check it (done above) then stop.
		if dirExists(filepath.Join(dir, ".git")) {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // reached the filesystem root
		}
		dir = parent
	}
}

// userThemeDir is $XDG_CONFIG_HOME/chameleon/themes, falling back to
// ~/.config/chameleon/themes. Returns "" if the home dir can't be determined.
func userThemeDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "chameleon", "themes")
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// Name resolves which theme to load: CHAMELEON_THEME if set, else "tokyonight".
func Name() string {
	if n := os.Getenv("CHAMELEON_THEME"); n != "" {
		return n
	}
	return "tokyonight"
}

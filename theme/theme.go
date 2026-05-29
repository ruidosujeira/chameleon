// Package theme loads the single shared visual theme that every adapter draws
// from. One theme, one identity: git, npm, docker and friends all come out
// looking like siblings.
package theme

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"chameleon/style"
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

// Load reads themes/<name>.toml, decoding it with BurntSushi/toml and turning
// every hex color into a style.Color up front.
func Load(name string) (*Theme, error) {
	path := fmt.Sprintf("themes/%s.toml", name)

	var raw rawTheme
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, fmt.Errorf("load theme %q: %w", name, err)
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

// Name resolves which theme to load: CHAMELEON_THEME if set, else "tokyonight".
func Name() string {
	if n := os.Getenv("CHAMELEON_THEME"); n != "" {
		return n
	}
	return "tokyonight"
}

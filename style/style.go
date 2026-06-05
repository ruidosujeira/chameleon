// Package style is Chameleon's own minimal styling layer.
//
// This is the core competency of the project, so it deliberately pulls in NO
// external framework (no Charm/lipgloss). Only three primitives live here:
// a truecolor type, a TTY-aware renderer, and display-width helpers.
package style

import (
	"os"
	"strconv"
	"strings"
	"unicode"
)

// Color is a 24-bit truecolor value.
type Color struct {
	R, G, B uint8
}

// Hex parses "#rrggbb" (with or without the leading '#'). Any malformed input
// degrades to black (the zero Color) rather than erroring — themes should be
// forgiving at the edges.
func Hex(s string) Color {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return Color{}
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return Color{}
	}
	return Color{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
	}
}

// Renderer decides whether color is emitted at all, and at what depth. Build it
// with New so the TTY/env detection runs once.
type Renderer struct {
	// Enabled is the master switch: when false, Paint returns text untouched.
	Enabled bool
	// Profile is the color depth. Its zero value is TrueColor, so a bare
	// Renderer{Enabled: true} renders 24-bit (what themes are authored in).
	Profile Profile
}

// New constructs a Renderer with color enabled according to the rules:
//   - CHAMELEON_FORCE_COLOR=1 forces color on (wins over everything);
//   - otherwise NO_COLOR being set forces it off;
//   - otherwise color is on only when stdout is a character device (a real
//     terminal), not a pipe or file.
//
// The color depth is detected from COLORTERM/TERM (see detectProfile) so a
// truecolor theme degrades gracefully on 256- and 16-color terminals.
//
// We intentionally avoid any TTY library: os.Stdout.Stat() + os.ModeCharDevice
// is all that's needed.
func New() *Renderer {
	return &Renderer{Enabled: colorEnabled(), Profile: detectProfile()}
}

func colorEnabled() bool {
	if os.Getenv("CHAMELEON_FORCE_COLOR") == "1" {
		return true
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Paint wraps text in a foreground-color SGR sequence (at the renderer's color
// depth) and resets afterwards. When the renderer is disabled it returns the
// text untouched, so output stays clean in pipes and under NO_COLOR.
func (r *Renderer) Paint(c Color, text string) string {
	if !r.Enabled {
		return text
	}
	return "\x1b[" + r.fgParams(c) + "m" + text + "\x1b[0m"
}

// Style is a foreground color plus text attributes. It lets a theme say "the
// branch is bold" or "the path is dim" without every adapter call site changing.
type Style struct {
	FG        Color
	HasFG     bool // when false, only the attributes are applied (color untouched)
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
}

// PaintStyle wraps text in the SGR sequence for s (attributes + optional
// foreground) and resets afterwards. A disabled renderer, or a Style with
// nothing set, returns the text untouched.
func (r *Renderer) PaintStyle(s Style, text string) string {
	if !r.Enabled {
		return text
	}
	var params []string
	if s.Bold {
		params = append(params, "1")
	}
	if s.Dim {
		params = append(params, "2")
	}
	if s.Italic {
		params = append(params, "3")
	}
	if s.Underline {
		params = append(params, "4")
	}
	if s.HasFG {
		params = append(params, r.fgParams(s.FG))
	}
	if len(params) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(params, ";") + "m" + text + "\x1b[0m"
}

// Width returns the display width of s in terminal columns.
//
// It counts runes (not bytes) and skips zero-width combining marks (Unicode
// categories Mn, Me) and format characters (Cf). Everything else is assumed to
// be one column wide.
//
// THIS IS THE ONE PLACE to swap in github.com/mattn/go-runewidth when wide CJK
// or emoji support is needed; today's adapters only emit width-1 glyphs, so the
// dependency is not yet justified.
func Width(s string) int {
	w := 0
	for _, r := range s {
		if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
			continue
		}
		w++
	}
	return w
}

// PadRight pads s with spaces until it occupies `width` display columns. If s
// is already at least that wide it is returned unchanged.
//
// CRITICAL INVARIANT: ALWAYS PadRight BEFORE Paint. Painting first injects ANSI
// escape codes into the string, and Width would then count those bytes as
// visible columns, throwing off every subsequent alignment. Pad the raw text,
// then color it.
func PadRight(s string, width int) string {
	pad := width - Width(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

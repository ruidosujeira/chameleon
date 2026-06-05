package style

import (
	"os"
	"strings"
)

// Profile is the color depth a terminal can render. Chameleon authors every
// theme in 24-bit truecolor and downsamples on the way out when the terminal
// can't do better — so a theme looks as close as possible everywhere without
// the theme author thinking about it.
type Profile int

const (
	// TrueColor is 24-bit (16M colors) — emitted verbatim. It is the zero value
	// so a bare Renderer{Enabled: true} renders truecolor (what we author in).
	TrueColor Profile = iota
	// Ansi256 is the xterm 256-color palette (6×6×6 cube + grayscale ramp).
	Ansi256
	// Ansi16 is the 8 normal + 8 bright ANSI colors — the floor for old terminals.
	Ansi16
)

// detectProfile picks a color depth from the environment. COLORTERM=truecolor
// wins outright; otherwise TERM decides, defaulting to truecolor when TERM is
// unhelpful (color was forced, so assume a modern emulator).
func detectProfile() Profile {
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		return TrueColor
	}
	term := os.Getenv("TERM")
	switch {
	case strings.Contains(term, "256color"):
		return Ansi256
	case term == "" || term == "dumb":
		return TrueColor
	default:
		return Ansi16
	}
}

// fgParams returns the SGR foreground parameters for c at this renderer's color
// depth (e.g. "38;2;r;g;b" for truecolor, "38;5;n" for 256, "31"/"91" for 16).
func (r *Renderer) fgParams(c Color) string {
	switch r.Profile {
	case Ansi256:
		return "38;5;" + itoa(toXterm256(c))
	case Ansi16:
		return itoa(toAnsiSGR(c))
	default: // TrueColor
		return "38;2;" + itoa(int(c.R)) + ";" + itoa(int(c.G)) + ";" + itoa(int(c.B))
	}
}

// toXterm256 maps a truecolor value onto the xterm 256-color palette: the
// grayscale ramp (232–255) when the channels are equal, otherwise the nearest
// cell of the 6×6×6 color cube (16–231). This is the de-facto standard mapping.
func toXterm256(c Color) int {
	if c.R == c.G && c.G == c.B {
		switch {
		case c.R < 8:
			return 16
		case c.R > 248:
			return 231
		default:
			return 232 + (int(c.R)-8)*24/247
		}
	}
	return 16 + 36*cube6(c.R) + 6*cube6(c.G) + cube6(c.B)
}

// cube6 quantizes one 0–255 channel to a 0–5 index on the xterm color cube,
// whose levels are 0, 95, 135, 175, 215, 255.
func cube6(v uint8) int {
	if v < 48 {
		return 0
	}
	if v < 115 {
		return 1
	}
	return (int(v) - 35) / 40
}

// ansi16Palette is the standard xterm RGB for the 16 base colors, index 0–15.
var ansi16Palette = [16]Color{
	{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
	{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
	{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
	{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// toAnsiSGR maps c to the nearest of the 16 base colors and returns its SGR
// foreground code: 30–37 for the normal eight, 90–97 for the bright eight.
func toAnsiSGR(c Color) int {
	best, bestDist := 0, 1<<31-1
	for i, p := range ansi16Palette {
		dr, dg, db := int(c.R)-int(p.R), int(c.G)-int(p.G), int(c.B)-int(p.B)
		if d := dr*dr + dg*dg + db*db; d < bestDist {
			best, bestDist = i, d
		}
	}
	if best < 8 {
		return 30 + best
	}
	return 90 + (best - 8)
}

// itoa is a tiny dependency-free int→string (avoids pulling strconv into this
// hot path; the values are always small and non-negative).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

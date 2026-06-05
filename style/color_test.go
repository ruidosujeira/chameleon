package style

import "testing"

func TestToXterm256(t *testing.T) {
	cases := []struct {
		in   Color
		want int
	}{
		{Color{0, 0, 0}, 16},        // black → cube origin
		{Color{255, 255, 255}, 231}, // white → cube corner
		{Color{255, 0, 0}, 196},     // pure red → cube
		{Color{128, 128, 128}, 243}, // mid gray → grayscale ramp
	}
	for _, c := range cases {
		if got := toXterm256(c.in); got != c.want {
			t.Errorf("toXterm256(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestToAnsiSGR(t *testing.T) {
	cases := []struct {
		in   Color
		want int
	}{
		{Color{0, 0, 0}, 30},       // black
		{Color{255, 255, 255}, 97}, // bright white
		{Color{255, 0, 0}, 91},     // bright red
		{Color{0, 255, 0}, 92},     // bright green
	}
	for _, c := range cases {
		if got := toAnsiSGR(c.in); got != c.want {
			t.Errorf("toAnsiSGR(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestPaintAtProfile(t *testing.T) {
	red := Color{255, 0, 0}
	cases := []struct {
		profile Profile
		want    string
	}{
		{TrueColor, "\x1b[38;2;255;0;0mX\x1b[0m"},
		{Ansi256, "\x1b[38;5;196mX\x1b[0m"},
		{Ansi16, "\x1b[91mX\x1b[0m"},
	}
	for _, c := range cases {
		r := &Renderer{Enabled: true, Profile: c.profile}
		if got := r.Paint(red, "X"); got != c.want {
			t.Errorf("profile %d: Paint = %q, want %q", c.profile, got, c.want)
		}
	}
}

func TestPaintStyle(t *testing.T) {
	r := &Renderer{Enabled: true} // TrueColor (zero value)
	green := Color{158, 206, 106}

	if got := r.PaintStyle(Style{FG: green, HasFG: true, Bold: true}, "hi"); got != "\x1b[1;38;2;158;206;106mhi\x1b[0m" {
		t.Errorf("bold+fg = %q", got)
	}
	if got := r.PaintStyle(Style{Dim: true}, "hi"); got != "\x1b[2mhi\x1b[0m" {
		t.Errorf("dim only = %q", got)
	}
	if got := r.PaintStyle(Style{Italic: true, Underline: true}, "hi"); got != "\x1b[3;4mhi\x1b[0m" {
		t.Errorf("italic+underline = %q", got)
	}
	// Nothing set → untouched, even when enabled.
	if got := r.PaintStyle(Style{}, "hi"); got != "hi" {
		t.Errorf("empty style = %q, want %q", got, "hi")
	}
	// Disabled renderer → always untouched.
	off := &Renderer{Enabled: false}
	if got := off.PaintStyle(Style{Bold: true, FG: green, HasFG: true}, "hi"); got != "hi" {
		t.Errorf("disabled = %q, want %q", got, "hi")
	}
}

func TestDetectProfile(t *testing.T) {
	cases := []struct {
		name      string
		colorterm string
		term      string
		want      Profile
	}{
		{"colorterm truecolor wins", "truecolor", "xterm", TrueColor},
		{"colorterm 24bit wins", "24bit", "xterm-256color", TrueColor},
		{"term 256color", "", "xterm-256color", Ansi256},
		{"term plain xterm", "", "xterm", Ansi16},
		{"term dumb falls back to truecolor", "", "dumb", TrueColor},
		{"term empty falls back to truecolor", "", "", TrueColor},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("COLORTERM", c.colorterm)
			t.Setenv("TERM", c.term)
			if got := detectProfile(); got != c.want {
				t.Errorf("detectProfile() = %d, want %d", got, c.want)
			}
		})
	}
}

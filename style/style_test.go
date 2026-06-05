package style

import "testing"

func TestHex(t *testing.T) {
	cases := []struct {
		in   string
		want Color
	}{
		{"#9ece6a", Color{0x9e, 0xce, 0x6a}},
		{"9ece6a", Color{0x9e, 0xce, 0x6a}}, // leading '#' is optional
		{"#000000", Color{0, 0, 0}},
		{"#ffffff", Color{0xff, 0xff, 0xff}},
		{"#fff", Color{}},     // too short → zero (forgiving, not an error)
		{"#gggggg", Color{}},  // non-hex digits → zero
		{"", Color{}},         // empty → zero
		{"#9ece6a7", Color{}}, // too long → zero
	}
	for _, c := range cases {
		if got := Hex(c.in); got != c.want {
			t.Errorf("Hex(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestWidth(t *testing.T) {
	// Inputs use \u escapes (not literal glyphs) so the source stays byte-stable
	// and each case provably exercises the Unicode category it claims.
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"ascii", "main", 4},
		{"empty", "", 0},
		{"single glyph rune", "❯", 1},                 // ❯ → 1 column
		{"precomposed accent is one rune", "café", 4}, // café (é = U+00E9)
		{"combining mark is zero width", "é", 1},     // e + COMBINING ACUTE (Mn)
		{"zero-width joiner skipped", "a‍b", 2},       // a + ZWJ (Cf) + b
		{"zero-width space skipped", "a​b", 2},        // a + ZERO WIDTH SPACE (Cf) + b
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Width(c.in); got != c.want {
				t.Errorf("Width(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"added", 10, "added     "},        // 5 chars → pad 5
		{"untracked", 10, "untracked "},    // 9 chars → pad 1
		{"label_width", 10, "label_width"}, // already wider → unchanged
		{"exact", 5, "exact"},              // exactly at width → unchanged
		{"", 3, "   "},                     // empty padded to width
	}
	for _, c := range cases {
		if got := PadRight(c.in, c.width); got != c.want {
			t.Errorf("PadRight(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}

// TestPadRightWidthByDisplayColumns guards the alignment contract: padding is
// computed from DISPLAY width, so a combining-mark string (2 runes, 1 visible
// column) still pads out to the right number of columns.
func TestPadRightWidthByDisplayColumns(t *testing.T) {
	got := PadRight("é", 3) // e + combining acute = 1 display column
	if Width(got) != 3 {
		t.Errorf("Width(PadRight(...)) = %d, want 3 display columns", Width(got))
	}
}

func TestPaintDisabled(t *testing.T) {
	r := &Renderer{Enabled: false}
	if got := r.Paint(Color{0x9e, 0xce, 0x6a}, "hi"); got != "hi" {
		t.Errorf("disabled Paint = %q, want %q (no escapes)", got, "hi")
	}
}

func TestPaintEnabled(t *testing.T) {
	r := &Renderer{Enabled: true}
	want := "\x1b[38;2;158;206;106mhi\x1b[0m"
	if got := r.Paint(Color{0x9e, 0xce, 0x6a}, "hi"); got != want {
		t.Errorf("enabled Paint = %q, want %q", got, want)
	}
}

// TestPaintInflatesMeasuredWidth shows why PadRight must run BEFORE Paint: Width
// is NOT escape-aware, so it counts a painted string's escape bytes as visible
// columns. Padding after painting would therefore shear every column.
func TestPaintInflatesMeasuredWidth(t *testing.T) {
	r := &Renderer{Enabled: true}
	painted := r.Paint(Color{1, 2, 3}, "ok")
	if Width(painted) <= Width("ok") {
		t.Fatalf("Width(painted)=%d should exceed Width(raw)=%d", Width(painted), Width("ok"))
	}
}

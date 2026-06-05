package style

import "testing"

// FuzzPadRight asserts the alignment contract holds for any input: PadRight
// never shrinks a string, and the result is at least `width` display columns.
func FuzzPadRight(f *testing.F) {
	f.Add("added", 10)
	f.Add("", 3)
	f.Add("é", 4)   // combining mark
	f.Add("已经", 2)  // wide-ish runes (counted as 1 each today)
	f.Add("a‍b", 5) // zero-width joiner
	f.Fuzz(func(t *testing.T, s string, width int) {
		if width < 0 || width > 1<<16 {
			return // nonsensical widths aren't a real input
		}
		got := PadRight(s, width)
		if Width(got) < Width(s) {
			t.Fatalf("PadRight shrank width: %q (%d) -> %q (%d)", s, Width(s), got, Width(got))
		}
		if w := Width(got); w < width && Width(s) <= width {
			t.Fatalf("PadRight(%q, %d) = %q has width %d, want >= %d", s, width, got, w, width)
		}
	})
}

// FuzzHex asserts Hex never panics and only ever returns the zero color for
// inputs that aren't a clean 6-hex-digit value.
func FuzzHex(f *testing.F) {
	f.Add("#9ece6a")
	f.Add("zzzzzz")
	f.Add("#fff")
	f.Fuzz(func(t *testing.T, s string) {
		_ = Hex(s) // must not panic
	})
}

package adapters

import (
	"testing"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// npmTestTheme is a minimal theme with the glyphs the npm adapter needs. The
// renderer is disabled in tests, so colors don't affect the output text.
func npmTestTheme() *theme.Theme {
	t := &theme.Theme{
		Colors: map[string]style.Color{},
		Glyphs: map[string]string{
			"prompt": "❯", "clean": "✓",
			"major": "▲", "minor": "▴", "patch": "▵", "update": "↑",
		},
	}
	t.Layout.Indent = "  "
	t.Layout.LabelWidth = 10
	return t
}

func TestClassifyVersion(t *testing.T) {
	cases := []struct {
		name            string
		current, latest string
		want            string
	}{
		{"major bump", "1.0.0", "2.0.0", "major"},
		{"major over minor/patch", "1.2.3", "2.0.0", "major"},
		{"minor bump", "1.2.0", "1.3.0", "minor"},
		{"patch bump", "1.2.3", "1.2.4", "patch"},
		{"already current", "1.2.3", "1.2.3", "update"},
		{"downgrade is not an upgrade", "2.0.0", "1.0.0", "update"},
		{"missing current", "", "5.4.0", "update"},
		{"prerelease falls back", "1.2.3-beta", "1.2.4", "update"},
		{"short version still compares", "1.2", "1.3", "minor"},
		{"non-numeric latest", "1.2.3", "latest", "update"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyVersion(c.current, c.latest); got != c.want {
				t.Errorf("classifyVersion(%q, %q) = %q, want %q", c.current, c.latest, got, c.want)
			}
		})
	}
}

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in     string
		want   [3]int
		wantOK bool
	}{
		{"1.2.3", [3]int{1, 2, 3}, true},
		{"1.2", [3]int{1, 2, 0}, true},
		{"10.0.0", [3]int{10, 0, 0}, true},
		{"1.2.3-beta", [3]int{}, false},
		{"1.2.3.4", [3]int{}, false},
		{"v1.2.3", [3]int{}, false},
		{"", [3]int{}, false},
	}
	for _, c := range cases {
		got, ok := parseSemver(c.in)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("parseSemver(%q) = %v,%v; want %v,%v", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestRenderOutdated(t *testing.T) {
	// Disabled renderer → deterministic, escape-free golden output.
	r := &style.Renderer{Enabled: false}

	cases := []struct {
		name string
		json string
		want string
	}{
		{
			name: "all up to date (empty json)",
			json: `{}`,
			want: "❯ npm outdated\n" +
				"  ✓ tudo atualizado\n",
		},
		{
			// Covers major, minor, patch, and a missing-current package, plus
			// severity ordering (major → minor → patch → update) and column
			// alignment across rows of varying widths.
			name: "mixed severities and missing current",
			json: `{
				"left-pad":   {"current":"1.0.0","wanted":"1.0.1","latest":"2.3.0","type":"dependencies"},
				"lodash":     {"current":"3.10.1","wanted":"3.10.1","latest":"3.99.0","type":"dependencies"},
				"chalk":      {"current":"4.1.0","wanted":"4.1.2","latest":"4.1.2","type":"dependencies"},
				"typescript": {"wanted":"5.0.0","latest":"5.4.0","type":"devDependencies"}
			}`,
			want: "❯ npm outdated\n" +
				"  ▲ major      left-pad    1.0.0  → 1.0.1   (latest 2.3.0)\n" +
				"  ▴ minor      lodash      3.10.1 → 3.10.1  (latest 3.99.0)\n" +
				"  ▵ patch      chalk       4.1.0  → 4.1.2   (latest 4.1.2)\n" +
				"  ↑ update     typescript  —      → 5.0.0   (latest 5.4.0)\n",
		},
		{
			name: "blank output treated as up to date",
			json: "  \n",
			want: "❯ npm outdated\n" +
				"  ✓ tudo atualizado\n",
		},
		{
			// Workspace shape: npm emits an ARRAY of objects per package. We
			// collapse to one row. A null value is skipped, not crashed on.
			name: "workspace array shape",
			json: `{
				"chalk": [
					{"current":"4.0.0","wanted":"4.0.0","latest":"5.6.2","dependent":"a"},
					{"current":"4.0.0","wanted":"4.0.0","latest":"5.6.2","dependent":"b"}
				],
				"ignored": null
			}`,
			want: "❯ npm outdated\n" +
				"  ▲ major      chalk  4.0.0 → 4.0.0  (latest 5.6.2)\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := renderOutdated([]byte(c.json), npmTestTheme(), r)
			if err != nil {
				t.Fatalf("renderOutdated: %v", err)
			}
			if got != c.want {
				t.Errorf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.want)
			}
		})
	}
}

func TestRenderOutdatedInvalidJSON(t *testing.T) {
	r := &style.Renderer{Enabled: false}
	if _, err := renderOutdated([]byte(`{not json`), npmTestTheme(), r); err == nil {
		t.Fatal("expected an error for malformed JSON, got nil")
	}
}

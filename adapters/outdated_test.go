package adapters

import (
	"testing"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// renderFn is the pure transform every package-manager adapter exposes.
type renderFn func([]byte, *theme.Theme, *style.Renderer) (string, error)

// TestOutdatedFamily golden-tests every package manager's normalizer against a
// fixture of its real machine output. They all funnel into renderUpgrades, so
// these double as proof that pip/cargo/brew/go/gem come out as npm's siblings.
func TestOutdatedFamily(t *testing.T) {
	r := &style.Renderer{Enabled: false}

	cases := []struct {
		name   string
		render renderFn
		in     string
		want   string
	}{
		{
			name:   "pip mixed severities",
			render: renderPipOutdated,
			in: `[
				{"name":"requests","version":"2.25.0","latest_version":"2.31.0"},
				{"name":"flask","version":"2.0.0","latest_version":"2.0.3"},
				{"name":"numpy","version":"1.20.0","latest_version":"2.0.0"}
			]`,
			want: "❯ pip list --outdated\n" +
				"  ▲ major      numpy     1.20.0 → 2.0.0   (latest 2.0.0)\n" +
				"  ◆ minor      requests  2.25.0 → 2.31.0  (latest 2.31.0)\n" +
				"  ▪ patch      flask     2.0.0  → 2.0.3   (latest 2.0.3)\n",
		},
		{
			name:   "pip empty is up to date",
			render: renderPipOutdated,
			in:     `[]`,
			want:   "❯ pip list --outdated\n  ✓ tudo atualizado\n",
		},
		{
			name:   "cargo skips up-to-date and '---' compat",
			render: renderCargoOutdated,
			in: `{"dependencies":[
				{"name":"serde","project":"1.0.130","compat":"1.0.190","latest":"1.0.190"},
				{"name":"tokio","project":"1.0.0","compat":"---","latest":"2.0.0"},
				{"name":"clap","project":"4.0.0","compat":"---","latest":"---"}
			]}`,
			want: "❯ cargo outdated\n" +
				"  ▲ major      tokio  1.0.0   → 2.0.0    (latest 2.0.0)\n" +
				"  ▪ patch      serde  1.0.130 → 1.0.190  (latest 1.0.190)\n",
		},
		{
			name:   "brew formulae and casks merged",
			render: renderBrewOutdated,
			in: `{
				"formulae":[{"name":"git","installed_versions":["2.39.0"],"current_version":"2.43.0"}],
				"casks":[{"name":"firefox","installed_versions":["120.0"],"current_version":"121.0"}]
			}`,
			want: "❯ brew outdated\n" +
				"  ▲ major      firefox  120.0  → 121.0   (latest 121.0)\n" +
				"  ◆ minor      git      2.39.0 → 2.43.0  (latest 2.43.0)\n",
		},
		{
			name:   "go list stream, only modules with Update",
			render: renderGoOutdated,
			in: `{"Path":"github.com/foo/bar","Version":"v1.2.0","Update":{"Version":"v2.0.0"}}
{"Path":"github.com/baz/qux","Version":"v1.0.0"}
{"Path":"golang.org/x/sync","Version":"v0.1.0","Update":{"Version":"v0.6.0"}}`,
			want: "❯ go list -m -u\n" +
				"  ▲ major      github.com/foo/bar  v1.2.0 → v2.0.0  (latest v2.0.0)\n" +
				"  ◆ minor      golang.org/x/sync   v0.1.0 → v0.6.0  (latest v0.6.0)\n",
		},
		{
			name:   "go list nothing outdated",
			render: renderGoOutdated,
			in:     `{"Path":"github.com/foo/bar","Version":"v1.0.0"}`,
			want:   "❯ go list -m -u\n  ✓ tudo atualizado\n",
		},
		{
			name: "gem line parse",
			render: func(b []byte, t *theme.Theme, r *style.Renderer) (string, error) {
				return renderGemOutdated(b, t, r), nil
			},
			in: "rails (6.0.0 < 7.0.0)\nrake (12.3.0 < 12.3.3)\nnokogiri (1.10.0 < 1.10.4)\nGEM is up to date noise\n",
			want: "❯ gem outdated\n" +
				"  ▲ major      rails     6.0.0  → 7.0.0   (latest 7.0.0)\n" +
				"  ▪ patch      nokogiri  1.10.0 → 1.10.4  (latest 1.10.4)\n" +
				"  ▪ patch      rake      12.3.0 → 12.3.3  (latest 12.3.3)\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.render([]byte(c.in), npmTestTheme(), r)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got != c.want {
				t.Errorf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.want)
			}
		})
	}
}

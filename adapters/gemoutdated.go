package adapters

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// GemOutdated re-renders `gem outdated`. Unlike the others it has no machine
// surface — its output is already line-oriented — so we parse the one stable
// line shape it emits: "name (current < latest)".
type GemOutdated struct{}

// Handles matches `gem outdated`.
func (GemOutdated) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "gem" && argv[1] == "outdated"
}

// Render captures gem's text output and rebuilds it.
func (GemOutdated) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	data, err := runCapture("gem outdated", "gem", "outdated")
	if err != nil {
		return "", err
	}
	return renderGemOutdated(data, t, r), nil
}

// renderGemOutdated is the pure transform — table-tested offline. Each line
// looks like "rails (6.0.0 < 7.0.0)"; anything that doesn't match is ignored.
func renderGemOutdated(data []byte, t *theme.Theme, r *style.Renderer) string {
	var ups []Upgrade
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		if u, ok := parseGemLine(sc.Text()); ok {
			ups = append(ups, u)
		}
	}
	return renderUpgrades("gem outdated", ups, t, r)
}

// parseGemLine pulls name/current/latest out of "name (current < latest)".
func parseGemLine(line string) (Upgrade, bool) {
	line = strings.TrimSpace(line)
	open := strings.IndexByte(line, '(')
	closeIdx := strings.LastIndexByte(line, ')')
	if open <= 0 || closeIdx <= open {
		return Upgrade{}, false
	}
	name := strings.TrimSpace(line[:open])
	inner := line[open+1 : closeIdx]
	parts := strings.SplitN(inner, "<", 2)
	if len(parts) != 2 {
		return Upgrade{}, false
	}
	current := strings.TrimSpace(parts[0])
	latest := strings.TrimSpace(parts[1])
	if name == "" || current == "" || latest == "" {
		return Upgrade{}, false
	}
	return Upgrade{Name: name, Current: current, Wanted: latest, Latest: latest}, true
}

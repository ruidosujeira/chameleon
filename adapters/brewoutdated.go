package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// BrewOutdated re-renders `brew outdated`. A package-manager sibling: formulae
// and casks are merged into one list drawn from the shared severity ramp.
type BrewOutdated struct{}

// Handles matches `brew outdated`.
func (BrewOutdated) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "brew" && argv[1] == "outdated"
}

// brewItem mirrors one formula/cask of `brew outdated --json=v2`. Note brew's
// confusing field naming: "installed_versions" is what you HAVE and
// "current_version" is the newest AVAILABLE (i.e. our latest).
type brewItem struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

type brewRaw struct {
	Formulae []brewItem `json:"formulae"`
	Casks    []brewItem `json:"casks"`
}

// Render captures brew's JSON and rebuilds the output.
func (BrewOutdated) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	data, err := runCapture("brew outdated", "brew", append([]string{"outdated", "--json=v2"}, argv[2:]...)...)
	if err != nil {
		return "", err
	}
	return renderBrewOutdated(data, t, r)
}

// renderBrewOutdated is the pure transform — table-tested offline. brew has no
// "wanted" distinct from latest, so the install target is the latest.
func renderBrewOutdated(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		trimmed = "{}"
	}
	var raw brewRaw
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("parse brew outdated json: %w", err)
	}
	items := append(append([]brewItem{}, raw.Formulae...), raw.Casks...)
	ups := make([]Upgrade, 0, len(items))
	for _, it := range items {
		var current string
		if n := len(it.InstalledVersions); n > 0 {
			current = it.InstalledVersions[n-1] // newest installed
		}
		ups = append(ups, Upgrade{Name: it.Name, Current: current, Wanted: it.CurrentVersion, Latest: it.CurrentVersion})
	}
	return renderUpgrades("brew outdated", ups, t, r), nil
}

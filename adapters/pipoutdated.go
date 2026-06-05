package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// PipOutdated re-renders `pip list --outdated`. A package-manager sibling: it
// normalizes pip's JSON into the shared Upgrade model and reuses renderUpgrades.
type PipOutdated struct{}

// Handles matches `pip list --outdated` (also pip3, and the -o short flag).
func (PipOutdated) Handles(argv []string) bool {
	return len(argv) >= 2 &&
		(argv[0] == "pip" || argv[0] == "pip3") &&
		argv[1] == "list" &&
		hasFlag(argv, "--outdated", "-o")
}

// pipRaw mirrors one entry of `pip list --outdated --format json`.
type pipRaw struct {
	Name    string `json:"name"`
	Version string `json:"version"`        // installed
	Latest  string `json:"latest_version"` // newest available
}

// Render captures pip's JSON and rebuilds the output.
func (PipOutdated) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	data, err := runCapture("pip list", "pip", append([]string{"list", "--outdated", "--format", "json"}, argv[2:]...)...)
	if err != nil {
		return "", err
	}
	return renderPipOutdated(data, t, r)
}

// renderPipOutdated is the pure transform — table-tested offline. pip reports no
// "wanted" version, so the install target is simply the latest.
func renderPipOutdated(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		trimmed = "[]"
	}
	var raw []pipRaw
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("parse pip list json: %w", err)
	}
	ups := make([]Upgrade, 0, len(raw))
	for _, e := range raw {
		ups = append(ups, Upgrade{Name: e.Name, Current: e.Version, Wanted: e.Latest, Latest: e.Latest})
	}
	return renderUpgrades("pip list --outdated", ups, t, r), nil
}

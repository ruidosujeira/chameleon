package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// CargoOutdated re-renders `cargo outdated` (the cargo-outdated subcommand). A
// package-manager sibling drawn from the same severity ramp as npm/pip.
type CargoOutdated struct{}

// Handles matches `cargo outdated`.
func (CargoOutdated) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "cargo" && argv[1] == "outdated"
}

// cargoRaw mirrors the `cargo outdated --format json` output. cargo-outdated
// uses "project" for the installed version, "compat" for the highest
// semver-compatible release (our "wanted"), and "latest" for the newest. A
// crate with nothing newer reports "---" in these fields and is skipped.
type cargoRaw struct {
	Dependencies []struct {
		Name    string `json:"name"`
		Project string `json:"project"`
		Compat  string `json:"compat"`
		Latest  string `json:"latest"`
	} `json:"dependencies"`
}

// Render captures cargo's JSON and rebuilds the output.
func (CargoOutdated) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	data, err := runCapture("cargo outdated", "cargo", append([]string{"outdated", "--format", "json"}, argv[2:]...)...)
	if err != nil {
		return "", err
	}
	return renderCargoOutdated(data, t, r)
}

// renderCargoOutdated is the pure transform — table-tested offline.
func renderCargoOutdated(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		trimmed = "{}"
	}
	var raw cargoRaw
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("parse cargo outdated json: %w", err)
	}
	ups := make([]Upgrade, 0, len(raw.Dependencies))
	for _, d := range raw.Dependencies {
		latest := cargoField(d.Latest)
		if latest == "" {
			continue // nothing newer than what's installed
		}
		wanted := cargoField(d.Compat)
		if wanted == "" {
			wanted = latest
		}
		ups = append(ups, Upgrade{Name: d.Name, Current: cargoField(d.Project), Wanted: wanted, Latest: latest})
	}
	return renderUpgrades("cargo outdated", ups, t, r), nil
}

// cargoField normalizes cargo-outdated's "---" sentinel (meaning "none") to an
// empty string.
func cargoField(s string) string {
	if s == "---" {
		return ""
	}
	return s
}

package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// SystemctlUnits re-renders `systemctl list-units`. A state-list sibling: each
// unit's active state (active/failed/inactive/…) drives the glyph and color.
type SystemctlUnits struct{}

// Handles matches `systemctl list-units`.
func (SystemctlUnits) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "systemctl" && argv[1] == "list-units"
}

// systemdUnit mirrors one element of `systemctl list-units --output=json`.
type systemdUnit struct {
	Unit        string `json:"unit"`
	Active      string `json:"active"` // active | inactive | failed | activating | …
	Sub         string `json:"sub"`    // running | exited | dead | …
	Description string `json:"description"`
}

// Render captures the unit list as JSON and rebuilds the output. Extra args
// (e.g. --type=service, --failed) are forwarded to systemctl.
func (SystemctlUnits) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	args := append([]string{"list-units"}, argv[2:]...)
	args = append(args, "--output=json")
	data, err := runCapture("systemctl list-units", "systemctl", args...)
	if err != nil {
		return "", err
	}
	return renderSystemctlUnits(data, t, r)
}

// renderSystemctlUnits is the pure transform — table-tested offline. The detail
// column is the unit description, falling back to its low-level sub-state.
func renderSystemctlUnits(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		trimmed = "[]"
	}
	var raw []systemdUnit
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("parse systemctl json: %w", err)
	}

	items := make([]stateItem, 0, len(raw))
	for _, u := range raw {
		detail := u.Description
		if detail == "" {
			detail = u.Sub
		}
		items = append(items, stateItem{state: u.Active, name: u.Unit, detail: detail})
	}
	return renderStates("systemctl list-units", "nenhuma unidade", items, t, r), nil
}

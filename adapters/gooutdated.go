package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// GoOutdated re-renders `go list -m -u` — Go modules with a newer release. A
// package-manager sibling drawn from the shared severity ramp.
type GoOutdated struct{}

// Handles matches `go list -m -u` (the "show available module updates" intent).
func (GoOutdated) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "go" && argv[1] == "list" &&
		hasFlag(argv, "-m") && hasFlag(argv, "-u")
}

// goMod mirrors one record of `go list -m -u -json all`. Only modules that have
// an Update block are outdated; everything else is current and skipped.
type goMod struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Update  *struct {
		Version string `json:"Version"`
	} `json:"Update"`
}

// Render captures the module list and rebuilds the output. We always request
// the JSON form regardless of how the user spelled the flags.
func (GoOutdated) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	data, err := runCapture("go list", "go", "list", "-m", "-u", "-json", "all")
	if err != nil {
		return "", err
	}
	return renderGoOutdated(data, t, r)
}

// renderGoOutdated is the pure transform — table-tested offline. `go list -json`
// emits a STREAM of concatenated JSON objects (not an array), so we decode in a
// loop. Versions carry a leading 'v' that parseSemver already tolerates.
func renderGoOutdated(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var ups []Upgrade
	for {
		var m goMod
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return "", fmt.Errorf("parse go list json: %w", err)
		}
		if m.Update == nil {
			continue // module is already at the latest
		}
		ups = append(ups, Upgrade{
			Name:    m.Path,
			Current: m.Version,
			Wanted:  m.Update.Version,
			Latest:  m.Update.Version,
		})
	}
	return renderUpgrades("go list -m -u", ups, t, r), nil
}

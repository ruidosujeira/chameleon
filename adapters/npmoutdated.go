package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// NpmOutdated re-renders `npm outdated`. It is one of the package-manager
// siblings: it normalizes npm's JSON into the shared Upgrade model and hands it
// to renderUpgrades, so npm, pip, cargo, brew, go and gem all read as one family.
type NpmOutdated struct{}

// Handles matches argv that begins with `npm outdated`.
func (NpmOutdated) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "npm" && argv[1] == "outdated"
}

// npmRaw mirrors one entry of `npm outdated --json`. `current` may be absent
// (package not installed yet), so it is a plain string defaulting to "".
type npmRaw struct {
	Current string `json:"current"`
	Wanted  string `json:"wanted"`
	Latest  string `json:"latest"`
}

// decodeEntry decodes one `npm outdated --json` value. npm normally gives a
// single object per package, but in a WORKSPACE it gives an array of objects
// (one per dependent workspace). We accept both, collapsing an array to its
// first entry — enough for one aligned row per package. A value that is neither
// (null, number, …) is reported as not-ok and skipped by the caller.
func decodeEntry(msg json.RawMessage) (npmRaw, bool) {
	trimmed := bytes.TrimSpace(msg)
	if len(trimmed) == 0 {
		return npmRaw{}, false
	}
	switch trimmed[0] {
	case '{':
		var obj npmRaw
		if err := json.Unmarshal(trimmed, &obj); err == nil {
			return obj, true
		}
	case '[':
		var arr []npmRaw
		if err := json.Unmarshal(trimmed, &arr); err == nil && len(arr) > 0 {
			return arr[0], true
		}
	}
	return npmRaw{}, false
}

// Render captures `npm outdated --json` and rebuilds the output. Extra args the
// user passes (e.g. `chameleon npm outdated -g`) are forwarded to npm.
func (NpmOutdated) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	data, err := captureOutdated(argv)
	if err != nil {
		return "", err
	}
	return renderOutdated(data, t, r)
}

// captureOutdated runs npm and returns its stdout. IMPORTANT: `npm outdated`
// exits with code 1 when there ARE outdated packages — that is success for us,
// not an error, and stdout still holds the JSON. runCapture encodes exactly that
// "useful non-zero exit" rule, shared by every package-manager adapter.
func captureOutdated(argv []string) ([]byte, error) {
	args := append([]string{"outdated", "--json"}, argv[2:]...)
	return runCapture("npm outdated", "npm", args...)
}

// renderOutdated turns npm's raw JSON into the styled output. It is pure (no
// exec) so it can be table-tested against fixtures offline. The drawing itself
// lives in the shared renderUpgrades; this function only normalizes npm's shape.
func renderOutdated(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		trimmed = "{}"
	}

	// Each value is normally an object, but in a workspace npm emits an array
	// of objects — decode lazily and accept both (see decodeEntry).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("parse npm outdated json: %w", err)
	}

	ups := make([]Upgrade, 0, len(raw))
	for name, msg := range raw {
		e, ok := decodeEntry(msg)
		if !ok {
			continue
		}
		ups = append(ups, Upgrade{Name: name, Current: e.Current, Wanted: e.Wanted, Latest: e.Latest})
	}

	return renderUpgrades("npm outdated", ups, t, r), nil
}

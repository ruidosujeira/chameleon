package adapters

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// NpmOutdated re-renders `npm outdated`. It is the sibling of GitStatus: same
// shared theme, same glyph-and-color-per-state model, same column discipline —
// so npm and git read as one visual family.
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

// npmPkg is a parsed, classified outdated package.
type npmPkg struct {
	name    string
	state   string // major | minor | patch | update
	current string
	wanted  string
	latest  string
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
// not an error, and stdout still holds the JSON. We only surface a real
// failure: npm missing/couldn't start, or an ExitError with empty stdout (e.g.
// run outside a project) — otherwise we'd silently report "up to date".
func captureOutdated(argv []string) ([]byte, error) {
	args := append([]string{"outdated", "--json"}, argv[2:]...)
	out, err := exec.Command("npm", args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return nil, fmt.Errorf("npm outdated: %w", err) // couldn't start npm
		}
		if len(bytes.TrimSpace(out)) == 0 {
			if msg := strings.TrimSpace(string(ee.Stderr)); msg != "" {
				return nil, fmt.Errorf("npm outdated: %s", msg)
			}
			return nil, fmt.Errorf("npm outdated: %w", err)
		}
		// Non-empty stdout with exit 1: outdated packages found — proceed.
	}
	return out, nil
}

// renderOutdated turns the raw JSON into the styled, aligned output. It is pure
// (no exec) so it can be table-tested against fixtures offline.
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

	pkgs := make([]npmPkg, 0, len(raw))
	for name, msg := range raw {
		e, ok := decodeEntry(msg)
		if !ok {
			continue
		}
		pkgs = append(pkgs, npmPkg{
			name:    name,
			state:   classifyVersion(e.Current, e.Latest),
			current: e.Current,
			wanted:  e.Wanted,
			latest:  e.Latest,
		})
	}
	// Deterministic order: most urgent first, then by name.
	sort.Slice(pkgs, func(i, j int) bool {
		if ri, rj := stateRank(pkgs[i].state), stateRank(pkgs[j].state); ri != rj {
			return ri < rj
		}
		return pkgs[i].name < pkgs[j].name
	})

	var sb strings.Builder

	// Command line: prompt glyph + the command — identical grammar to git.
	sb.WriteString(r.Paint(t.Colors["prompt"], t.Glyphs["prompt"]))
	sb.WriteString(" ")
	sb.WriteString(r.Paint(t.Colors["command"], "npm outdated"))
	sb.WriteString("\n")

	// Nothing outdated → a single clean line, mirroring git's "tudo limpo".
	if len(pkgs) == 0 {
		sb.WriteString(t.Layout.Indent)
		sb.WriteString(r.Paint(t.Colors["staged"], t.Glyphs["clean"]+" tudo atualizado"))
		sb.WriteString("\n")
		return sb.String(), nil
	}

	// Column widths from raw (un-painted) display text — pad BEFORE paint.
	var nameW, curW, wantW int
	for i := range pkgs {
		nameW = max(nameW, style.Width(pkgs[i].name))
		curW = max(curW, style.Width(displayCurrent(pkgs[i].current)))
		wantW = max(wantW, style.Width(pkgs[i].wanted))
	}

	for _, p := range pkgs {
		c := npmStateColor(t, p.state)
		label := style.PadRight(p.state, t.Layout.LabelWidth)
		cur := style.PadRight(displayCurrent(p.current), curW)
		want := style.PadRight(p.wanted, wantW)

		sb.WriteString(t.Layout.Indent)
		sb.WriteString(r.Paint(c, npmGlyph(t, p.state)+" "+label)) // glyph+label in state color
		sb.WriteString(" ")
		sb.WriteString(r.Paint(t.Colors["path"], style.PadRight(p.name, nameW)))
		sb.WriteString("  ")
		sb.WriteString(r.Paint(t.Colors["dim"], cur))
		sb.WriteString(" ")
		sb.WriteString(r.Paint(t.Colors["dim"], "→"))
		sb.WriteString(" ")
		sb.WriteString(r.Paint(c, want)) // the target version pops in the severity color
		sb.WriteString("  ")
		sb.WriteString(r.Paint(t.Colors["dim"], "(latest "+p.latest+")"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// displayCurrent renders a possibly-absent current version. A missing current
// (package not installed) shows as an em dash.
func displayCurrent(current string) string {
	if current == "" {
		return "—"
	}
	return current
}

// classifyVersion compares current → latest by semver (major.minor.patch) and
// returns the severity state. A missing current, a non-numeric/prerelease
// version, or no detectable upgrade all fall back to the generic "update".
func classifyVersion(current, latest string) string {
	if current == "" {
		return "update"
	}
	c, ok1 := parseSemver(current)
	l, ok2 := parseSemver(latest)
	if !ok1 || !ok2 {
		return "update"
	}
	switch {
	case l[0] > c[0]:
		return "major"
	case l[0] == c[0] && l[1] > c[1]:
		return "minor"
	case l[0] == c[0] && l[1] == c[1] && l[2] > c[2]:
		return "patch"
	default:
		return "update"
	}
}

// parseSemver splits "major.minor.patch" into ints. Missing trailing
// components default to 0 (e.g. "1.2" → [1,2,0]). Any non-numeric component
// (a prerelease/build suffix like "1.2.3-beta") or more than three components
// makes it un-parseable (ok=false), so the caller degrades to "update".
func parseSemver(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// stateRank orders states most-urgent-first for stable output.
func stateRank(state string) int {
	switch state {
	case "major":
		return 0
	case "minor":
		return 1
	case "patch":
		return 2
	default: // update
		return 3
	}
}

// npmStateColor resolves the color for a state, preferring an explicit theme
// key and falling back to a related existing color when a theme omits one.
func npmStateColor(t *theme.Theme, state string) style.Color {
	if c, ok := t.Colors[state]; ok {
		return c
	}
	switch state {
	case "major":
		return t.Colors["behind"]
	case "minor":
		return t.Colors["modified"]
	case "patch":
		return t.Colors["renamed"]
	default:
		return t.Colors["branch"]
	}
}

// npmGlyph resolves the glyph for a state, falling back to a generic up-arrow.
func npmGlyph(t *theme.Theme, state string) string {
	if g, ok := t.Glyphs[state]; ok && g != "" {
		return g
	}
	return "↑"
}

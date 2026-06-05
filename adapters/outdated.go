package adapters

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// Upgrade is one outdated dependency in a tool-agnostic shape. Every package
// manager adapter (npm, pip, cargo, brew, go, gem) normalizes its machine
// output into a slice of these and hands them to renderUpgrades — which is why
// they all come out looking like one family, drawn from the same severity ramp.
type Upgrade struct {
	Name    string
	Current string // installed version ("" if not installed → shown as an em dash)
	Wanted  string // version installed by an update now; when == Current the arrow is suppressed
	Latest  string // newest available
}

// renderUpgrades draws the shared "outdated" layout for ANY package manager. It
// is pure (no exec) so every adapter that feeds it can be table-tested offline.
//
// Each row is:
//
//	<indent><severity glyph> <severity label>  <name>  <current> → <wanted>  (latest …)
//
// Severity (major/minor/patch/update) is classified from current→latest, and
// both the glyph and the color come from the shared theme — so npm, pip, cargo
// and friends are visually indistinguishable apart from the command line.
func renderUpgrades(command string, ups []Upgrade, t *theme.Theme, r *style.Renderer) string {
	type row struct {
		Upgrade
		state string
	}
	rows := make([]row, 0, len(ups))
	for _, u := range ups {
		rows = append(rows, row{u, classifyVersion(u.Current, u.Latest)})
	}
	// Deterministic order: most urgent first, then by name.
	sort.Slice(rows, func(i, j int) bool {
		if ri, rj := stateRank(rows[i].state), stateRank(rows[j].state); ri != rj {
			return ri < rj
		}
		return rows[i].Name < rows[j].Name
	})

	var sb strings.Builder

	// Command line: prompt glyph + the command — identical grammar to git.
	sb.WriteString(styled(t, r, "prompt", t.Glyphs["prompt"]))
	sb.WriteString(" ")
	sb.WriteString(styled(t, r, "command", command))
	sb.WriteString("\n")

	// Nothing outdated → a single calm line, mirroring git's "tudo limpo".
	if len(rows) == 0 {
		sb.WriteString(t.Layout.Indent)
		sb.WriteString(styled(t, r, "staged", t.Glyphs["clean"]+" tudo atualizado"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Column widths from raw (un-painted) display text — pad BEFORE paint.
	var nameW, curW, wantW int
	for i := range rows {
		nameW = max(nameW, style.Width(rows[i].Name))
		curW = max(curW, style.Width(displayCurrent(rows[i].Current)))
		wantW = max(wantW, style.Width(rows[i].Wanted))
	}

	for _, p := range rows {
		c := upgradeColor(t, p.state)
		label := style.PadRight(p.state, t.Layout.LabelWidth)
		cur := style.PadRight(displayCurrent(p.Current), curW)

		sb.WriteString(t.Layout.Indent)
		sb.WriteString(r.Paint(c, upgradeGlyph(t, p.state)+" "+label)) // glyph+label in severity color
		sb.WriteString(" ")
		sb.WriteString(styled(t, r, "path", style.PadRight(p.Name, nameW)))
		sb.WriteString("  ")
		sb.WriteString(styled(t, r, "dim", cur))
		if p.Current == p.Wanted {
			// current already satisfies the range: "X → X" is a no-op that reads
			// like a bug. Blank the arrow+target, keep the (latest …) aligned.
			sb.WriteString(strings.Repeat(" ", wantW+3)) // width of " → " + padded want
		} else {
			sb.WriteString(" ")
			sb.WriteString(styled(t, r, "dim", "→"))
			sb.WriteString(" ")
			sb.WriteString(r.Paint(c, style.PadRight(p.Wanted, wantW))) // target pops in severity color
		}
		sb.WriteString("  ")
		sb.WriteString(styled(t, r, "dim", "(latest "+p.Latest+")"))
		sb.WriteString("\n")
	}

	return sb.String()
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

// parseSemver splits "major.minor.patch" into ints. A leading 'v' (Go modules
// write "v1.2.3") is tolerated. Missing trailing components default to 0 (e.g.
// "1.2" → [1,2,0]). Any non-numeric component (a prerelease/build suffix like
// "1.2.3-beta") or more than three components makes it un-parseable (ok=false),
// so the caller degrades to "update".
func parseSemver(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
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

// upgradeColor resolves the color for a severity state, preferring an explicit
// theme key and falling back to a related existing color when a theme omits one.
func upgradeColor(t *theme.Theme, state string) style.Color {
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

// upgradeGlyph resolves the glyph for a severity state, falling back to a
// generic up-arrow.
func upgradeGlyph(t *theme.Theme, state string) string {
	if g, ok := t.Glyphs[state]; ok && g != "" {
		return g
	}
	return "↑"
}

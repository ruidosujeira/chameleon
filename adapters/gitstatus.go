// Package adapters holds the per-tool renderers. Each adapter captures a
// tool's MACHINE output (porcelain/json) and re-renders it from scratch in the
// shared theme — it never recolors the human-facing text in place.
package adapters

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// GitStatus re-renders `git status`. It deliberately targets only the
// NON-DIFF surface of git: branch state and the staged/worktree/untracked
// file lists. Diffs are delta's territory and we do not touch them.
type GitStatus struct{}

// Handles matches argv that begins with `git status`.
func (GitStatus) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "git" && argv[1] == "status"
}

// entry is one changed path plus the state code that produced it (the code
// doubles as both the glyph key and the human label).
type entry struct {
	code string
	path string
}

// Render shells out to porcelain v2 and rebuilds the output from the parsed
// machine form.
func (GitStatus) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	out, err := exec.Command("git", "status", "--porcelain=v2", "--branch").Output()
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}

	var (
		branch    string
		upstream  string
		ahead     int
		behind    int
		staged    []entry
		worktree  []entry
		untracked []entry
	)

	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case '#':
			// Header: "# <key> <value...>".
			f := strings.Fields(line)
			if len(f) < 3 {
				continue
			}
			switch f[1] {
			case "branch.head":
				branch = f[2]
			case "branch.upstream":
				upstream = f[2]
			case "branch.ab":
				// "# branch.ab +N -M"
				if len(f) >= 4 {
					ahead, _ = strconv.Atoi(strings.TrimPrefix(f[2], "+"))
					behind, _ = strconv.Atoi(strings.TrimPrefix(f[3], "-"))
				}
			}
		case '1':
			// Ordinary: "1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>".
			f := strings.SplitN(line, " ", 9)
			if len(f) < 9 {
				continue
			}
			classify(f[1], f[8], &staged, &worktree)
		case '2':
			// Renamed/copied: "2 <XY> ... <Xscore> <path>\t<orig>".
			f := strings.SplitN(line, " ", 10)
			if len(f) < 10 {
				continue
			}
			path := f[9]
			if i := strings.IndexByte(path, '\t'); i >= 0 {
				path = path[:i] // keep the new name, drop the original
			}
			classify(f[1], path, &staged, &worktree)
		case '?':
			// Untracked: "? <path>".
			f := strings.SplitN(line, " ", 2)
			if len(f) == 2 {
				untracked = append(untracked, entry{code: "untracked", path: f[1]})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("scan git status: %w", err)
	}

	return draw(t, r, branch, upstream, ahead, behind, staged, worktree, untracked), nil
}

// mapCode turns a porcelain v2 status char into a theme glyph/label key. '.'
// (unchanged) and anything unknown map to "" and are dropped by the caller.
func mapCode(c byte) string {
	switch c {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'T':
		return "typechange"
	}
	return ""
}

// classify splits an XY status field into a staged (index, X) entry and a
// worktree (Y) entry, skipping the '.' = unchanged side.
func classify(xy, path string, staged, worktree *[]entry) {
	if len(xy) != 2 {
		return
	}
	if code := mapCode(xy[0]); code != "" {
		*staged = append(*staged, entry{code: code, path: path})
	}
	if code := mapCode(xy[1]); code != "" {
		*worktree = append(*worktree, entry{code: code, path: path})
	}
}

// draw assembles the final styled output.
func draw(
	t *theme.Theme, r *style.Renderer,
	branch, upstream string, ahead, behind int,
	staged, worktree, untracked []entry,
) string {
	var sb strings.Builder

	// Command line: prompt glyph + the command.
	sb.WriteString(r.Paint(t.Colors["prompt"], t.Glyphs["prompt"]))
	sb.WriteString(" ")
	sb.WriteString(r.Paint(t.Colors["command"], "git status"))
	sb.WriteString("\n")

	// Branch line: branch glyph + name, then ahead/behind, then upstream dim.
	sb.WriteString(t.Layout.Indent)
	sb.WriteString(r.Paint(t.Colors["branch"], t.Glyphs["branch"]+" "+branch))
	if ahead > 0 {
		sb.WriteString(" ")
		sb.WriteString(r.Paint(t.Colors["ahead"], fmt.Sprintf("%s%d", t.Glyphs["ahead"], ahead)))
	}
	if behind > 0 {
		sb.WriteString(" ")
		sb.WriteString(r.Paint(t.Colors["behind"], fmt.Sprintf("%s%d", t.Glyphs["behind"], behind)))
	}
	if upstream != "" {
		sb.WriteString(" ")
		sb.WriteString(r.Paint(t.Colors["dim"], upstream))
	}
	sb.WriteString("\n")

	// Nothing changed → a single clean line and we're done.
	if len(staged) == 0 && len(worktree) == 0 && len(untracked) == 0 {
		sb.WriteString(t.Layout.Indent)
		sb.WriteString(r.Paint(t.Colors["staged"], t.Glyphs["clean"]+" tudo limpo"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Three grouped blocks. Layout groups by stage, but COLOR is per state
	// (added green, deleted red, modified orange…) so add and delete never
	// look alike just because both are staged.
	writeBlock(&sb, t, r, staged)
	writeBlock(&sb, t, r, worktree)
	writeBlock(&sb, t, r, untracked)

	return sb.String()
}

// codeColor resolves the color for a state. It prefers an explicit per-state
// color from the theme and falls back to a sensible related color when a
// theme omits one.
func codeColor(t *theme.Theme, code string) style.Color {
	if c, ok := t.Colors[code]; ok {
		return c
	}
	switch code {
	case "modified", "typechange":
		return t.Colors["modified"]
	case "untracked":
		return t.Colors["untracked"]
	default:
		return t.Colors["staged"]
	}
}

// writeBlock renders one group of entries. Each line is:
//
//	<indent><state glyph> <label padded to LabelWidth>  <path>
//
// The glyph+label color comes from the entry's own state (see codeColor).
// Note the PadRight-before-Paint ordering required by style.PadRight.
func writeBlock(sb *strings.Builder, t *theme.Theme, r *style.Renderer, entries []entry) {
	for _, e := range entries {
		c := codeColor(t, e.code)
		label := style.PadRight(e.code, t.Layout.LabelWidth) // pad raw text...
		left := r.Paint(c, t.Glyphs[e.code]+" "+label)       // ...THEN paint.
		sb.WriteString(t.Layout.Indent)
		sb.WriteString(left)
		sb.WriteString(" ")
		sb.WriteString(r.Paint(t.Colors["path"], e.path))
		sb.WriteString("\n")
	}
}

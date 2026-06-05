package adapters

import (
	"sort"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// stateItem is one row of a state-list adapter — a pod, a container, a CI run, a
// service: a state keyword (which drives the glyph and color), a primary name,
// and an optional dim detail column.
type stateItem struct {
	state  string
	name   string
	detail string
}

// renderStates draws the shared state-list layout used by kubectl, docker, gh
// and systemctl — the visual cousin of the package-manager family. It is pure
// (no exec) so every adapter feeding it can be table-tested offline.
//
// Each row is:
//
//	<indent><state glyph> <state label>  <name>  <detail dim>
//
// Rows are ordered worst-first (failing states float to the top) then by name,
// so an unhealthy pod or a red CI run never hides below the healthy ones.
func renderStates(command, emptyMsg string, items []stateItem, t *theme.Theme, r *style.Renderer) string {
	sort.SliceStable(items, func(i, j int) bool {
		if ri, rj := severityRank(items[i].state), severityRank(items[j].state); ri != rj {
			return ri < rj
		}
		return items[i].name < items[j].name
	})

	var sb strings.Builder

	// Command line: prompt glyph + the command — identical grammar to git/npm.
	sb.WriteString(styled(t, r, "prompt", t.Glyphs["prompt"]))
	sb.WriteString(" ")
	sb.WriteString(styled(t, r, "command", command))
	sb.WriteString("\n")

	if len(items) == 0 {
		sb.WriteString(t.Layout.Indent)
		sb.WriteString(styled(t, r, "dim", t.Glyphs["clean"]+" "+emptyMsg))
		sb.WriteString("\n")
		return sb.String()
	}

	// Column widths from raw (un-painted) display text — pad BEFORE paint. The
	// label column grows past LabelWidth when a state keyword is long (k8s loves
	// "CrashLoopBackOff") so the name column always lines up.
	labelW, nameW := t.Layout.LabelWidth, 0
	for i := range items {
		labelW = max(labelW, style.Width(items[i].state))
		nameW = max(nameW, style.Width(items[i].name))
	}

	for _, it := range items {
		c := stateColor(t, it.state)
		label := style.PadRight(it.state, labelW)

		sb.WriteString(t.Layout.Indent)
		sb.WriteString(r.Paint(c, stateGlyph(t, it.state)+" "+label)) // glyph+label in state color
		sb.WriteString(" ")
		sb.WriteString(styled(t, r, "path", style.PadRight(it.name, nameW)))
		if it.detail != "" {
			sb.WriteString("  ")
			sb.WriteString(styled(t, r, "dim", it.detail))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// severityRank orders states worst-first: bad → warn → ok → unknown.
func severityRank(state string) int {
	switch stateSeverity(state) {
	case "bad":
		return 0
	case "warn":
		return 1
	case "ok":
		return 2
	default:
		return 3
	}
}

// stateSeverity buckets an arbitrary state keyword (from any of the tools) into
// one of ok/warn/bad/unknown. Matching is case-insensitive so "Running",
// "running" and "RUNNING" all land together.
func stateSeverity(state string) string {
	switch strings.ToLower(state) {
	case "running", "active", "success", "succeeded", "up", "ready",
		"available", "healthy", "completed", "listening", "mounted", "plugged":
		return "ok"
	case "pending", "restarting", "queued", "in_progress", "created", "paused",
		"waiting", "reloading", "activating", "deactivating", "containercreating",
		"terminating", "podinitializing", "neutral", "skipped":
		return "warn"
	case "failed", "failure", "error", "crashloopbackoff", "exited", "dead",
		"cancelled", "canceled", "unhealthy", "oomkilled", "imagepullbackoff",
		"errimagepull", "evicted", "timed_out", "startup_failure", "removing":
		return "bad"
	default:
		return "unknown"
	}
}

// stateColor resolves the color for a state. It prefers an explicit per-state
// theme key (so a theme can special-case "running"), then the severity-bucket
// key (ok/warn/bad/unknown), then a hardcoded fallback from the git palette.
func stateColor(t *theme.Theme, state string) style.Color {
	if c, ok := t.Colors[strings.ToLower(state)]; ok {
		return c
	}
	sev := stateSeverity(state)
	if c, ok := t.Colors[sev]; ok {
		return c
	}
	switch sev {
	case "ok":
		return t.Colors["staged"]
	case "warn":
		return t.Colors["modified"]
	case "bad":
		return t.Colors["behind"]
	default:
		return t.Colors["dim"]
	}
}

// stateGlyph resolves the glyph for a state, following the same precedence as
// stateColor: explicit per-state key, then severity bucket, then a fallback.
func stateGlyph(t *theme.Theme, state string) string {
	if g, ok := t.Glyphs[strings.ToLower(state)]; ok && g != "" {
		return g
	}
	sev := stateSeverity(state)
	if g, ok := t.Glyphs[sev]; ok && g != "" {
		return g
	}
	switch sev {
	case "ok":
		return "●"
	case "warn":
		return "◐"
	case "bad":
		return "✗"
	default:
		return "○"
	}
}

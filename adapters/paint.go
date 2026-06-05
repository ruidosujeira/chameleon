package adapters

import (
	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// styled paints text in the theme's full styling for a named STRUCTURAL key —
// its color plus any bold/dim/italic/underline from the theme's [styles] table.
// Per-state colors that don't map to a single theme key (added/major/running/…)
// keep painting with r.Paint directly; styles are for the fixed scaffolding
// (prompt, command, branch, path, dim, …).
func styled(t *theme.Theme, r *style.Renderer, key, text string) string {
	return r.PaintStyle(t.Style(key), text)
}

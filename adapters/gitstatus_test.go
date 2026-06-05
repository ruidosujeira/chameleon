package adapters

import (
	"testing"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// gitTestTheme is a minimal theme carrying every glyph the git adapter draws.
// The renderer is disabled in tests, so colors don't affect the output text and
// the Colors map can stay empty (codeColor degrades gracefully).
func gitTestTheme() *theme.Theme {
	t := &theme.Theme{
		Colors: map[string]style.Color{},
		Glyphs: map[string]string{
			"prompt": "❯", "branch": "⎇", "ahead": "↑", "behind": "↓",
			"clean": "✓", "staged": "✓", "conflict": "≠",
			"added": "✚", "deleted": "✗", "modified": "●",
			"renamed": "»", "copied": "⊕", "typechange": "~", "untracked": "?",
		},
	}
	t.Layout.Indent = "  "
	t.Layout.LabelWidth = 10
	return t
}

func TestRenderStatus(t *testing.T) {
	// Disabled renderer → deterministic, escape-free golden output.
	r := &style.Renderer{Enabled: false}

	cases := []struct {
		name      string
		porcelain string
		want      string
	}{
		{
			// Clean tree with an upstream and no ahead/behind: branch line shows
			// the upstream dim, then the single calm "tudo limpo" line.
			name: "clean with upstream",
			porcelain: "# branch.oid abc123\n" +
				"# branch.head main\n" +
				"# branch.upstream origin/main\n" +
				"# branch.ab +0 -0\n",
			want: "❯ git status\n" +
				"  ⎇ main origin/main\n" +
				"  ✓ tudo limpo\n",
		},
		{
			// A fresh branch with no upstream: no ahead/behind, no upstream label.
			name:      "no upstream branch",
			porcelain: "# branch.head feature\n",
			want: "❯ git status\n" +
				"  ⎇ feature\n" +
				"  ✓ tudo limpo\n",
		},
		{
			// The full surface: ahead/behind, a staged add + delete + rename, a
			// worktree-only modify, and an untracked file. Exercises classify
			// (staged vs worktree split), the type-2 rename path (new name kept,
			// original after the tab dropped), and column alignment across the
			// varying label widths (added/deleted/renamed/modified/untracked).
			name: "mixed changes",
			porcelain: "# branch.oid abc123\n" +
				"# branch.head main\n" +
				"# branch.upstream origin/main\n" +
				"# branch.ab +2 -1\n" +
				"1 A. N... 000000 100644 100644 0000000 1111111 parser.go\n" +
				"1 D. N... 100644 000000 000000 2222222 0000000 old.txt\n" +
				"1 .M N... 100644 100644 100644 3333333 3333333 app.go\n" +
				"2 R. N... 100644 100644 100644 4444444 4444444 R100 new.txt\torig.txt\n" +
				"? debug.tmp\n",
			want: "❯ git status\n" +
				"  ⎇ main ↑2 ↓1 origin/main\n" +
				"  ✚ added      parser.go\n" +
				"  ✗ deleted    old.txt\n" +
				"  » renamed    new.txt\n" +
				"  ● modified   app.go\n" +
				"  ? untracked  debug.tmp\n",
		},
		{
			// Merge conflict: a porcelain `u` line. It must surface (a conflict
			// block, drawn first) rather than being silently dropped — the bug
			// the old parser had, where a conflicted tree looked clean.
			name: "merge conflict surfaces",
			porcelain: "# branch.head main\n" +
				"u UU N... 100644 100644 100644 100644 h1 h2 h3 merged.txt\n" +
				"1 .M N... 100644 100644 100644 aaa bbb app.go\n",
			want: "❯ git status\n" +
				"  ⎇ main\n" +
				"  ≠ conflict   merged.txt\n" +
				"  ● modified   app.go\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := renderStatus([]byte(c.porcelain), false, gitTestTheme(), r)
			if err != nil {
				t.Fatalf("renderStatus: %v", err)
			}
			if got != c.want {
				t.Errorf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.want)
			}
		})
	}
}

func TestRenderStatusCompact(t *testing.T) {
	r := &style.Renderer{Enabled: false}

	cases := []struct {
		name      string
		porcelain string
		want      string
	}{
		{
			// Counts per group, each with its own glyph: 3 staged (✚), 1 worktree
			// (●), 1 untracked (?), plus ahead/behind — all on one line.
			name: "counts on one line",
			porcelain: "# branch.head main\n# branch.ab +2 -1\n" +
				"1 A. N... 000000 100644 100644 0 1 parser.go\n" +
				"1 D. N... 100644 000000 000000 2 0 old.txt\n" +
				"1 .M N... 100644 100644 100644 3 3 app.go\n" +
				"2 R. N... 100644 100644 100644 4 4 R100 new.txt\torig.txt\n" +
				"? debug.tmp\n",
			want: "⎇ main ↑2 ↓1  ✚3  ●1  ?1\n",
		},
		{
			// Clean tree collapses to "<branch> ✓".
			name:      "clean collapses to a checkmark",
			porcelain: "# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -0\n",
			want:      "⎇ main ✓\n",
		},
		{
			name:      "conflict count shows",
			porcelain: "# branch.head main\nu UU N... 100644 100644 100644 100644 h1 h2 h3 merged.txt\n? scratch.tmp\n",
			want:      "⎇ main  ≠1  ?1\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := renderStatus([]byte(c.porcelain), true, gitTestTheme(), r)
			if err != nil {
				t.Fatalf("renderStatus: %v", err)
			}
			if got != c.want {
				t.Errorf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.want)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name         string
		xy, path     string
		wantStaged   []entry
		wantWorktree []entry
	}{
		{"staged add only", "A.", "x", []entry{{"added", "x"}}, nil},
		{"worktree modify only", ".M", "x", nil, []entry{{"modified", "x"}}},
		{"modified on both sides", "MM", "x",
			[]entry{{"modified", "x"}}, []entry{{"modified", "x"}}},
		{"unchanged is dropped", "..", "x", nil, nil},
		{"unknown code is dropped", "X.", "x", nil, nil},
		{"malformed field is ignored", "M", "x", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var staged, worktree []entry
			classify(c.xy, c.path, &staged, &worktree)
			if !sameEntries(staged, c.wantStaged) {
				t.Errorf("staged = %v, want %v", staged, c.wantStaged)
			}
			if !sameEntries(worktree, c.wantWorktree) {
				t.Errorf("worktree = %v, want %v", worktree, c.wantWorktree)
			}
		})
	}
}

func sameEntries(a, b []entry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

# Contributing to 🦎 Chameleon

Thanks for being here. The fastest, most useful way to contribute is to **add an
adapter** — teach Chameleon to re-render one more tool. This guide walks you
through it end to end, using the two adapters that already ship as worked
references.

If you came from the launch post asking *"what would you want themed?"* — the
answer is: open an issue describing the tool, or better, send the adapter. This
document is the on-ramp.

---

## The one idea behind every adapter

Chameleon does **not** recolor text with regex. For each tool it:

1. Calls the tool's **machine surface** (`--json`, `--porcelain`, …).
2. **Parses** that structured output into a small typed model.
3. **Re-renders** the whole thing from scratch, pulling every glyph and color
   from the **shared theme**.

So `git`, `npm`, and whatever you add all come out looking like siblings,
because they drink from the same theme file. Keep that contract and your
adapter will fit right in.

---

## Quick start

Requires **Go 1.22+**. No runtime dependencies beyond the TOML parser; the
binary imports **zero** Charm packages (VHS is a docs-only dev tool).

```sh
git clone https://github.com/ruidosujeira/chameleon && cd chameleon
go build -o chameleon .          # build
go test ./...                    # run the suite
go vet ./...                     # static checks
./chameleon git status           # try it
```

Before opening a PR, the green bar is just: `go vet ./... && go test ./...`
(the same thing CI runs).

---

## Project layout

```
🦎 chameleon
├── style/      our OWN styling layer — zero Charm. Color · Renderer · Width · PadRight
├── theme/      loads themes/<name>.toml, resolves hex → Color
├── adapters/   one renderer per tool — gitstatus.go, npmoutdated.go (+ yours)
└── main.go     dispatcher + adapter registry
```

You'll touch `adapters/` (new file), `main.go` (one line to register), and
`themes/tokyonight.toml` (new color/glyph keys). That's it.

---

## ⚠️ The one invariant you must not break

**Always `PadRight` before `Paint`.**

`Paint` injects ANSI escape codes. If you pad *after* painting, `Width` counts
those invisible escapes as visible columns and your alignment shears apart.

```go
label := style.PadRight(state, t.Layout.LabelWidth) // 1. pad the RAW text
left  := r.Paint(color, glyph+" "+label)            // 2. THEN paint
```

Compute every column width from the **un-painted** display string, pad, *then*
paint. Both existing adapters do this — copy the pattern.

---

## Add an adapter, step by step

### 0. The contract

Every adapter implements this interface (defined in [`main.go`](main.go)):

```go
type Adapter interface {
    Handles(argv []string) bool                                       // do I want this command?
    Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) // draw it
}
```

`argv` is the command **after** `chameleon` (so `chameleon git status -s` →
`["git","status","-s"]`). The first registered adapter whose `Handles` returns
true wins; everything else runs raw.

### 1. Pick the machine surface

Find the flag that makes the tool emit structured data — never scrape the
human-facing text. Two shapes cover most tools, and we already have one of each
to copy:

| Input shape | Reference adapter | Parse with |
|---|---|---|
| Line-oriented porcelain | [`adapters/gitstatus.go`](adapters/gitstatus.go) (`git status --porcelain=v2 --branch`) | `bufio.Scanner` |
| JSON | [`adapters/npmoutdated.go`](adapters/npmoutdated.go) (`npm outdated --json`) | `encoding/json` |

### 2. Capture (exec) — keep it separate from render

Shell out and return raw bytes. **Read real tools carefully**: some signal
state through exit codes. `npm outdated` exits **1** when packages *are*
outdated — that's success for us, and stdout still holds the JSON. See
`captureOutdated` in npmoutdated.go for how to tell "useful non-zero exit" from
a real failure.

Keep capture (impure, calls `exec`) and render (pure, just transforms data) in
**separate functions**. That's what lets the renderer be table-tested offline
with no tool installed — do the same.

### 3. Parse into a small typed model

Decode into your own structs with exactly the fields you'll draw. Classify state
here, not while drawing (e.g. npmoutdated's `classifyVersion` turns
current→latest into `major | minor | patch | update`).

### 4. Re-render from the theme

The house style for a row is:

```
<indent><glyph> <label padded to LabelWidth>  <columns…>
```

Rules every adapter follows:

- **Glyph and color come from the theme, keyed by state** — never hardcode them.
  Resolve with a fallback so a theme that omits your key still works (see
  `npmStateColor` / `npmGlyph` and `codeColor`).
- **Color is per state, even when layout groups by something else.** In git,
  added is green and deleted is red, regardless of which block they sit in.
- **Column widths come from the raw display text** (`style.Width`), padded
  before painting (the invariant above).
- **Sort deterministically** so output is stable across runs (npmoutdated sorts
  most-urgent-first, then by name).
- **Have a clean/empty state.** When there's nothing to report, print one calm
  line — git's `✓ tudo limpo`, npm's `✓ tudo atualizado`. User-facing status
  strings are **pt-BR**; match that for consistency.
- **Open with the prompt line**: prompt glyph + the command, exactly like the
  others, so the family reads as one.

### 5. Add theme keys

Give your states colors and glyphs in [`themes/tokyonight.toml`](themes/tokyonight.toml):

```toml
[colors]
mystate = "#7aa2f7"

[glyphs]
mystate = "◆"
```

Pick **visually distinct** glyphs — shapes that survive a real terminal font.
(We learned this the hard way: a size-only ramp `▲ ▴ ▵` blurred together and was
replaced by distinct shapes `▲ ◆ ▪`.) Always provide a code-side fallback too,
so the adapter degrades gracefully on a theme that hasn't been updated.

### 6. Register it

One line in [`main.go`](main.go):

```go
var registry = []Adapter{
    adapters.GitStatus{},
    adapters.NpmOutdated{},
    adapters.MyTool{},   // ← add here; order matters (first match wins)
}
```

### 7. Test it

Table-test the **pure** render function against golden strings with the renderer
disabled — no escape codes, deterministic output:

```go
r := &style.Renderer{Enabled: false}
got, _ := renderMyTool([]byte(fixtureJSON), testTheme(), r)
if got != want { t.Errorf("…\n--- got ---\n%s\n--- want ---\n%s", got, want) }
```

See [`adapters/npmoutdated_test.go`](adapters/npmoutdated_test.go). Cover the
happy path, the empty/clean state, the malformed-input path, and any alignment
edge cases (varying column widths). Tip for getting golden spacing exactly
right: temporarily print each line wrapped in `|…|` so trailing spaces are
visible, then paste the truth in.

### 8. (Optional) demo

If your adapter is worth showing, extend [`demo/family.tape`](demo/family.tape)
and re-record (`go build -o chameleon . && vhs demo/family.tape`). VHS is
docs-only — it never enters the binary.

---

## Minimal skeleton to copy

```go
package adapters

import (
    "os/exec"
    "strings"

    "github.com/ruidosujeira/chameleon/style"
    "github.com/ruidosujeira/chameleon/theme"
)

type MyTool struct{}

func (MyTool) Handles(argv []string) bool {
    return len(argv) >= 2 && argv[0] == "mytool" && argv[1] == "status"
}

func (MyTool) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
    out, err := capture(argv) // impure: exec
    if err != nil {
        return "", err
    }
    return render(out, t, r)  // pure: transform — this is what you unit-test
}

func capture(argv []string) ([]byte, error) {
    return exec.Command("mytool", "status", "--json").Output()
}

func render(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
    var sb strings.Builder
    sb.WriteString(r.Paint(t.Colors["prompt"], t.Glyphs["prompt"]) + " ")
    sb.WriteString(r.Paint(t.Colors["command"], "mytool status") + "\n")
    // parse data → rows; for each row:
    //   label := style.PadRight(state, t.Layout.LabelWidth)  // pad RAW
    //   sb.WriteString(t.Layout.Indent)
    //   sb.WriteString(r.Paint(color, glyph+" "+label))      // THEN paint
    //   …columns, widths from style.Width, padded before paint…
    return sb.String(), nil
}
```

---

## PR checklist

- [ ] `go vet ./... && go test ./...` are green.
- [ ] New states have **colors and glyphs** in `themes/tokyonight.toml`, plus
      code-side fallbacks.
- [ ] Render is **pure** and table-tested (capture is separate).
- [ ] `PadRight` before `Paint` everywhere; columns align under varying widths.
- [ ] No Charm / heavy dependencies added to the binary.
- [ ] Clean/empty state prints one calm line (pt-BR, like the others).
- [ ] Commit messages are clear and present-tense; one logical change per commit.

Open a PR against `main`. Small, focused PRs get reviewed fastest. Questions?
Open an issue — half-formed ideas welcome.

Be kind, assume good faith. 🦎

<div align="center">

<img width="942" height="293" alt="image" src="https://github.com/user-attachments/assets/11e3b5bb-71cc-4438-b765-21cd68816bbc" />

**A *semantic* reformatter for terminal output.**

It captures the machine-readable output of your tools and **re-renders it from
scratch** — with a single theme shared across all of them.

![Chameleon re-rendering `git status`](demo/demo.gif)

</div>

---

## The idea

Most terminal colorizers (`grc`, `colout`, `pipecolor`) do one thing: they take
the human-readable text a tool already printed and **recolor** it with regex.
Chameleon starts somewhere else.

> 🦎 **Chameleon doesn't recolor. It re-captures and redraws.**

For each tool, Chameleon calls its **machine surface** (`--porcelain`,
`--json`, …), parses the structured data, and **re-renders the entire output**
from the theme. The original human text is discarded — what you see is drawn by
Chameleon.

And every adapter drinks from the **same theme**. So `git`, `npm`, and `docker`
come out looking like **siblings** — something `bat`, `eza`, and `delta` don't
deliver, because each carries its own theme.

| | Recolors existing text | Single theme across tools |
|---|:---:|:---:|
| `grc` / `colout` | ✅ | ❌ |
| `bat` / `eza` / `delta` | — | ❌ (per-tool theme) |
| **🦎 Chameleon** | ❌ (re-renders) | ✅ |

---

## Demo

The GIF above was recorded with [VHS](https://github.com/charmbracelet/vhs)
from [`demo/demo.tape`](demo/demo.tape), against a repository with a staged add,
a staged delete, a modified file, an untracked file, and 2 commits ahead of
`origin`:

```
❯ git status
  ⎇ main ↑2 origin/main
  ✗ deleted    old.txt
  ✚ added      parser.go
  ● modified   app.go
  ? untracked  debug.tmp
```

---

## Installation

Requires **Go 1.22+**. No runtime dependencies beyond the TOML parser.

```sh
git clone <repo> chameleon && cd chameleon
go build -o chameleon .
```

> **Note:** today the theme is resolved at `themes/<name>.toml` **relative to
> the current directory**. For global use, run from the project root or keep a
> `themes/` folder alongside. Packaging (`embed`/`~/.config`) is on the roadmap.

---

## Usage

Prefix any supported command with `chameleon`:

```sh
chameleon git status
```

A command with no matching adapter? Chameleon runs it **raw**, without
interfering (a placeholder for the future `grc`-style regex fallback):

```sh
chameleon echo "passes straight through"   # → passes straight through
```

### Environment variables

| Variable | Effect |
|---|---|
| `CHAMELEON_THEME` | Name of the theme to load (default: `tokyonight`). |
| `CHAMELEON_FORCE_COLOR=1` | Force color even without a TTY (useful in pipes/CI). |
| `NO_COLOR` | Disable all color ([no-color.org](https://no-color.org)). |

With none of these set, Chameleon enables color **only when `stdout` is a
terminal** — in a pipe it stays raw. No TTY library: just `os.Stdout.Stat()` +
`os.ModeCharDevice`.

---

## Themes 🎨

A theme is a versionable TOML file — your project's visual identity lives in it.
See [`themes/tokyonight.toml`](themes/tokyonight.toml):

```toml
name = "tokyonight"

[colors]
prompt = "#9ece6a"
branch = "#bb9af7"
ahead  = "#e0af68"
path   = "#7aa2f7"
# per-state colors (each entry is colored by its own state)
added   = "#9ece6a"
deleted = "#f7768e"
# …

[glyphs]
prompt = "❯"
branch = "⎇"
ahead  = "↑"
modified = "●"
# …

[layout]
indent = "  "
label_width = 10
```

Create `themes/dracula.toml`, swap the colors, and use it with
`CHAMELEON_THEME=dracula chameleon git status`. **Every** adapter follows along
— that's the whole point.

---

## Architecture

Four pieces, deliberately small:

```
🦎 chameleon
├── style/      our OWN styling layer — zero Charm, zero framework
│               Color · Renderer (TTY/NO_COLOR) · Width · PadRight
├── theme/      loads themes/<name>.toml and resolves hex → Color
├── adapters/   one renderer per tool (today: git status)
└── main.go     dispatcher: first adapter whose Handles() matches wins;
                otherwise, run the command raw
```

The `style` layer is the **core competency**, which is why it depends on no
Charm packages (lipgloss/bubbletea). It's just three primitives:

```go
type Color struct{ R, G, B uint8 }
func (r *Renderer) Paint(c Color, text string) string   // 24-bit truecolor
func Width(s string) int                                 // width by rune
func PadRight(s string, width int) string                // alignment
```

> ⚠️ **Critical invariant:** always `PadRight` **before** `Paint`. Painting
> first injects escape codes that `Width` would then count as visible columns,
> breaking alignment.

To add a tool, implement the contract and register it:

```go
type Adapter interface {
    Handles(argv []string) bool
    Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error)
}
```

---

## Re-recording the demo

VHS is a Charm dev tool used **only to record docs** — it is **not** a runtime
dependency; the Chameleon binary imports no Charm package.

```sh
go build -o chameleon .
vhs demo/demo.tape          # → demo/demo.gif
```

---

## Roadmap

Out of scope for now, in rough order of interest:

- [ ] Package themes (`embed.FS` / `~/.config/chameleon`)
- [ ] More adapters: `npm`, `docker`, `kubectl`, … (all on the single theme)
- [ ] `grc`-style regex fallback for commands without an adapter
- [ ] Truecolor → 256/16 color downsampling
- [ ] CJK/emoji width via `go-runewidth` (swap point already marked in `style.Width`)

**Explicitly out of scope:** diffs — that's
[`delta`](https://github.com/dandavison/delta)'s territory. Chameleon targets
only the **non-diff** surfaces of tools.

---

<div align="center">
🦎 <em>One color to theme them all.</em>
</div>

<div align="center">

<img width="942" height="293" alt="image" src="https://github.com/user-attachments/assets/11e3b5bb-71cc-4438-b765-21cd68816bbc" />

**A *semantic* reformatter for terminal output.**

It captures the machine-readable output of your tools and **re-renders it from
scratch** — with a single theme shared across all of them.

![git status and npm outdated, re-rendered from one theme](demo/family.gif)

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

## One theme, two tools

This is the whole pitch in one screen — two unrelated tools, captured from their
machine output and redrawn from the **same** theme. Same prompt glyph, same
glyph-and-color-per-state model, same aligned columns. They read as siblings:

<table>
<tr>
<th align="left"><code>chameleon git status</code></th>
<th align="left"><code>chameleon npm outdated</code></th>
</tr>
<tr valign="top">
<td>

```
❯ git status
  ⎇ main
  ✗ deleted    old.txt
  ✚ added      parser.go
  ● modified   app.go
  ? untracked  debug.tmp
```

</td>
<td>

```
❯ npm outdated
  ▲ major   chalk      4.0.0 → 4.1.2  (latest 5.6.2)
  ▲ major   is-number  6.0.0 → 6.0.0  (latest 7.0.0)
  ▴ minor   semver     7.3.0 → 7.8.1  (latest 7.8.1)
```

</td>
</tr>
</table>

The GIF above was recorded with [VHS](https://github.com/charmbracelet/vhs)
from [`demo/family.tape`](demo/family.tape). `npm outdated` is colored by upgrade
severity (major / minor / patch), read straight from the shared theme.

---

## Installation

Requires **Go 1.22+**. No runtime dependencies beyond the TOML parser.

```sh
go install github.com/ruidosujeira/chameleon@latest
```

Or build from a clone:

```sh
git clone https://github.com/ruidosujeira/chameleon && cd chameleon
go build -o chameleon .
```

The default theme is **embedded in the binary**, so `chameleon` works from any
directory out of the box — no `themes/` folder required. To override it, drop a
`.chameleon.toml` in your project (or a theme under `~/.config/chameleon/themes/`);
see [Themes](#themes-) for the full resolution order.

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

Create a theme, swap the colors, and use it with
`CHAMELEON_THEME=dracula chameleon git status`. **Every** adapter follows along
— that's the whole point.

### Where themes come from

Chameleon resolves the theme in this order — the **first** source that exists wins:

1. **`.chameleon.toml`** in the CWD or any parent up to the git root — your
   team's versionable theme; a full theme file that overrides everything.
2. **`~/.config/chameleon/themes/<name>.toml`** (respects `XDG_CONFIG_HOME`) —
   your personal theme.
3. **`./themes/<name>.toml`** relative to the CWD — dev convenience inside the repo.
4. **Embedded built-in** — compiled into the binary; the guaranteed fallback so
   it always works, from anywhere.

The theme name (steps 2–4) comes from `CHAMELEON_THEME`, defaulting to `tokyonight`.

---

## Architecture

Four pieces, deliberately small:

```
🦎 chameleon
├── style/      our OWN styling layer — zero Charm, zero framework
│               Color · Renderer (TTY/NO_COLOR) · Width · PadRight
├── theme/      loads themes/<name>.toml and resolves hex → Color
├── adapters/   one renderer per tool — git status, npm outdated
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

### Adapters today

| Command | Captures | States (glyph · color from theme) |
|---|---|---|
| `git status` | `git status --porcelain=v2 --branch` | added · deleted · modified · renamed · copied · typechange · untracked |
| `npm outdated` | `npm outdated --json` | major · minor · patch · update (by upgrade severity) |

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
vhs demo/family.tape        # → demo/family.gif
```

---

## Roadmap

Out of scope for now, in rough order of interest:

- [x] Embedded built-in theme + `.chameleon.toml` / `~/.config` override layers
- [ ] More adapters: `docker`, `kubectl`, `cargo`, … (all on the single theme)
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

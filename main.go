// Command chameleon is a semantic reformatter for terminal output. It captures
// each tool's machine output (porcelain/json) and re-renders it from scratch
// using one shared theme, so every tool comes out looking like a sibling.
package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"

	"github.com/ruidosujeira/chameleon/adapters"
	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// embeddedThemes carries the built-in themes compiled into the binary, so
// `chameleon` works from ANY directory without a themes/ folder alongside it.
// This package (main) is what sees ./themes at build time.
//
//go:embed themes/*.toml
var embeddedThemes embed.FS

// Adapter is the contract every per-tool renderer implements.
type Adapter interface {
	// Handles reports whether this adapter wants to render argv.
	Handles(argv []string) bool
	// Render produces the styled output for argv.
	Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error)
}

// registry is the ordered list of adapters; the first whose Handles returns
// true wins.
var registry = []Adapter{
	adapters.GitStatus{},
	adapters.NpmOutdated{},
}

func main() {
	argv := os.Args[1:]

	t, err := theme.Load(theme.Name(), embeddedThemes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chameleon:", err)
		os.Exit(1)
	}
	r := style.New()

	for _, a := range registry {
		if !a.Handles(argv) {
			continue
		}
		out, err := a.Render(argv, t, r)
		if err != nil {
			fmt.Fprintln(os.Stderr, "chameleon:", err)
			os.Exit(1)
		}
		fmt.Print(out)
		return
	}

	// No adapter matched: run the command raw. This is a placeholder for the
	// future grc-style regex fallback that will recolor unknown commands.
	if len(argv) == 0 {
		return
	}
	runRaw(argv)
}

func runRaw(argv []string) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "chameleon:", err)
		os.Exit(1)
	}
}

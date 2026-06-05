package adapters

import (
	"testing"

	"github.com/ruidosujeira/chameleon/style"
)

// The adapters consume the output of external tools we don't control, so the
// parsers must never panic on malformed or hostile input — at worst they return
// an error. These fuzz tests assert exactly that (the seed corpus also runs as
// ordinary cases under `go test`, no -fuzz needed).

func FuzzRenderStatus(f *testing.F) {
	f.Add("# branch.head main\n# branch.ab +1 -2\n1 .M N... 100644 100644 100644 a b app.go\n? x.tmp\n")
	f.Add("u UU N... 100644 100644 100644 100644 a b c merged.txt\n")
	f.Add("2 R. N... 100644 100644 100644 a b R100 new\told\n")
	f.Add("")
	f.Add("garbage\n#\n1\n2\nu\n?\n")
	r := &style.Renderer{Enabled: false}
	th := gitTestTheme()
	f.Fuzz(func(t *testing.T, data string) {
		_, _ = renderStatus([]byte(data), false, th, r)
		_, _ = renderStatus([]byte(data), true, th, r)
	})
}

func FuzzRenderOutdated(f *testing.F) {
	f.Add(`{"chalk":{"current":"4.0.0","wanted":"4.1.0","latest":"5.0.0"}}`)
	f.Add(`{"x":[{"current":"1.0.0","wanted":"1.0.0","latest":"2.0.0"}]}`)
	f.Add(`{}`)
	f.Add(`[]`)
	f.Add(`{not json`)
	r := &style.Renderer{Enabled: false}
	th := npmTestTheme()
	f.Fuzz(func(t *testing.T, data string) {
		_, _ = renderOutdated([]byte(data), th, r)
	})
}

func FuzzRenderDockerPS(f *testing.F) {
	f.Add("{\"Names\":\"db\",\"State\":\"running\",\"Status\":\"Up\"}\n{\"Names\":\"c\",\"State\":\"exited\"}")
	f.Add("not ndjson")
	f.Add("")
	r := &style.Renderer{Enabled: false}
	th := stateTestTheme()
	f.Fuzz(func(t *testing.T, data string) {
		_, _ = renderDockerPS([]byte(data), th, r)
	})
}

func FuzzRenderKubectlPods(f *testing.F) {
	f.Add(`{"items":[{"metadata":{"name":"p"},"status":{"phase":"Running"}}]}`)
	f.Add(`{"items":null}`)
	f.Add(`{`)
	r := &style.Renderer{Enabled: false}
	th := stateTestTheme()
	f.Fuzz(func(t *testing.T, data string) {
		_, _ = renderKubectlPods([]byte(data), th, r)
	})
}

func FuzzParseGemLine(f *testing.F) {
	f.Add("rails (6.0.0 < 7.0.0)")
	f.Add("()")
	f.Add("no parens here")
	f.Add("weird (< )")
	f.Fuzz(func(t *testing.T, line string) {
		parseGemLine(line) // must not panic; ok flag handles the rest
	})
}

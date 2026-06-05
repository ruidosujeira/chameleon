package adapters

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runCapture runs name+args and returns its stdout. An ExitError is TOLERATED
// when stdout is non-empty — several tools (npm outdated being the canonical
// case) exit non-zero precisely BECAUSE they found something to report, yet
// still write the machine output we want. A failure to start the command, or an
// exit error with empty stdout, surfaces as an error (carrying stderr when the
// tool wrote a diagnostic there). `label` is the human prefix on errors.
func runCapture(label, name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return nil, fmt.Errorf("%s: %w", label, err) // couldn't start the tool
		}
		if len(bytes.TrimSpace(out)) == 0 {
			if msg := strings.TrimSpace(string(ee.Stderr)); msg != "" {
				return nil, fmt.Errorf("%s: %s", label, msg)
			}
			return nil, fmt.Errorf("%s: %w", label, err)
		}
		// Non-empty stdout with a non-zero exit: useful output — proceed.
	}
	return out, nil
}

// hasFlag reports whether any of flags appears as a standalone token in argv.
func hasFlag(argv []string, flags ...string) bool {
	for _, a := range argv {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

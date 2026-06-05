package adapters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// DockerPS re-renders `docker ps`. A state-list sibling: each container's State
// (running/exited/paused/…) drives the glyph and color from the shared theme.
type DockerPS struct{}

// Handles matches `docker ps`.
func (DockerPS) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "docker" && argv[1] == "ps"
}

// dockerLine mirrors one line of `docker ps --format '{{json .}}'`, which emits
// newline-delimited JSON objects (one per container) rather than an array.
type dockerLine struct {
	Names  string `json:"Names"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Image  string `json:"Image"`
}

// Render captures the container list as NDJSON and rebuilds the output. Extra
// args (e.g. -a to include stopped containers) are forwarded to docker.
func (DockerPS) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	args := append([]string{"ps"}, argv[2:]...)
	args = append(args, "--format", "{{json .}}")
	data, err := runCapture("docker ps", "docker", args...)
	if err != nil {
		return "", err
	}
	return renderDockerPS(data, t, r)
}

// renderDockerPS is the pure transform — table-tested offline. The detail column
// is docker's own human Status string ("Up 2 hours", "Exited (0) 3m ago").
func renderDockerPS(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	var items []stateItem
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // container rows can be long
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var d dockerLine
		if err := json.Unmarshal(line, &d); err != nil {
			return "", fmt.Errorf("parse docker ps json: %w", err)
		}
		items = append(items, stateItem{state: d.State, name: d.Names, detail: d.Status})
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("scan docker ps: %w", err)
	}
	return renderStates("docker ps", "nenhum container", items, t, r), nil
}

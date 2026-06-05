package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// GhRun re-renders `gh run list` — recent GitHub Actions runs. A state-list
// sibling: each run's conclusion (success/failure/…) or in-flight status drives
// the glyph and color from the shared theme.
type GhRun struct{}

// Handles matches `gh run list`.
func (GhRun) Handles(argv []string) bool {
	return len(argv) >= 2 && argv[0] == "gh" && argv[1] == "run" &&
		(len(argv) == 2 || argv[2] == "list")
}

// ghRun mirrors one element of `gh run list --json …`.
type ghRun struct {
	Status       string `json:"status"`       // queued | in_progress | completed
	Conclusion   string `json:"conclusion"`   // success | failure | cancelled | … (when completed)
	WorkflowName string `json:"workflowName"` // the workflow's name
	DisplayTitle string `json:"displayTitle"` // the run's title (commit subject)
	HeadBranch   string `json:"headBranch"`
}

// Render captures recent runs as JSON and rebuilds the output.
func (GhRun) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	data, err := runCapture("gh run list", "gh", "run", "list",
		"--json", "status,conclusion,workflowName,displayTitle,headBranch")
	if err != nil {
		return "", err
	}
	return renderGhRun(data, t, r)
}

// renderGhRun is the pure transform — table-tested offline. A finished run's
// state is its conclusion; an unfinished one's is its status.
func renderGhRun(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		trimmed = "[]"
	}
	var raw []ghRun
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("parse gh run list json: %w", err)
	}

	items := make([]stateItem, 0, len(raw))
	for _, run := range raw {
		state := run.Conclusion
		if state == "" {
			state = run.Status // still queued / in_progress
		}
		name := run.WorkflowName
		if name == "" {
			name = run.DisplayTitle
		}
		items = append(items, stateItem{state: state, name: name, detail: run.HeadBranch})
	}
	return renderStates("gh run list", "nenhuma execução", items, t, r), nil
}

package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// KubectlPods re-renders `kubectl get pods`. A state-list sibling: each pod's
// phase (or a container's waiting reason like CrashLoopBackOff) drives the glyph
// and color from the shared theme.
type KubectlPods struct{}

// Handles matches `kubectl get pods` (and the `pod`/`po` aliases).
func (KubectlPods) Handles(argv []string) bool {
	if len(argv) < 3 || argv[0] != "kubectl" || argv[1] != "get" {
		return false
	}
	switch argv[2] {
	case "pods", "pod", "po":
		return true
	}
	return false
}

// kubePods mirrors the slice of `kubectl get pods -o json` we draw from.
type kubePods struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Phase             string `json:"phase"`
			ContainerStatuses []struct {
				Ready        bool `json:"ready"`
				RestartCount int  `json:"restartCount"`
				State        map[string]struct {
					Reason string `json:"reason"`
				} `json:"state"`
			} `json:"containerStatuses"`
		} `json:"status"`
	} `json:"items"`
}

// Render captures the pod list as JSON and rebuilds the output. Extra args (a
// namespace, a selector) are forwarded to kubectl.
func (KubectlPods) Render(argv []string, t *theme.Theme, r *style.Renderer) (string, error) {
	args := append(append([]string{"get"}, argv[2:]...), "-o", "json")
	data, err := runCapture("kubectl get pods", "kubectl", args...)
	if err != nil {
		return "", err
	}
	return renderKubectlPods(data, t, r)
}

// renderKubectlPods is the pure transform — table-tested offline. A pod's state
// is its container's waiting reason when one exists (so CrashLoopBackOff,
// ImagePullBackOff, … surface), otherwise its phase.
func renderKubectlPods(data []byte, t *theme.Theme, r *style.Renderer) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		trimmed = "{}"
	}
	var raw kubePods
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("parse kubectl json: %w", err)
	}

	items := make([]stateItem, 0, len(raw.Items))
	for _, p := range raw.Items {
		state := p.Status.Phase
		ready, total, restarts := 0, len(p.Status.ContainerStatuses), 0
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCount
			if w, ok := cs.State["waiting"]; ok && w.Reason != "" {
				state = w.Reason // e.g. CrashLoopBackOff, ImagePullBackOff
			}
		}
		detail := fmt.Sprintf("%d/%d ready", ready, total)
		if restarts > 0 {
			detail += fmt.Sprintf("  ⟳%d", restarts)
		}
		items = append(items, stateItem{state: state, name: p.Metadata.Name, detail: detail})
	}
	return renderStates("kubectl get pods", "nenhum pod", items, t, r), nil
}

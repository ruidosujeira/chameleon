package adapters

import (
	"testing"

	"github.com/ruidosujeira/chameleon/style"
	"github.com/ruidosujeira/chameleon/theme"
)

// stateTestTheme carries the severity-bucket glyphs the state-list renderer
// falls back to (ok/warn/bad/unknown), plus prompt/clean. Renderer is disabled
// in tests, so colors don't affect the output text.
func stateTestTheme() *theme.Theme {
	t := &theme.Theme{
		Colors: map[string]style.Color{},
		Glyphs: map[string]string{
			"prompt": "❯", "clean": "✓",
			"ok": "●", "warn": "◐", "bad": "✗", "unknown": "○",
		},
	}
	t.Layout.Indent = "  "
	t.Layout.LabelWidth = 10
	return t
}

func TestStateListFamily(t *testing.T) {
	r := &style.Renderer{Enabled: false}

	cases := []struct {
		name   string
		render renderFn
		in     string
		want   string
	}{
		{
			name:   "kubectl worst-first with crashloop",
			render: renderKubectlPods,
			in: `{"items":[
				{"metadata":{"name":"web-7d9f"},"status":{"phase":"Running","containerStatuses":[{"ready":true,"restartCount":0,"state":{"running":{}}}]}},
				{"metadata":{"name":"job-x12"},"status":{"phase":"Running","containerStatuses":[{"ready":false,"restartCount":5,"state":{"waiting":{"reason":"CrashLoopBackOff"}}}]}}
			]}`,
			want: "❯ kubectl get pods\n" +
				"  ✗ CrashLoopBackOff job-x12   0/1 ready  ⟳5\n" +
				"  ● Running          web-7d9f  1/1 ready\n",
		},
		{
			name:   "kubectl no pods",
			render: renderKubectlPods,
			in:     `{"items":[]}`,
			want:   "❯ kubectl get pods\n  ✓ nenhum pod\n",
		},
		{
			name:   "docker exited floats above running",
			render: renderDockerPS,
			in: `{"Names":"db","State":"running","Status":"Up 3 days","Image":"postgres"}
{"Names":"cache","State":"exited","Status":"Exited (0) 1h ago","Image":"redis"}`,
			want: "❯ docker ps\n" +
				"  ✗ exited     cache  Exited (0) 1h ago\n" +
				"  ● running    db     Up 3 days\n",
		},
		{
			name:   "gh runs ordered bad→warn→ok",
			render: renderGhRun,
			in: `[
				{"status":"completed","conclusion":"success","workflowName":"CI","headBranch":"main"},
				{"status":"completed","conclusion":"failure","workflowName":"Deploy","headBranch":"main"},
				{"status":"in_progress","conclusion":"","workflowName":"Nightly","headBranch":"dev"}
			]`,
			want: "❯ gh run list\n" +
				"  ✗ failure     Deploy   main\n" +
				"  ◐ in_progress Nightly  dev\n" +
				"  ● success     CI       main\n",
		},
		{
			name:   "systemctl failed above active",
			render: renderSystemctlUnits,
			in: `[
				{"unit":"nginx.service","active":"active","sub":"running","description":"A high performance web server"},
				{"unit":"mysql.service","active":"failed","sub":"failed","description":"MySQL database"}
			]`,
			want: "❯ systemctl list-units\n" +
				"  ✗ failed     mysql.service  MySQL database\n" +
				"  ● active     nginx.service  A high performance web server\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.render([]byte(c.in), stateTestTheme(), r)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got != c.want {
				t.Errorf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.want)
			}
		})
	}
}

package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Runner func(ctx context.Context, argv []string) (string, error)

type Agent struct {
	Agent   string `json:"agent"`
	Status  string `json:"agent_status"`
	Cwd     string `json:"cwd"`
	FgCwd   string `json:"foreground_cwd"`
	PaneID  string `json:"pane_id"`
	TabID   string `json:"tab_id"`
	WSID    string `json:"workspace_id"`
	Title   string `json:"terminal_title"`
	Focused bool   `json:"focused"`
}

type listResp struct {
	Result struct {
		Agents []Agent `json:"agents"`
	} `json:"result"`
}

func List(ctx context.Context, run Runner) ([]Agent, error) {
	if run == nil {
		return nil, fmt.Errorf("no runner")
	}
	out, err := run(ctx, []string{"herdr", "agent", "list"})
	if err != nil {
		return nil, err
	}
	var r listResp
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return nil, err
	}
	return r.Result.Agents, nil
}

func ForProject(agents []Agent, projectPath string) []Agent {
	out := []Agent{}
	for _, a := range agents {
		for _, p := range []string{a.Cwd, a.FgCwd} {
			if p == projectPath || strings.HasPrefix(p, projectPath+"/") {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

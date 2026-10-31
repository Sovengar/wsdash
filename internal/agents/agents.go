package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Runner func(ctx context.Context, argv []string) (string, error)

type SessionRef struct {
	Agent  string `json:"agent"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Value  string `json:"value"`
}

type Agent struct {
	Agent         string     `json:"agent"`
	Status        string     `json:"agent_status"`
	Cwd           string     `json:"cwd"`
	FgCwd         string     `json:"foreground_cwd"`
	PaneID        string     `json:"pane_id"`
	TabID         string     `json:"tab_id"`
	WSID          string     `json:"workspace_id"`
	Title         string     `json:"terminal_title"`
	TitleStripped string     `json:"terminal_title_stripped"`
	AgentSession  SessionRef `json:"agent_session"`
	Focused       bool       `json:"focused"`
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

// SessionName resolves the human name of an agent's pi session. It reads at
// most the last 512 KiB of the session file, so huge sessions stay cheap.
// Callers should keep it out of the render path (the TUI caches it per path).
func SessionName(a Agent) string {
	if name := sessionInfoName(a.AgentSession.Value); name != "" {
		return name
	}
	if name := stripTerminalTitle(a.TitleStripped); name != "" {
		return name
	}
	return uuidPrefix(a.AgentSession.Value)
}

const sessionTailBytes = 512 << 10

func sessionInfoName(path string) string {
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return ""
	}
	start := st.Size() - sessionTailBytes
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return ""
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	if start > 0 {
		// The first line is probably cut in half; drop it.
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			return ""
		}
		data = data[i+1:]
	}

	name := ""
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var rec struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if json.Unmarshal(line, &rec) == nil && rec.Type == "session_info" {
			name = rec.Name
		}
	}
	return name
}

func stripTerminalTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.TrimPrefix(title, "π - ")
	if u := os.Getenv("USER"); u != "" {
		title = strings.TrimSuffix(title, " - "+u)
	}
	return strings.TrimSpace(title)
}

func uuidPrefix(value string) string {
	if value == "" {
		return ""
	}
	base := filepath.Base(value)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if i := strings.LastIndexByte(base, '_'); i >= 0 {
		base = base[i+1:]
	}
	if len(base) < 8 {
		return ""
	}
	candidate := base[:8]
	for _, r := range candidate {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return ""
		}
	}
	return candidate
}

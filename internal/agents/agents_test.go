package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sessionFile(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "2026-01-01T00-00-00-000Z_01a1926a-574a-7189-bec4-0ed4a17ea8e0.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func info(name string) string {
	return `{"type":"session_info","id":"x","parentId":null,"timestamp":"t","name":"` + name + `"}`
}

func TestSessionNameLastSessionInfoWins(t *testing.T) {
	path := sessionFile(t,
		info("first"),
		`{"type":"message","text":"ignored"}`,
		info("last"),
	)
	if got := SessionName(Agent{AgentSession: SessionRef{Value: path}}); got != "last" {
		t.Errorf("got %q, want last", got)
	}
}

func TestSessionNameMissingFileFallsBackToUUID(t *testing.T) {
	value := "/nonexistent/2026-01-01T00-00-00-000Z_01a1926a-574a-7189-bec4-0ed4a17ea8e0.jsonl"
	if got := SessionName(Agent{AgentSession: SessionRef{Value: value}}); got != "01a1926a" {
		t.Errorf("got %q, want 01a1926a", got)
	}
}

func TestSessionNameNoSessionInfoFallsBackToTitle(t *testing.T) {
	path := sessionFile(t, `{"type":"message","text":"no info here"}`)
	t.Setenv("USER", "buble")
	a := Agent{AgentSession: SessionRef{Value: path}, TitleStripped: "π - Fix the dashboard - buble"}
	if got := SessionName(a); got != "Fix the dashboard" {
		t.Errorf("got %q, want Fix the dashboard", got)
	}
}

func TestSessionNameReadsOnlyTailOfGiantFile(t *testing.T) {
	path := sessionFile(t,
		info("old name"),
		`{"type":"message","text":"`+strings.Repeat("x", 600<<10)+`"}`,
		info("new name"),
	)
	if got := SessionName(Agent{AgentSession: SessionRef{Value: path}}); got != "new name" {
		t.Errorf("got %q, want new name", got)
	}
}

func TestSessionNameTerminalTitleFallback(t *testing.T) {
	t.Setenv("USER", "buble")
	a := Agent{
		AgentSession:  SessionRef{Value: "/nonexistent/2026-01-01T00-00-00-000Z_01a1926a-574a-7189-bec4-0ed4a17ea8e0.jsonl"},
		TitleStripped: "π - Add tabs to sidebar rail - buble",
	}
	if got := SessionName(a); got != "Add tabs to sidebar rail" {
		t.Errorf("got %q, want Add tabs to sidebar rail", got)
	}
}

func TestSessionNameUUIDFallback(t *testing.T) {
	a := Agent{AgentSession: SessionRef{Value: "/tmp/2026-01-01T00-00-00-000Z_deadbeef-1111-2222-3333-444444444444.jsonl"}}
	if got := SessionName(a); got != "deadbeef" {
		t.Errorf("got %q, want deadbeef", got)
	}
	if got := SessionName(Agent{}); got != "" {
		t.Errorf("empty agent got %q, want empty", got)
	}
}

func TestListParsesAgentSession(t *testing.T) {
	raw := `{"result":{"agents":[{"agent":"pi","agent_session":{"value":"/tmp/s.jsonl"},"terminal_title_stripped":"π - x - u"}]}}`
	var r listResp
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.Result.Agents[0].AgentSession.Value != "/tmp/s.jsonl" {
		t.Errorf("session not parsed: %+v", r.Result.Agents[0])
	}
}

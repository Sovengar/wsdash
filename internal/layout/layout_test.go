package layout

import (
	"errors"
	"testing"
)

func TestPaneValidate(t *testing.T) {
	ok := []Pane{{}, {Split: "right"}, {Split: "down", Ratio: 0.5}, {Ratio: 1}, {Ratio: 0}}
	for _, p := range ok {
		if err := p.Validate(); err != nil {
			t.Errorf("%+v should be valid: %v", p, err)
		}
	}
	if err := (Pane{Split: "up"}).Validate(); !errors.Is(err, ErrBadSplit) {
		t.Errorf("got %v, want ErrBadSplit", err)
	}
	for _, r := range []float64{-0.1, 1.1} {
		if err := (Pane{Ratio: r}).Validate(); !errors.Is(err, ErrBadRatio) {
			t.Errorf("ratio %v: got %v, want ErrBadRatio", r, err)
		}
	}
}

func TestWorkspaceValidateWalksAllPanes(t *testing.T) {
	ws := Workspace{Tabs: []Tab{
		{Panes: []Pane{{Split: "right"}}},
		{Panes: []Pane{{}, {Split: "sideways"}}},
	}}
	if err := ws.Validate(); !errors.Is(err, ErrBadSplit) {
		t.Fatalf("got %v", err)
	}
	if err := (Workspace{}).Validate(); err != nil {
		t.Fatalf("empty workspace should be valid: %v", err)
	}
}

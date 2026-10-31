package layout

type Workspace struct {
	Name string
	Root string
	Env  map[string]string
	Tabs []Tab
}

type Tab struct {
	Label string
	Cwd   string
	Panes []Pane
}

type Pane struct {
	Command string
	Cwd     string
	Env     map[string]string
	Split   string
	Ratio   float64
	Focus   bool
	WaitFor string
}

func (p Pane) Validate() error {
	switch p.Split {
	case "", "right", "down":
	default:
		return ErrBadSplit
	}
	if p.Ratio < 0 || p.Ratio > 1 {
		return ErrBadRatio
	}
	return nil
}

func (w Workspace) Validate() error {
	for _, t := range w.Tabs {
		for _, p := range t.Panes {
			if err := p.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

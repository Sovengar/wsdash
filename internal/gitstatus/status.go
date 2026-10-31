package gitstatus

import (
	"context"
	"os/exec"
	"strings"

	"wsdash/internal/project"
)

func Status(ctx context.Context, path string) project.Status {
	st := project.Status{}
	// branch
	out, err := run(ctx, path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		st.Err = err
		return st
	}
	st.Branch = strings.TrimSpace(out)
	// dirty
	d, _ := run(ctx, path, "status", "--porcelain")
	st.Dirty = strings.TrimSpace(d) != ""
	// ahead/behind
	ab, _ := run(ctx, path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	fields := strings.Fields(strings.TrimSpace(ab))
	if len(fields) == 2 {
		st.AheadBehind = "↑" + fields[0] + " ↓" + fields[1]
	}
	// last commit
	lc, _ := run(ctx, path, "log", "-1", "--format=%h %s")
	st.LastCommit = strings.TrimSpace(lc)
	return st
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	b, err := cmd.Output()
	return string(b), err
}

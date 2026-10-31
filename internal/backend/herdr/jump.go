package herdr

import (
	"context"

	"wsdash/internal/agents"
)

func JumpAgent(ctx context.Context, run func(context.Context, []string) (string, error), a agents.Agent) error {
	if _, err := run(ctx, []string{"herdr", "workspace", "focus", a.WSID}); err != nil {
		return err
	}
	_, err := run(ctx, []string{"herdr", "tab", "focus", a.TabID})
	return err
}

package automation

import (
	"context"
	"fmt"

	automationactions "twitch-obs-bot/automation/actions"
)

type ActionRegistry struct {
	handlers map[string]automationactions.Handler
}

func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		handlers: make(map[string]automationactions.Handler),
	}
}

func (r *ActionRegistry) Register(name string, handler automationactions.Handler) {
	r.handlers[name] = handler
}

func (r *ActionRegistry) Execute(
	ctx context.Context,
	name string,
	args map[string]string,
) error {
	h, ok := r.handlers[name]
	if !ok {
		return fmt.Errorf("unknown action: %s", name)
	}

	return h.Execute(ctx, args)
}

package actions

import (
	"context"
	"fmt"
	"time"
)

type WaitAction struct{}

func (a *WaitAction) Execute(ctx context.Context, args map[string]string) error {
	dStr := args["duration"]
	if dStr == "" {
		return fmt.Errorf("missing duration")
	}

	d, err := time.ParseDuration(dStr)
	if err != nil {
		return err
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

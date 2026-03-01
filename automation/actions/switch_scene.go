package actions

import (
	"context"
	"fmt"
)

type SceneSwitcher interface {
	SwitchScene(scene string) error
}

type SwitchSceneAction struct {
	Obs SceneSwitcher
}

func (a *SwitchSceneAction) Execute(ctx context.Context, args map[string]string) error {
	scene := args["scene"]
	if scene == "" {
		return fmt.Errorf("missing scene")
	}

	if a.Obs == nil {
		return fmt.Errorf("obs unavailable")
	}

	return a.Obs.SwitchScene(scene)
}

package actions

import (
	"context"
	"fmt"

	obsservice "twitch-obs-bot/services/obs"

	"github.com/andreykaipov/goobs"
)

type SwitchSceneAction struct {
	Obs *goobs.Client
}

func (a *SwitchSceneAction) Execute(ctx context.Context, args map[string]string) error {
	scene := args["scene"]
	if scene == "" {
		return fmt.Errorf("missing scene")
	}

	return obsservice.SwitchScene(a.Obs, scene)
}

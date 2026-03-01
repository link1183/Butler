package obs

import (
	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/scenes"
)

func SwitchScene(obs *goobs.Client, scene string) error {
	_, err := obs.Scenes.SetCurrentProgramScene(&scenes.SetCurrentProgramSceneParams{
		SceneName: &scene,
	})

	return err
}

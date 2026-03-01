package actions

import (
	"context"
	"fmt"
)

type VarSetter interface {
	Set(key, value string)
}

type SetVarAction struct {
	Vars VarSetter
}

func (a *SetVarAction) Execute(ctx context.Context, args map[string]string) error {
	name := args["name"]
	value := args["value"]

	if name == "" {
		return fmt.Errorf("missing var name")
	}

	a.Vars.Set(name, value)
	return nil
}

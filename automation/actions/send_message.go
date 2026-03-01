package actions

import (
	"context"
	"fmt"
)

type ChatSender interface {
	Say(channel, message string)
}

type SendMessageAction struct {
	Client  ChatSender
	Channel string
}

func (a *SendMessageAction) Execute(ctx context.Context, args map[string]string) error {
	message := args["message"]
	if message == "" {
		return fmt.Errorf("missing message")
	}

	a.Client.Say(a.Channel, message)
	return nil
}

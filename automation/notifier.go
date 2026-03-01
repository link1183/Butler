package automation

import "fmt"

type Notifier interface {
	TemplateFailed(ruleIndex int, actionType string)
	ActionFailed(ruleIndex int, actionType string)
}

type ChatNotifier struct {
	Client  interface{ Say(channel, message string) }
	Channel string
}

func (n *ChatNotifier) TemplateFailed(ruleIndex int, actionType string) {
	if n == nil || n.Client == nil || n.Channel == "" {
		return
	}

	n.Client.Say(n.Channel, fmt.Sprintf(
		"Command failed while preparing %s for rule %d.",
		actionType,
		ruleIndex,
	))
}

func (n *ChatNotifier) ActionFailed(ruleIndex int, actionType string) {
	if n == nil || n.Client == nil || n.Channel == "" {
		return
	}

	n.Client.Say(n.Channel, fmt.Sprintf(
		"Command failed while running %s for rule %d.",
		actionType,
		ruleIndex,
	))
}

package matchers

import (
	"strings"

	"twitch-obs-bot/events"
)

type ContainsMatcher struct {
	EventType string
	Substring string
}

func (m *ContainsMatcher) Match(evt events.Event) (map[string]string, bool) {
	if evt.Type != m.EventType {
		return nil, false
	}

	msg, ok := evt.Data["message"].(string)
	if !ok {
		return nil, false
	}

	if strings.Contains(msg, m.Substring) {
		return map[string]string{}, true
	}

	return nil, false
}

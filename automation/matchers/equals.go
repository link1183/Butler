package matchers

import "twitch-obs-bot/events"

type EqualsMatcher struct {
	EventType string
	Value     string
}

func (m *EqualsMatcher) Match(evt events.Event) (map[string]string, bool) {
	if evt.Type != m.EventType {
		return nil, false
	}

	msg, ok := evt.Data["message"].(string)
	if !ok {
		return nil, false
	}

	if msg == m.Value {
		return map[string]string{}, true
	}

	return nil, false
}

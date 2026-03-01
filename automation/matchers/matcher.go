// Package matchers
package matchers

import "twitch-obs-bot/events"

type Matcher interface {
	Match(evt events.Event) (map[string]string, bool)
}

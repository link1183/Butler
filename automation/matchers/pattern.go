package matchers

import (
	"regexp"
	"strings"

	"twitch-obs-bot/events"
)

type PatternMatcher struct {
	EventType string
	regex     *regexp.Regexp
}

func NewPatternMatcher(eventType, pattern string) *PatternMatcher {
	reVar := regexp.MustCompile(`\{(\w+)\}`)
	regexPattern := pattern

	matches := reVar.FindAllStringSubmatch(pattern, -1)

	for _, match := range matches {
		name := match[1]
		regexPattern = strings.ReplaceAll(
			regexPattern,
			"{"+name+"}",
			"(?P<"+name+">.+)",
		)
	}

	regexPattern = "^" + regexPattern + "$"

	return &PatternMatcher{
		EventType: eventType,
		regex:     regexp.MustCompile(regexPattern),
	}
}

func (m *PatternMatcher) Match(evt events.Event) (map[string]string, bool) {
	if evt.Type != m.EventType {
		return nil, false
	}

	msg, ok := evt.Data["message"].(string)
	if !ok {
		return nil, false
	}

	res := m.regex.FindStringSubmatch(msg)
	if res == nil {
		return nil, false
	}

	vars := make(map[string]string)
	for i, name := range m.regex.SubexpNames() {
		if i != 0 && name != "" {
			vars[name] = res[i]
		}
	}

	return vars, true
}

package automation

import (
	"fmt"
	"regexp"
	"strings"
)

var templateRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func (e *Engine) interpolate(input string, localVars map[string]string) (string, error) {
	result := input

	matches := templateRe.FindAllStringSubmatch(input, -1)

	for _, match := range matches {

		full := match[0]    // {{...}}
		content := match[1] // inside

		parts := strings.SplitN(content, "|", 2)

		key := parts[0]
		var defaultVal string
		if len(parts) == 2 {
			defaultVal = parts[1]
		}

		var value string
		found := false

		// Global var
		if strings.HasPrefix(key, "vars.") {
			name := strings.TrimPrefix(key, "vars.")
			if v, ok := e.vars.Get(name); ok {
				value = v
				found = true
			}
		} else {
			// Local var
			if v, ok := localVars[key]; ok {
				value = v
				found = true
			}
		}

		if !found {
			if defaultVal != "" {
				value = defaultVal
			} else {
				return "", fmt.Errorf("unresolved template: %s", full)
			}
		}

		result = strings.ReplaceAll(result, full, value)
	}

	return result, nil
}

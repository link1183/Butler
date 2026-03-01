package automation

import (
	"encoding/json"
	"fmt"
	"os"
)

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	for i, rule := range cfg.Rules {
		if len(rule.Actions) == 0 {
			return Config{}, fmt.Errorf("rule[%d]: no actions defined", i)
		}

		t := rule.Trigger

		count := 0
		if t.Equals != "" {
			count++
		}
		if t.Pattern != "" {
			count++
		}
		if t.Contains != "" {
			count++
		}

		if count == 0 {
			return Config{}, fmt.Errorf(
				"rule[%d]: trigger must define one of equals, pattern, or contains",
				i,
			)
		}

		if count > 1 {
			return Config{}, fmt.Errorf(
				"rule[%d]: trigger must define only one match type",
				i,
			)
		}
	}

	return cfg, nil
}

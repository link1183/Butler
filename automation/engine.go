// Package automation
package automation

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	automationconditions "twitch-obs-bot/automation/conditions"
	automationmatchers "twitch-obs-bot/automation/matchers"
	"twitch-obs-bot/events"
)

type Trigger struct {
	Type string `json:"type"`

	Equals   string `json:"equals,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Contains string `json:"contains,omitempty"`
}

type Action struct {
	Type string            `json:"type"`
	Args map[string]string `json:"args"`

	compiledArgs map[string]*Template
}

type ExecutionPolicy struct {
	Mode string `json:"mode"` // parallel | single | replace | ignore
}

type Rule struct {
	Trigger    Trigger          `json:"trigger"`
	Conditions []Condition      `json:"conditions,omitempty"`
	Actions    []Action         `json:"actions"`
	Execution  *ExecutionPolicy `json:"execution,omitempty"`

	matcher    automationmatchers.Matcher
	conditions []automationconditions.Evaluator
}

type Config struct {
	Rules []Rule `json:"rules"`
}

type Engine struct {
	rules     []Rule
	registry  *ActionRegistry
	vars      *VarStore
	execMgr   *ExecutionManager
	logger    *slog.Logger
	actionLog *slog.Logger
	notifier  Notifier
}

func NewEngine(
	cfg Config,
	registry *ActionRegistry,
	rootCtx context.Context,
	logger *slog.Logger,
) *Engine {
	if logger == nil {
		logger = slog.Default()
	}

	e := &Engine{
		rules:     cfg.Rules,
		registry:  registry,
		vars:      NewVarStore(),
		execMgr:   NewExecutionManager(rootCtx, logger.With("component", "exec")),
		logger:    logger.With("component", "engine"),
		actionLog: logger.With("component", "action"),
	}

	for i := range e.rules {

		t := e.rules[i].Trigger

		if t.Equals != "" {
			e.rules[i].matcher = &automationmatchers.EqualsMatcher{
				EventType: t.Type,
				Value:     t.Equals,
			}
		}

		if t.Pattern != "" {
			e.rules[i].matcher = automationmatchers.NewPatternMatcher(
				t.Type,
				t.Pattern,
			)
		}

		if t.Contains != "" {
			e.rules[i].matcher = &automationmatchers.ContainsMatcher{
				EventType: t.Type,
				Substring: t.Contains,
			}
		}

		for _, cond := range e.rules[i].Conditions {
			if cond.VarEquals != nil {
				e.rules[i].conditions = append(
					e.rules[i].conditions,
					&automationconditions.VarEqualsEvaluator{
						Name:  cond.VarEquals.Name,
						Value: cond.VarEquals.Value,
					},
				)
			}
			if cond.VarExists != nil {
				e.rules[i].conditions = append(
					e.rules[i].conditions,
					&automationconditions.VarExistsEvaluator{
						Name: cond.VarExists.Name,
					},
				)
			}
		}

		for ai := range e.rules[i].Actions {
			e.rules[i].Actions[ai].compiledArgs = make(map[string]*Template)

			for k, v := range e.rules[i].Actions[ai].Args {
				tpl, err := ParseTemplate(v)
				if err != nil {
					panic(fmt.Errorf("rule[%d] action[%d] arg %q template parse error: %w", i, ai, k, err))
				}
				e.rules[i].Actions[ai].compiledArgs[k] = tpl
			}
		}
	}

	return e
}

func (e *Engine) HandleEvent(evt events.Event) {
	for idx, rule := range e.rules {

		vars, matched := rule.matcher.Match(evt)
		if !matched {
			continue
		}
		vars = enrichVars(vars, evt)

		e.logger.Info("rule matched", slog.Int("rule", idx))

		// Evaluate conditions
		ok := true
		for _, cond := range rule.conditions {
			if !cond.Evaluate(vars, e.vars) {
				ok = false
				break
			}
		}
		if !ok {
			e.logger.Info("conditions failed", slog.Int("rule", idx))
			continue
		}

		mode := "parallel"
		if rule.Execution != nil && rule.Execution.Mode != "" {
			mode = rule.Execution.Mode
		}

		e.logger.Debug("starting rule execution",
			slog.Int("rule", idx),
			slog.String("mode", mode),
		)

		exec := e.execMgr.StartWithPolicy(idx, mode)
		if exec == nil {
			return
		}

		go func(ruleIndex int, r Rule, vars map[string]string, exec *Execution) {
			defer e.execMgr.Finish(ruleIndex, exec.ID)

			ctx := exec.Ctx

			for _, action := range r.Actions {

				select {
				case <-ctx.Done():
					e.execMgr.logger.Info("execution cancelled",
						slog.Int("rule", ruleIndex),
						slog.String("execution", exec.ID),
					)
					return
				default:
				}

				resolved := make(map[string]string)

				for k, tpl := range action.compiledArgs {
					val, err := tpl.Render(vars, e.vars)
					if err != nil {
						e.logger.Error("template render failed",
							slog.Int("rule", ruleIndex),
							slog.String("action", action.Type),
							slog.String("arg", k),
							slog.Any("error", err),
						)
						if e.notifier != nil {
							e.notifier.TemplateFailed(ruleIndex, action.Type)
						}
						return
					}
					resolved[k] = val
				}

				e.actionLog.Info("starting action",
					slog.Int("rule", ruleIndex),
					slog.String("execution", exec.ID),
					slog.String("type", action.Type),
					slog.Any("args", resolved),
				)

				err := e.registry.Execute(ctx, action.Type, resolved)
				if err != nil {
					e.actionLog.Error("action failed",
						slog.Int("rule", ruleIndex),
						slog.String("execution", exec.ID),
						slog.String("type", action.Type),
						slog.Any("error", err),
					)
					if e.notifier != nil {
						e.notifier.ActionFailed(ruleIndex, action.Type)
					}
					return
				}
			}
		}(idx, rule, vars, exec)

		return
	}
}

func (e *Engine) Vars() *VarStore {
	return e.vars
}

func (e *Engine) SetNotifier(notifier Notifier) {
	e.notifier = notifier
}

func (e *Engine) LogCapabilities() {
	e.logger.Info("automation capabilities",
		slog.Int("total_rules", len(e.rules)),
	)

	triggerTypes := make(map[string]bool)
	actionTypes := make(map[string]bool)
	conditionTypes := make(map[string]bool)

	for i, rule := range e.rules {

		triggerTypes[rule.Trigger.Type] = true

		// Trigger Info
		switch {
		case rule.Trigger.Pattern != "":
			e.logger.Info("configured rule",
				slog.Int("rule", i),
				slog.String("trigger", "pattern"),
				slog.String("event_type", rule.Trigger.Type),
				slog.String("pattern", rule.Trigger.Pattern),
				slog.Int("actions", len(rule.Actions)),
			)

		case rule.Trigger.Equals != "":
			e.logger.Info("configured rule",
				slog.Int("rule", i),
				slog.String("trigger", "equals"),
				slog.String("event_type", rule.Trigger.Type),
				slog.String("equals", rule.Trigger.Equals),
				slog.Int("actions", len(rule.Actions)),
			)

		case rule.Trigger.Contains != "":
			e.logger.Info("configured rule",
				slog.Int("rule", i),
				slog.String("trigger", "contains"),
				slog.String("event_type", rule.Trigger.Type),
				slog.String("contains", rule.Trigger.Contains),
				slog.Int("actions", len(rule.Actions)),
			)

		default:
			e.logger.Warn(
				"trigger has no match mode defined",
				slog.Int("rule", i),
			)
		}

		// Action Info
		for _, action := range rule.Actions {
			actionTypes[action.Type] = true
		}

		// Condition Info
		for _, cond := range rule.Conditions {
			if cond.VarEquals != nil {
				conditionTypes["var_equals"] = true
			}
			if cond.VarExists != nil {
				conditionTypes["var_exists"] = true
			}
		}
	}

	e.logger.Info("supported automation types",
		slog.Any("trigger_types", mapKeys(triggerTypes)),
		slog.Any("action_types", mapKeys(actionTypes)),
		slog.Any("condition_types", mapKeys(conditionTypes)),
	)
}

func mapKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func enrichVars(vars map[string]string, evt events.Event) map[string]string {
	if vars == nil {
		vars = make(map[string]string)
	}

	for key, value := range evt.Data {
		if _, exists := vars[key]; exists {
			continue
		}

		str, ok := value.(string)
		if !ok {
			continue
		}

		vars[key] = str
	}

	return vars
}

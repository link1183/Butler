// Package conditions
package conditions

type Store interface {
	Get(key string) (string, bool)
}

type Evaluator interface {
	Evaluate(vars map[string]string, store Store) bool
}

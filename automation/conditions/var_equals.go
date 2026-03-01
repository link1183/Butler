package conditions

type VarEqualsEvaluator struct {
	Name  string
	Value string
}

func (c *VarEqualsEvaluator) Evaluate(vars map[string]string, store Store) bool {
	val, ok := store.Get(c.Name)
	if !ok {
		return false
	}

	return val == c.Value
}

package conditions

type VarExistsEvaluator struct {
	Name string
}

func (c *VarExistsEvaluator) Evaluate(vars map[string]string, store Store) bool {
	_, ok := store.Get(c.Name)
	return ok
}

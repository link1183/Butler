package automation

type Condition struct {
	VarEquals *VarEqualsCondition `json:"var_equals,omitempty"`
	VarExists *VarExistsCondition `json:"var_exists,omitempty"`
}

type VarEqualsCondition struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type VarExistsCondition struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

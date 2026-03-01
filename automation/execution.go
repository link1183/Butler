package automation

import "context"

type Execution struct {
	ID     string
	Ctx    context.Context
	Cancel context.CancelFunc
}

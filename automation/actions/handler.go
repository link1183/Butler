// Package actions
package actions

import "context"

type Handler interface {
	Execute(ctx context.Context, args map[string]string) error
}

package testable

import "context"

type Testable interface {
	Name() string
	Test(context.Context, string) (string, error)
	Init(context.Context, int) error
	Cleanup()
}

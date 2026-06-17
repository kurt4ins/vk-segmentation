package service_test

import "context"

// fakeTx is a Transactor that runs fn inline, so service transaction logic can
// be unit-tested without a real database.
type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func intp(i int) *int { return &i }

package falta_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/a20r/falta"
	"github.com/stretchr/testify/assert"
)

// TestConcurrentUse exercises shared package-level factories from many goroutines at once. Combined with the -race
// flag in CI, this gates data races in factory reuse — the way factories are meant to be used in real programs.
func TestConcurrentUse(t *testing.T) {
	errQuery := falta.Newf("query %d failed")
	errOrder := falta.NewM("order {{.id}} rejected")
	errStore := falta.New[int]("store {{.}} offline")
	errShutdown := falta.NewError("shutting down")
	root := errors.New("root cause")

	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			q := errQuery.New(i).Wrap(root)
			assert.ErrorIs(t, q, errQuery)
			assert.ErrorIs(t, q, root)

			o := errOrder.New(falta.M{"id": i})
			assert.ErrorIs(t, o, errOrder)

			s := errStore.New(i)
			assert.ErrorIs(t, s, errStore)

			sh := errShutdown.Annotate("draining").Wrap(q)
			assert.ErrorIs(t, sh, errShutdown)
			assert.ErrorIs(t, sh, errQuery)
			assert.ErrorIs(t, sh, root)
		}(i)
	}

	wg.Wait()
}

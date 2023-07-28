package falta_test

import (
	"fmt"
	"testing"

	"github.com/a20r/falta"
	"github.com/a20r/mesa"
)

func TestFactory_fmtFactory(t *testing.T) {
	table := mesa.MethodMesa[falta.Factory[any], string, []any, error]{
		NewInstance: func(ctx *mesa.Ctx, errFmt string) falta.Factory[any] {
			return falta.Newf(errFmt)
		},

		Target: func(ctx *mesa.Ctx, inst falta.Factory[any], in []any) error {
			return inst.New(in...)
		},

		Cases: []mesa.MethodCase[falta.Factory[any], string, []any, error]{
			{
				Name:   "Return new error with params",
				Fields: "test error: the %s is %s",
				Input:  []any{"dog", "black"},

				Check: func(ctx *mesa.Ctx, inst falta.Factory[any], in []any, out error) {
					ctx.As.EqualError(out, fmt.Sprintf("test error: the %s is %s", in...))
				},
			},
			{
				Name:   "Return new error without params",
				Fields: "test error",
				Input:  []any{},

				Check: func(ctx *mesa.Ctx, inst falta.Factory[any], in []any, out error) {
					ctx.As.EqualError(out, "test error")
				},
			},
			{
				Name:   "Check if error Is the same",
				Fields: "test error: %s is %s",
				Input:  []any{"cat", "brown"},

				Check: func(ctx *mesa.Ctx, inst falta.Factory[any], in []any, out error) {
					err := inst.New("elon", "dumb")
					ctx.As.ErrorIs(out, err)
				},
			},
			{
				Name:   "Check wrapped error",
				Fields: "test error: %s is %s",
				Input:  []any{"cat", "brown"},

				Check: func(ctx *mesa.Ctx, inst falta.Factory[any], in []any, out error) {
					wrappedErr := fmt.Errorf("wrapped error")
					err := inst.New("elon", "dumb").Wrap(wrappedErr)
					ctx.As.ErrorIs(out, err)
					ctx.As.ErrorIs(err, wrappedErr)
				},
			},
		},
	}

	table.Run(t)
}

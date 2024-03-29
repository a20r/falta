package falta_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/a20r/falta"
	"github.com/a20r/mesa"
	"github.com/stretchr/testify/assert"
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
			{
				Name:   "Check annotation",
				Fields: "test error: %s is %s",
				Input:  []any{"cat", "brown"},

				Check: func(ctx *mesa.Ctx, inst falta.Factory[any], in []any, out error) {
					wrappedErr := fmt.Errorf("wrapped error")
					err := inst.New("elon", "dumb").Annotate("he really is").Wrap(wrappedErr)
					ctx.As.ErrorIs(out, err)
					ctx.As.ErrorIs(err, wrappedErr)
					ctx.As.Equal("test error: elon is dumb: he really is: "+wrappedErr.Error(), err.Error())
				},
			},
			{
				Name:   "Check if factory errors.Is the new error",
				Fields: "test error: %s is %s",
				Input:  []any{"cat", "brown"},

				Check: func(ctx *mesa.Ctx, inst falta.Factory[any], in []any, out error) {
					ctx.As.ErrorIs(out, inst)
					ctx.As.ErrorIs(inst, out)
					ctx.As.ErrorIs(inst, inst)
					ctx.As.ErrorIs(out, out)
				},
			},
		},
	}

	table.Run(t)
}

func TestCapture(t *testing.T) {
	as := assert.New(t)

	errCannotOpenFile := falta.Newf(`open: cannot open file %s`)

	open := func(name string) (file *os.File, err error) {
		defer errCannotOpenFile.New(name).Capture(&err)

		f, err := os.Open(name)

		if err != nil {
			return nil, err
		}

		return f, nil
	}

	_, err := open("does-not-exist.txt")
	t.Log(err)

	as.ErrorIs(err, errCannotOpenFile)
}

func TestNewError(t *testing.T) {
	err := falta.NewError("falta test: test error")
	wrappedErr := fmt.Errorf("wrapped error: %w", err)
	as := assert.New(t)
	as.ErrorIs(wrappedErr, err)
}

func TestNewM(t *testing.T) {
	f := falta.NewM("falta test: [code={{.code}}] test error with message '{{.message}}'")

	err := f.New(falta.M{
		"code":    503,
		"message": "Bad Gateway",
	})

	expectedErr := fmt.Errorf("falta test: [code=503] test error with message 'Bad Gateway'")
	assert.ErrorIs(t, err, expectedErr)
}

func TestExtendableFactory(t *testing.T) {
	ErrCallFailed := falta.NewM("falta test: [code={{.code}}] test error with message '{{.message}}'")
	ErrCallFailedWithReason := ErrCallFailed.Extend(falta.NewM("because {{.reason}}"))

	as := assert.New(t)

	{
		err := ErrCallFailed.New(falta.M{
			"code":    503,
			"message": "Bad Gateway",
		})

		expectedErr := fmt.Errorf("falta test: [code=503] test error with message 'Bad Gateway'")
		as.ErrorIs(err, expectedErr)
	}

	{
		err := ErrCallFailedWithReason.New(falta.M{
			"code":    503,
			"message": "Bad Gateway",
			"reason":  "server is down",
		})

		expectedErr := fmt.Errorf("falta test: [code=503] test error with message 'Bad Gateway' because server is down")
		as.ErrorIs(err, expectedErr)
	}
}

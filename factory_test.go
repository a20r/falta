package falta_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/a20r/falta"
	"github.com/stretchr/testify/assert"
)

func TestNewf(t *testing.T) {
	tests := []struct {
		name  string
		fmt   string
		input []any
		check func(t *testing.T, factory falta.Factory[any], input []any, out falta.Falta)
	}{
		{
			name:  "Return new error with params",
			fmt:   "test error: the %s is %s",
			input: []any{"dog", "black"},

			check: func(t *testing.T, _ falta.Factory[any], input []any, out falta.Falta) {
				assert.EqualError(t, out, fmt.Sprintf("test error: the %s is %s", input...))
			},
		},
		{
			name:  "Return new error without params",
			fmt:   "test error",
			input: []any{},

			check: func(t *testing.T, _ falta.Factory[any], _ []any, out falta.Falta) {
				assert.EqualError(t, out, "test error")
			},
		},
		{
			name:  "Check if error Is the same",
			fmt:   "test error: %s is %s",
			input: []any{"cat", "brown"},

			check: func(t *testing.T, factory falta.Factory[any], _ []any, out falta.Falta) {
				assert.ErrorIs(t, out, factory.New("elon", "dumb"))
			},
		},
		{
			name:  "Check wrapped error",
			fmt:   "test error: %s is %s",
			input: []any{"cat", "brown"},

			check: func(t *testing.T, factory falta.Factory[any], _ []any, out falta.Falta) {
				wrappedErr := errors.New("wrapped error")
				err := factory.New("elon", "dumb").Wrap(wrappedErr)
				assert.ErrorIs(t, out, err)
				assert.ErrorIs(t, err, wrappedErr)
			},
		},
		{
			name:  "Check annotation",
			fmt:   "test error: %s is %s",
			input: []any{"cat", "brown"},

			check: func(t *testing.T, factory falta.Factory[any], _ []any, out falta.Falta) {
				wrappedErr := errors.New("wrapped error")
				err := factory.New("elon", "dumb").Annotate("he really is").Wrap(wrappedErr)
				assert.ErrorIs(t, out, err)
				assert.ErrorIs(t, err, wrappedErr)
				assert.EqualError(t, err, "test error: elon is dumb: he really is: "+wrappedErr.Error())
			},
		},
		{
			name:  "Check if factory errors.Is the new error",
			fmt:   "test error: %s is %s",
			input: []any{"cat", "brown"},

			check: func(t *testing.T, factory falta.Factory[any], _ []any, out falta.Falta) {
				assert.ErrorIs(t, out, factory)
				assert.ErrorIs(t, factory, out)
				assert.ErrorIs(t, factory, factory)
				assert.ErrorIs(t, out, out)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := falta.Newf(test.fmt)
			test.check(t, factory, test.input, factory.New(test.input...))
		})
	}
}

func TestNew(t *testing.T) {
	type circle struct {
		Radius float64
	}

	tests := []struct {
		name     string
		fmt      string
		input    []circle
		expected string
	}{
		{
			name:     "Renders the template with the value provided",
			fmt:      "invalid circle: radius ({{.Radius}}) <= 0",
			input:    []circle{{Radius: -1}},
			expected: "invalid circle: radius (-1) <= 0",
		},
		{
			name:     "Falls back to the template source when no value is provided",
			fmt:      "invalid circle: radius ({{.Radius}}) <= 0",
			input:    nil,
			expected: "invalid circle: radius ({{.Radius}}) <= 0",
		},
		{
			name:     "Only uses the first value provided",
			fmt:      "invalid circle: radius ({{.Radius}}) <= 0",
			input:    []circle{{Radius: -1}, {Radius: -2}},
			expected: "invalid circle: radius (-1) <= 0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := falta.New[circle](test.fmt)
			err := factory.New(test.input...)
			assert.EqualError(t, err, test.expected)
			assert.ErrorIs(t, err, factory)
		})
	}
}

func TestCapture(t *testing.T) {
	errCannotOpenFile := falta.Newf(`open: cannot open file %s`)

	open := func(name string) (file *os.File, err error) {
		defer errCannotOpenFile.New(name).Capture(&err)

		f, err := os.Open(name)

		if err != nil {
			return nil, err
		}

		return f, nil
	}

	t.Run("Wraps the error returned by the function", func(t *testing.T) {
		_, err := open("does-not-exist.txt")
		assert.ErrorIs(t, err, errCannotOpenFile)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Leaves a nil error alone", func(t *testing.T) {
		f, err := open(os.DevNull)
		assert.NoError(t, err)
		assert.NoError(t, f.Close())
	})
}

func TestNewError(t *testing.T) {
	t.Run("Is found by errors.Is through a wrapping error", func(t *testing.T) {
		err := falta.NewError("falta test: test error")
		wrappedErr := fmt.Errorf("wrapped error: %w", err)
		assert.ErrorIs(t, wrappedErr, err)
	})

	t.Run("Panics when the message contains verbs", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = falta.NewError("falta test: test error: %s")
		})
	})
}

func TestFalta_Annotate(t *testing.T) {
	t.Run("Appends the annotation to the message", func(t *testing.T) {
		err := falta.NewError("falta test: test error").Annotate("because reasons")
		assert.EqualError(t, err, "falta test: test error: because reasons")
	})

	t.Run("Panics when the annotation contains verbs", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = falta.NewError("falta test: test error").Annotate("because %s")
		})
	})
}

func TestFalta_Unwrap(t *testing.T) {
	t.Run("Returns the wrapped error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped error")
		err := falta.NewError("falta test: test error").Wrap(wrappedErr)
		assert.Equal(t, wrappedErr, errors.Unwrap(err))
	})

	t.Run("Returns nil when nothing is wrapped", func(t *testing.T) {
		assert.NoError(t, errors.Unwrap(falta.NewError("falta test: test error")))
	})
}

// TestLiteralMessages locks in that messages are never run back through a printf interpreter at construction time:
// NewError and the template factories build their errors with errors.New, so '%' sequences that are not caught by the
// verb guard survive untouched. Only Newf formats, and only when arguments are passed.
func TestLiteralMessages(t *testing.T) {
	t.Run("NewError keeps non-verb percent sequences", func(t *testing.T) {
		assert.EqualError(t, falta.NewError("50%% done"), "50%% done")
		assert.EqualError(t, falta.NewError("50% done"), "50% done")
	})

	t.Run("Annotations stay literal", func(t *testing.T) {
		err := falta.NewError("db: down").Annotate("at 50%% capacity")
		assert.EqualError(t, err, "db: down: at 50%% capacity")
	})

	t.Run("Template factories keep percents from data", func(t *testing.T) {
		f := falta.NewM("falta test: value {{.v}}")
		assert.EqualError(t, f.New(falta.M{"v": "100%s"}), "falta test: value 100%s")
	})

	t.Run("Newf formats verbs when args are passed", func(t *testing.T) {
		assert.EqualError(t, falta.Newf("%d%% done").New(50), "50% done")
	})
}

func TestNewM(t *testing.T) {
	f := falta.NewM("falta test: [code={{.code}}] test error with message '{{.message}}'")

	err := f.New(falta.M{
		"code":    503,
		"message": "Bad Gateway",
	})

	assert.EqualError(t, err, "falta test: [code=503] test error with message 'Bad Gateway'")
	assert.ErrorIs(t, err, f)
}

func TestExtendableFactory(t *testing.T) {
	t.Run("Extends a template factory", func(t *testing.T) {
		errCallFailed := falta.NewM("falta test: [code={{.code}}] test error with message '{{.message}}'")
		errCallFailedWithReason := errCallFailed.Extend(falta.NewM("because {{.reason}}"))

		assert.EqualError(t, errCallFailed.New(falta.M{
			"code":    503,
			"message": "Bad Gateway",
		}), "falta test: [code=503] test error with message 'Bad Gateway'")

		assert.EqualError(t, errCallFailedWithReason.New(falta.M{
			"code":    503,
			"message": "Bad Gateway",
			"reason":  "server is down",
		}), "falta test: [code=503] test error with message 'Bad Gateway' because server is down")
	})

	t.Run("Extends a fmt factory", func(t *testing.T) {
		errCallFailed := falta.Newf("falta test: [code=%d] test error")
		errCallFailedWithReason := errCallFailed.Extend(falta.Newf("because %s"))

		assert.EqualError(t, errCallFailedWithReason.New(503, "server is down"),
			"falta test: [code=503] test error because server is down")
	})

	t.Run("Panics when factory kinds are mixed", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = falta.NewM("{{.a}}").Extend(mFactoryStub{})
		})

		assert.Panics(t, func() {
			_ = falta.Newf("%s").Extend(anyFactoryStub{})
		})
	})
}

// mFactoryStub is a falta.Factory[falta.M] that is not created by falta, so extending a template factory with it
// must panic.
type mFactoryStub struct{}

func (mFactoryStub) New(...falta.M) falta.Falta { return falta.NewError("stub") }
func (mFactoryStub) Error() string              { return "stub" }

// anyFactoryStub is a falta.Factory[any] that is not created by falta, so extending a fmt factory with it must panic.
type anyFactoryStub struct{}

func (anyFactoryStub) New(...any) falta.Falta { return falta.NewError("stub") }
func (anyFactoryStub) Error() string          { return "stub" }

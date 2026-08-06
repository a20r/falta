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
		name        string
		errFmt      string
		args        []any
		expectedErr string
	}{
		{
			name:        "with params",
			errFmt:      "test error: the %s is %s",
			args:        []any{"dog", "black"},
			expectedErr: "test error: the dog is black",
		},
		{
			name:        "without params",
			errFmt:      "test error",
			args:        nil,
			expectedErr: "test error",
		},
		{
			name:        "with numeric params",
			errFmt:      "test error: got %d, want %d",
			args:        []any{503, 200},
			expectedErr: "test error: got 503, want 200",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			as := assert.New(t)
			factory := falta.Newf(test.errFmt)
			err := factory.New(test.args...)

			as.EqualError(err, test.expectedErr)
			as.ErrorIs(err, factory)
		})
	}
}

func TestNewf_Is(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("test error: %s is %s")

	err := factory.New("cat", "brown")
	other := factory.New("elon", "dumb")

	as.ErrorIs(err, other, "errors from the same factory should match")
	as.ErrorIs(err, factory, "an error should match the factory that built it")
	as.ErrorIs(factory, err, "a factory should match the errors it builds")
	as.ErrorIs(factory, factory)
	as.ErrorIs(err, err)

	unrelated := falta.Newf("some other error: %s")
	as.NotErrorIs(err, unrelated.New("nope"), "errors with different declarations should not match")
}

func TestNewf_Wrap(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("test error: %s is %s")

	wrappedErr := errors.New("wrapped error")
	err := factory.New("elon", "dumb").Wrap(wrappedErr)

	as.EqualError(err, "test error: elon is dumb: wrapped error")
	as.ErrorIs(err, wrappedErr, "the wrapped error should be reachable through errors.Is")
	as.ErrorIs(err, factory, "wrapping should not break factory identity")
	as.Equal(wrappedErr, err.Unwrap())
}

func TestNewf_Annotate(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("test error: %s is %s")

	wrappedErr := errors.New("wrapped error")
	err := factory.New("elon", "dumb").Annotate("he really is").Wrap(wrappedErr)

	as.EqualError(err, "test error: elon is dumb: he really is: wrapped error")
	as.ErrorIs(err, wrappedErr)
	as.ErrorIs(err, factory)
}

func TestNewf_AnnotatePanicsOnVerbs(t *testing.T) {
	as := assert.New(t)
	err := falta.Newf("test error: %s").New("boom")

	as.Panics(func() {
		_ = err.Annotate("this annotation has a %s verb")
	})
}

func TestNew(t *testing.T) {
	type circle struct {
		Radius float64
	}

	tests := []struct {
		name        string
		errFmt      string
		args        []circle
		expectedErr string
	}{
		{
			name:        "with fields",
			errFmt:      "invalid circle: radius ({{.Radius}}) <= 0",
			args:        []circle{{Radius: -1}},
			expectedErr: "invalid circle: radius (-1) <= 0",
		},
		{
			name:        "without fields returns the raw template",
			errFmt:      "invalid circle: radius ({{.Radius}}) <= 0",
			args:        nil,
			expectedErr: "invalid circle: radius ({{.Radius}}) <= 0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			as := assert.New(t)
			factory := falta.New[circle](test.errFmt)
			err := factory.New(test.args...)

			as.EqualError(err, test.expectedErr)
			as.ErrorIs(err, factory)
		})
	}
}

func TestNew_PanicsOnInvalidTemplate(t *testing.T) {
	assert.Panics(t, func() {
		falta.New[struct{}]("invalid template: {{.Missing")
	})
}

func TestNewM(t *testing.T) {
	as := assert.New(t)
	factory := falta.NewM("falta test: [code={{.code}}] test error with message '{{.message}}'")

	err := factory.New(falta.M{
		"code":    503,
		"message": "Bad Gateway",
	})

	as.EqualError(err, "falta test: [code=503] test error with message 'Bad Gateway'")
	as.ErrorIs(err, factory)
}

// TestIsSemantics pins down exactly what errors.Is matches on. Falta compares the factory's
// declaration string and falls back to comparing rendered messages; it does not compare factory
// instances. The last two cases document that fallback — they are what the behavior *is*, not a
// contract worth relying on. Tightening Is to compare factory identity would flip them.
func TestIsSemantics(t *testing.T) {
	as := assert.New(t)

	a := falta.Newf("boom: %s")
	b := falta.Newf("boom: %s")  // a distinct factory with an identical declaration
	c := falta.Newf("other: %s") // a distinct factory with a different declaration

	t.Run("same factory, different data", func(t *testing.T) {
		as.ErrorIs(a.New("x"), a.New("y"))
	})

	t.Run("error matches its factory", func(t *testing.T) {
		as.ErrorIs(a.New("x"), a)
		as.ErrorIs(a, a.New("x"))
	})

	t.Run("different declaration does not match", func(t *testing.T) {
		as.NotErrorIs(a.New("x"), c.New("x"))
		as.NotErrorIs(a.New("x"), c)
	})

	t.Run("identical declarations are interchangeable", func(t *testing.T) {
		as.ErrorIs(a.New("x"), b.New("y"), "matching is by declaration string, not factory identity")
		as.ErrorIs(a.New("x"), b)
	})

	t.Run("equal rendered message matches", func(t *testing.T) {
		as.ErrorIs(a.New("x"), errors.New("boom: x"), "Is falls back to comparing messages")
		as.NotErrorIs(a.New("x"), errors.New("unrelated"))
	})
}

func TestNewError(t *testing.T) {
	as := assert.New(t)

	err := falta.NewError("falta test: test error")
	wrappedErr := fmt.Errorf("wrapped error: %w", err)

	as.EqualError(err, "falta test: test error")
	as.ErrorIs(wrappedErr, err)
}

func TestNewError_PanicsOnVerbs(t *testing.T) {
	assert.Panics(t, func() {
		falta.NewError("falta test: test error with a %s verb")
	})
}

func TestExtend(t *testing.T) {
	as := assert.New(t)

	errCallFailed := falta.NewM("falta test: [code={{.code}}] test error with message '{{.message}}'")
	errCallFailedWithReason := errCallFailed.Extend(falta.NewM("because {{.reason}}"))

	t.Run("base factory", func(t *testing.T) {
		err := errCallFailed.New(falta.M{
			"code":    503,
			"message": "Bad Gateway",
		})

		as.EqualError(err, "falta test: [code=503] test error with message 'Bad Gateway'")
	})

	t.Run("extended factory", func(t *testing.T) {
		err := errCallFailedWithReason.New(falta.M{
			"code":    503,
			"message": "Bad Gateway",
			"reason":  "server is down",
		})

		as.EqualError(err, "falta test: [code=503] test error with message 'Bad Gateway' because server is down")
		as.ErrorIs(err, errCallFailedWithReason)
	})

	t.Run("fmt factory", func(t *testing.T) {
		base := falta.Newf("test error: %s")
		extended := base.Extend(falta.Newf("because %s"))

		err := extended.New("it broke", "the disk is full")
		as.EqualError(err, "test error: it broke because the disk is full")
	})
}

// foreignFactory is a Factory implementation that does not come from falta itself, used to check that
// Extend rejects factories it does not know how to compose with.
type foreignFactory[T any] struct{}

func (foreignFactory[T]) Error() string          { return "foreign factory" }
func (foreignFactory[T]) New(_ ...T) falta.Falta { return falta.NewError("foreign factory") }

func TestExtend_PanicsOnMismatchedFactories(t *testing.T) {
	as := assert.New(t)

	as.Panics(func() {
		falta.NewM("{{.a}}").Extend(foreignFactory[falta.M]{})
	}, "tmpl factories can only be extended by other tmpl factories of the same type")

	as.Panics(func() {
		falta.Newf("%s").Extend(foreignFactory[any]{})
	}, "fmt factories can only be extended by other fmt factories")
}

func TestCapture(t *testing.T) {
	as := assert.New(t)

	errCannotOpenFile := falta.Newf("open: cannot open file %s")

	open := func(name string) (file *os.File, err error) {
		defer errCannotOpenFile.New(name).Capture(&err)

		f, err := os.Open(name) //nolint:gosec // the test controls the path

		if err != nil {
			return nil, err
		}

		return f, nil
	}

	t.Run("captures errors", func(t *testing.T) {
		_, err := open("does-not-exist.txt")

		as.Error(err)
		as.ErrorIs(err, errCannotOpenFile)
		as.ErrorIs(err, os.ErrNotExist, "the underlying error should stay inspectable")
		as.Contains(err.Error(), "open: cannot open file does-not-exist.txt")
	})

	t.Run("leaves nil errors alone", func(t *testing.T) {
		f, err := open(os.DevNull)

		as.NoError(err)
		as.NotNil(f)
		as.NoError(f.Close())
	})
}

func TestUnwrap(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("test error: %s")

	as.NoError(factory.New("boom").Unwrap(), "an unwrapped error has nothing underneath it")

	inner := errors.New("inner")
	as.Equal(inner, factory.New("boom").Wrap(inner).Unwrap())
}

func TestFactoryError(t *testing.T) {
	as := assert.New(t)

	as.EqualError(falta.Newf("test error: %s"), "test error: %s")
	as.EqualError(falta.NewM("test error: {{.reason}}"), "test error: {{.reason}}")
	as.EqualError(falta.New[struct{}]("test error"), "test error")
}

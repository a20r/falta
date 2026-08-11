package falta_test

import (
	"errors"
	"fmt"
	"os"
	"sync"
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

// Characterization: falta does not guard printf arity, so fmt's own error markers leak into
// the message. The README's "things that will bite you" section documents this.
func TestNewf_ArgCountMismatch(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("a %s b %s")

	as.EqualError(factory.New("x"), "a x b %!s(MISSING)")
	as.EqualError(factory.New("x", "y", "z"), "a x b y%!(EXTRA string=z)")
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

func TestNewf_WrapNilCause(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("boom: %s")

	err := factory.New("x").Wrap(nil)

	as.NoError(err.Unwrap(), "wrapping nil should not invent a cause")
	as.ErrorIs(err, factory, "wrapping nil should not break factory identity")
	as.Contains(err.Error(), "boom: x")
}

// Characterization: Wrap replaces the tracked cause rather than stacking. After a second
// Wrap, the first cause is still part of the message but is no longer reachable through
// errors.Is or Unwrap. Pinned so a change here is a decision, not an accident.
func TestNewf_WrapTwice(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("boom: %s")

	first := errors.New("first cause")
	second := errors.New("second cause")
	err := factory.New("x").Wrap(first).Wrap(second)

	as.EqualError(err, "boom: x: first cause: second cause")
	as.Equal(second, err.Unwrap())
	as.ErrorIs(err, second)
	as.NotErrorIs(err, first, "the first cause survives only in the message text")
	as.ErrorIs(err, factory)
}

func TestNewf_Annotate(t *testing.T) {
	as := assert.New(t)
	factory := falta.Newf("test error: %s is %s")

	wrappedErr := errors.New("wrapped error")
	err := factory.New("elon", "dumb").Annotate("he really is").Wrap(wrappedErr)

	as.EqualError(err, "test error: elon is dumb: he really is: wrapped error")
	as.ErrorIs(err, wrappedErr)
	as.ErrorIs(err, factory)

	as.NoError(factory.New("a", "b").Annotate("ctx").Unwrap(), "Annotate alone does not set a cause")
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

func TestNew_ExtraArgsIgnored(t *testing.T) {
	type point struct{ X int }

	factory := falta.New[point]("point: {{.X}}")

	assert.EqualError(t, factory.New(point{X: 1}, point{X: 2}), "point: 1",
		"only the first value should be used")
}

func TestNew_PanicsOnInvalidTemplate(t *testing.T) {
	assert.Panics(t, func() {
		falta.New[struct{}]("invalid template: {{.Missing")
	})
}

func TestNew_PanicsWhenTemplateCannotExecute(t *testing.T) {
	type point struct{ X int }

	factory := falta.New[point]("nope: {{.Missing}}")

	assert.Panics(t, func() {
		_ = factory.New(point{X: 1})
	}, "a template referencing a missing field parses at declaration but panics when New runs it")
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

// Characterization: text/template renders a missing map key as "<no value>" rather than
// failing, so a typo in a key produces a quietly wrong message instead of a panic.
func TestNewM_MissingKeyRendersNoValue(t *testing.T) {
	factory := falta.NewM("code={{.code}}")

	assert.EqualError(t, factory.New(falta.M{"status": 503}), "code=<no value>")
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

func TestNewError_WrapAndAnnotateKeepIdentity(t *testing.T) {
	as := assert.New(t)

	sentinel := falta.NewError("queue: already closed")
	cause := errors.New("connection reset")

	wrapped := sentinel.Wrap(cause)
	as.EqualError(wrapped, "queue: already closed: connection reset")
	as.ErrorIs(wrapped, sentinel)
	as.ErrorIs(wrapped, cause)
	as.Equal(cause, wrapped.Unwrap())

	annotated := sentinel.Annotate("during shutdown")
	as.EqualError(annotated, "queue: already closed: during shutdown")
	as.ErrorIs(annotated, sentinel)

	as.EqualError(sentinel, "queue: already closed", "Wrap and Annotate must not mutate the sentinel")
}

// The verb guard rejects fmt verbs (% followed by a word character), not every percent sign.
func TestVerbGuardAllowsBarePercent(t *testing.T) {
	as := assert.New(t)

	as.NotPanics(func() {
		as.EqualError(falta.NewError("disk 100% full"), "disk 100% full")
	})

	as.NotPanics(func() {
		err := falta.Newf("job %s failed").New("backup").Annotate("50% complete")
		as.EqualError(err, "job backup failed: 50% complete")
	})
}

func TestExtend(t *testing.T) {
	errCallFailed := falta.NewM("falta test: [code={{.code}}] test error with message '{{.message}}'")
	errCallFailedWithReason := errCallFailed.Extend(falta.NewM("because {{.reason}}"))

	t.Run("base factory", func(t *testing.T) {
		err := errCallFailed.New(falta.M{
			"code":    503,
			"message": "Bad Gateway",
		})

		assert.EqualError(t, err, "falta test: [code=503] test error with message 'Bad Gateway'")
	})

	t.Run("extended factory", func(t *testing.T) {
		as := assert.New(t)

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
		assert.EqualError(t, err, "test error: it broke because the disk is full")
	})
}

func TestExtend_Chain(t *testing.T) {
	base := falta.NewM("call failed")
	withCode := base.Extend(falta.NewM("[code={{.code}}]"))
	withReason := withCode.Extend(falta.NewM("because {{.reason}}"))

	err := withReason.New(falta.M{"code": 503, "reason": "the upstream is down"})

	assert.EqualError(t, err, "call failed [code=503] because the upstream is down")
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
	errCannotOpenFile := falta.Newf("open: cannot open file %s")

	open := func(name string) (file *os.File, err error) {
		defer errCannotOpenFile.New(name).Capture(&err)

		f, err := os.Open(name)

		if err != nil {
			return nil, err
		}

		return f, nil
	}

	t.Run("captures errors", func(t *testing.T) {
		as := assert.New(t)

		_, err := open("does-not-exist.txt")

		as.Error(err)
		as.ErrorIs(err, errCannotOpenFile)
		as.ErrorIs(err, os.ErrNotExist, "the underlying error should stay inspectable")
		as.Contains(err.Error(), "open: cannot open file does-not-exist.txt")
	})

	t.Run("leaves nil errors alone", func(t *testing.T) {
		as := assert.New(t)

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

func TestErrorsAs(t *testing.T) {
	as := assert.New(t)

	factory := falta.Newf("boom: %s")
	cause := errors.New("the cause")
	outer := fmt.Errorf("outer context: %w", factory.New("x").Wrap(cause))

	var f falta.Falta
	as.True(errors.As(outer, &f), "errors.As should find the Falta through a fmt.Errorf chain")
	as.EqualError(f, "boom: x: the cause")
	as.Equal(cause, f.Unwrap())

	as.ErrorIs(outer, factory, "factory identity should survive external wrapping")
	as.ErrorIs(outer, cause)

	var missing falta.Falta
	as.False(errors.As(errors.New("plain"), &missing))
}

// TestIsSemantics pins down exactly what errors.Is matches on. Falta compares the factory's
// declaration string and falls back to comparing rendered messages; it does not compare factory
// instances. The fallback cases document what the behavior *is*, not a contract worth relying
// on. Tightening Is to compare factory identity would flip them.
func TestIsSemantics(t *testing.T) {
	a := falta.Newf("boom: %s")
	b := falta.Newf("boom: %s")  // a distinct factory with an identical declaration
	c := falta.Newf("other: %s") // a distinct factory with a different declaration

	t.Run("same factory, different data", func(t *testing.T) {
		assert.ErrorIs(t, a.New("x"), a.New("y"))
	})

	t.Run("error matches its factory", func(t *testing.T) {
		as := assert.New(t)
		as.ErrorIs(a.New("x"), a)
		as.ErrorIs(a, a.New("x"))
	})

	t.Run("different declaration does not match", func(t *testing.T) {
		as := assert.New(t)
		as.NotErrorIs(a.New("x"), c.New("x"))
		as.NotErrorIs(a.New("x"), c)
	})

	t.Run("identical declarations are interchangeable", func(t *testing.T) {
		as := assert.New(t)
		as.ErrorIs(a.New("x"), b.New("y"), "matching is by declaration string, not factory identity")
		as.ErrorIs(a.New("x"), b)
	})

	t.Run("equal rendered message matches", func(t *testing.T) {
		as := assert.New(t)
		as.ErrorIs(a.New("x"), errors.New("boom: x"), "Is falls back to comparing messages")
		as.NotErrorIs(a.New("x"), errors.New("unrelated"))
	})

	t.Run("raw declaration string matches", func(t *testing.T) {
		assert.ErrorIs(t, a.New("x"), errors.New("boom: %s"),
			"a target whose message equals the declaration matches")
	})

	t.Run("plain error on the err side never matches", func(t *testing.T) {
		assert.NotErrorIs(t, errors.New("boom: x"), a.New("x"),
			"the matching logic lives on the falta side; a plain error under inspection has no Is method")
	})

	t.Run("template factories match from either side", func(t *testing.T) {
		as := assert.New(t)
		m := falta.NewM("code={{.code}}")
		err := m.New(falta.M{"code": 1})

		as.ErrorIs(err, m)
		as.ErrorIs(m, err)
		as.NotErrorIs(m, errors.New("unrelated"))
	})
}

func TestFactoryError(t *testing.T) {
	as := assert.New(t)

	as.EqualError(falta.Newf("test error: %s"), "test error: %s")
	as.EqualError(falta.NewM("test error: {{.reason}}"), "test error: {{.reason}}")
	as.EqualError(falta.New[struct{}]("test error"), "test error")
}

// Factories are declared once at package level and used from every goroutine that can fail,
// so shared use has to be safe. Run with -race (CI does) for this to mean anything.
func TestConcurrentUse(t *testing.T) {
	as := assert.New(t)

	tmplFactory := falta.NewM("worker {{.id}} failed")
	fmtFactory := falta.Newf("worker %d failed")
	cause := errors.New("cause")

	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			tmplErr := tmplFactory.New(falta.M{"id": id})
			fmtErr := fmtFactory.New(id).Annotate("during shutdown").Wrap(cause)

			as.ErrorIs(tmplErr, tmplFactory)
			as.ErrorIs(fmtErr, fmtFactory)
			as.ErrorIs(fmtErr, cause)
			as.NotErrorIs(tmplErr, fmtFactory)
		}(i)
	}

	wg.Wait()
}

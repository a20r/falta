package falta_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/a20r/falta"
	"github.com/stretchr/testify/assert"
)

// Compile-time pins: these fail the build if the public API stops satisfying its contracts.
var (
	_ error                            = falta.Falta{}
	_ error                            = falta.Newf("compile pin: %d")
	_ falta.Factory[int]               = falta.New[int]("pin {{.}}")
	_ falta.ExtendableFactory[any]     = falta.Newf("compile pin: %d")
	_ falta.ExtendableFactory[falta.M] = falta.NewM("compile pin")
	_ interface{ Unwrap() error }      = falta.Falta{}
	_ interface{ Is(error) bool }      = falta.Falta{}
)

// TestValueSemantics locks in that Falta values behave as values: Wrap, Annotate, and Extend return independent
// copies and never mutate the receiver, so package-level sentinels stay pristine no matter what call sites do.
func TestValueSemantics(t *testing.T) {
	t.Run("Wrap and Annotate return copies", func(t *testing.T) {
		base := falta.NewError("base error")
		cause := errors.New("cause")

		wrapped := base.Wrap(cause)
		annotated := base.Annotate("note")

		assert.EqualError(t, base, "base error")
		assert.NoError(t, errors.Unwrap(base))
		assert.EqualError(t, wrapped, "base error: cause")
		assert.Equal(t, cause, errors.Unwrap(wrapped))
		assert.EqualError(t, annotated, "base error: note")
	})

	t.Run("derived errors are independent of each other", func(t *testing.T) {
		base := falta.NewError("base error")

		first := base.Wrap(errors.New("first"))
		second := base.Wrap(errors.New("second"))

		assert.EqualError(t, first, "base error: first")
		assert.EqualError(t, second, "base error: second")
	})

	t.Run("Extend does not mutate the base factory", func(t *testing.T) {
		base := falta.Newf("base: %s")
		ext := base.Extend(falta.Newf("ext: %s"))

		assert.EqualError(t, base.New("a"), "base: a")
		assert.EqualError(t, ext.New("a", "b"), "base: a ext: b")
	})

	t.Run("Extend chains accumulate left to right", func(t *testing.T) {
		e := falta.Newf("a=%d").Extend(falta.Newf("b=%d")).Extend(falta.Newf("c=%d"))
		assert.EqualError(t, e.New(1, 2, 3), "a=1 b=2 c=3")

		m := falta.NewM("a={{.a}}").Extend(falta.NewM("b={{.b}}"))
		assert.EqualError(t, m.New(falta.M{"a": 1, "b": 2}), "a=1 b=2")
	})
}

// TestVerbGuard pins the panic guard: NewError and Annotate reject printf verbs at construction time so a
// misdirected "you meant Newf" bug fails loudly at init instead of shipping a literal %s to users.
func TestVerbGuard(t *testing.T) {
	verbs := []string{"%s", "%d", "%v", "%w", "%q", "%x", "%T", "%f"}

	for _, verb := range verbs {
		verb := verb

		t.Run("NewError panics on "+verb, func(t *testing.T) {
			assert.Panics(t, func() { _ = falta.NewError("oops " + verb) })
		})

		t.Run("Annotate panics on "+verb, func(t *testing.T) {
			assert.Panics(t, func() { _ = falta.NewError("base").Annotate("oops " + verb) })
		})
	}

	nonVerbs := []string{"", "%%", "100%", "% ", "%-", "%."}

	for _, s := range nonVerbs {
		s := s

		t.Run("NewError accepts "+fmt.Sprintf("%q", s), func(t *testing.T) {
			assert.NotPanics(t, func() {
				assert.Equal(t, s, falta.NewError(s).Error())
			})
		})
	}
}

// TestNewfPrintfSemantics pins that Newf is exactly fmt.Sprintf: verbs render, and arg-count mismatches produce
// fmt's standard %! diagnostics rather than being silently swallowed.
func TestNewfPrintfSemantics(t *testing.T) {
	assert.EqualError(t, falta.Newf("%s and %d").New("a", 7), "a and 7")
	assert.EqualError(t, falta.Newf("%q").New("a"), `"a"`)
	assert.EqualError(t, falta.Newf("%v").New([]int{1, 2}), "[1 2]")
	assert.EqualError(t, falta.Newf("%s %s").New("only"), "only %!s(MISSING)")
	assert.EqualError(t, falta.Newf("%s").New("a", "b"), "a%!(EXTRA string=b)")
}

func TestTemplateEdgeCases(t *testing.T) {
	t.Run("invalid template syntax panics at construction", func(t *testing.T) {
		assert.Panics(t, func() { _ = falta.New[struct{}]("{{") })
	})

	t.Run("missing struct field panics when the error is built", func(t *testing.T) {
		f := falta.New[struct{ A int }]("value {{.B}}")
		assert.Panics(t, func() { _ = f.New(struct{ A int }{A: 1}) })
	})

	t.Run("missing map key renders <no value>", func(t *testing.T) {
		assert.EqualError(t, falta.NewM("v={{.missing}}").New(falta.M{}), "v=<no value>")
	})

	t.Run("nested map data", func(t *testing.T) {
		f := falta.NewM("user {{.user.id}}: {{.user.name}}")
		err := f.New(falta.M{"user": falta.M{"id": 7, "name": "alex"}})
		assert.EqualError(t, err, "user 7: alex")
	})

	t.Run("scalar template value", func(t *testing.T) {
		f := falta.New[int]("code {{.}}")
		assert.EqualError(t, f.New(7), "code 7")
	})
}

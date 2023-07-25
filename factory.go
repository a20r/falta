package falta

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
)

// Factory is an error factory.
type Factory[T any] interface {
	New(vs ...T) Falta
}

// Falta is an error returned by the Factory
type Falta struct {
	errFmt string
	error
}

// Wrap wraps the error provided with the Falta instance.
func (f Falta) Wrap(err error) Falta {
	return Falta{errFmt: f.errFmt, error: fmt.Errorf(f.error.Error()+": %w", f.error)}
}

// Is returns true if the error provided is a Falta instance created by the same factory.
func (f Falta) Is(err error) bool {
	other := Falta{}
	return errors.As(err, &other) && other.errFmt == f.errFmt
}

// New creates a new Falta instance that construct errors by executing the provided template string on a struct
// of the type provided.
func New[T any](errFmt string) Factory[T] {
	return newTmplFalta[T](errFmt)
}

// Newf creates a new Falta instance that will construct errors using the printf format string provided.
func Newf(errFmt string) Factory[any] {
	return newFmtFactory(errFmt)
}

// Fields is a convenience type for using Falta instances with maps.
type Fields map[string]any

type tmplFalta[T any] struct {
	errFmt string
	tmpl   *template.Template
}

func newTmplFalta[T any](errFmt string) tmplFalta[T] {
	return tmplFalta[T]{
		errFmt: errFmt,
		tmpl:   template.Must(template.New("tmplFactoryFmt").Parse(errFmt)),
	}
}

// New constructs a new error by executing the Falta's template with the struct provided. It panics if the template
// returns an error with it executes.
func (f tmplFalta[T]) New(vs ...T) Falta {
	if len(vs) == 0 {
		return Falta{errFmt: f.errFmt, error: f}
	}

	builder := new(strings.Builder)

	if err := f.tmpl.Execute(builder, vs[0]); err != nil {
		panic(err)
	}

	return Falta{errFmt: f.errFmt, error: fmt.Errorf(builder.String())}
}

func (f tmplFalta[T]) Error() string {
	return f.errFmt
}

type fmtFalta struct {
	errFmt string
}

func newFmtFactory(errFmt string) fmtFalta {
	return fmtFalta{
		errFmt: errFmt,
	}
}

func (f fmtFalta) New(vs ...any) Falta {
	if len(vs) == 0 {
		return Falta{errFmt: f.errFmt, error: f}
	}

	return Falta{errFmt: f.errFmt, error: fmt.Errorf(f.errFmt, vs...)}
}

func (f fmtFalta) Error() string {
	return f.errFmt
}

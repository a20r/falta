package falta

import (
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
	error
}

func (f Falta) Wrap(err error) Falta {
	return Falta{fmt.Errorf(f.error.Error()+": %w", f.error)}
}

// New creates a new Falta instance that construct errors by executing the provided template string on a struct
// of the type provided.
func New[T any](errFmt string) Factory[T] {
	return newTmplFalta[T](errFmt)
}

// Newf creates a new Falta instance that will construct errors using the printf format string provided.
func Newf[T any](errFmt string) Factory[any] {
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
		return Falta{f}
	}

	builder := new(strings.Builder)

	if err := f.tmpl.Execute(builder, vs[0]); err != nil {
		panic(err)
	}

	return Falta{fmt.Errorf(builder.String())}
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
		return Falta{f}
	}

	return Falta{fmt.Errorf(f.errFmt, vs...)}
}

func (f fmtFalta) Error() string {
	return f.errFmt
}

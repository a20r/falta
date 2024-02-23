package falta

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
)

// Factory is an error factory.
type Factory[T any] interface {
	error
	New(vs ...T) Falta
}

// Falta is an error returned by the Factory
type Falta struct {
	errFmt     string
	wrappedErr error
	error
}

func NewError(msg string) Falta {
	return Falta{
		errFmt: msg,
		error:  fmt.Errorf(msg),
	}
}

// Wrap wraps the error provided with the Falta instance.
func (f Falta) Wrap(err error) Falta {
	f.error = fmt.Errorf("%s: %w", f.error.Error(), err)
	f.wrappedErr = err
	return f
}

// Annotate adds an annotation to the error to provide more context to why it's happening
func (f Falta) Annotate(annotation string) Falta {
	f.error = fmt.Errorf("%s: %s", f.error.Error(), annotation)
	return f
}

// Unwrap returns the wrapped error if there is one.
func (f Falta) Unwrap() error {
	return f.wrappedErr
}

// Is returns true if the error provided is a Falta instance created by the same factory.
func (f Falta) Is(err error) bool {
	if f.wrappedErr != nil && errors.Is(err, f.wrappedErr) {
		return true
	}

	other := Falta{}
	return errors.As(err, &other) && other.errFmt == f.errFmt || err.Error() == f.errFmt
}

// Capture captures the error provided and wraps it with the Falta instance if it's not nil. This should be called
// with defer at the top of the function for which you are trying to capture the error. This ensures that all errors
// returned from your function will be wrapped by the function passed into Capture. You should use a named return
// value for the error so that the error Capture wraps is the one returned from teh function.
func (f Falta) Capture(err *error) {
	if *err != nil {
		*err = f.Wrap(*err)
	}
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

func (f tmplFalta[T]) Is(err error) bool {
	other := Falta{}
	return errors.As(err, &other) && other.errFmt == f.errFmt || err.Error() == f.errFmt
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

func (f fmtFalta) Is(err error) bool {
	other := Falta{}
	return errors.As(err, &other) && other.errFmt == f.errFmt || err.Error() == f.errFmt
}

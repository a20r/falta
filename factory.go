package falta

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// Factory is an error factory.
type Factory[T any] interface {
	error
	New(vs ...T) Falta
}

// ExtendableFactory is an error factory that can be extended.
type ExtendableFactory[T any] interface {
	Factory[T]
	Extend(f Factory[T]) ExtendableFactory[T]
}

// Falta is an error returned by the Factory
type Falta struct {
	errFmt     string
	wrappedErr error
	error
}

// NewError returns a new Falta error type with the provided error string.
//
// NOTE (a20r, 2024-02-25): It panics if the provided message contains any fmt verbs.
func NewError(msg string) Falta {
	panicIfStringHasVerbs(msg)

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
//
// NOTE (a20r, 2024-02-25): It panics if the provided annotation contains any fmt verbs.
func (f Falta) Annotate(annotation string) Falta {
	panicIfStringHasVerbs(annotation)

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
	errAs := errors.As(err, &other) && other.errFmt == f.errFmt
	errFmtEq := err.Error() == f.errFmt
	errValueEq := err.Error() == f.Error()
	return errAs || errFmtEq || errValueEq
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

var verbsRegex *regexp.Regexp

func init() {
	const verbsRegexStr = `\%\w`
	re, err := regexp.Compile(verbsRegexStr)

	if err != nil {
		panic(fmt.Errorf("falta: cannot compile verbs regex: %w", err))
	}

	verbsRegex = re
}

func panicIfStringHasVerbs(msg string) {
	if verbsRegex.MatchString(msg) {
		panic(fmt.Errorf(`falta: string "%s" has verbs`, msg))
	}
}

// New creates a new Falta instance that construct errors by executing the provided template string on a struct
// of the type provided.
func New[T any](errFmt string) Factory[T] {
	return newTmplFalta[T](errFmt)
}

// M is a convenience type for using Falta instances with maps.
type M map[string]any

// NewM returns a new ExtendableFactory instance using a template that expects a falta.M (i.e., map[string]any).
// This is a convenience function for calling falta.New[falta.M](...)
func NewM(errFmt string) ExtendableFactory[M] {
	return newTmplFalta[M](errFmt)
}

// Newf creates a new Falta instance that will construct errors using the printf format string provided.
func Newf(errFmt string) ExtendableFactory[any] {
	return newFmtFactory(errFmt)
}

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
		panic(fmt.Errorf("falta: cannot execute template: %w", err))
	}

	return Falta{errFmt: f.errFmt, error: fmt.Errorf(builder.String())}
}

func (f tmplFalta[T]) Extend(other Factory[T]) ExtendableFactory[T] {
	v, ok := other.(tmplFalta[T])

	if !ok {
		panic(fmt.Errorf("falta: tmpl factories can only be extended by other tmpl factories with the same type"))
	}

	return newTmplFalta[T](f.errFmt + " " + v.errFmt)
}

func (f tmplFalta[T]) Error() string {
	return f.errFmt
}

func (f tmplFalta[T]) Is(err error) bool {
	other := Falta{}
	errAs := errors.As(err, &other) && other.errFmt == f.errFmt
	errFmtEq := err.Error() == f.errFmt
	errValueEq := err.Error() == f.Error()
	return errAs || errFmtEq || errValueEq
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

func (f fmtFalta) Extend(other Factory[any]) ExtendableFactory[any] {
	v, ok := other.(fmtFalta)

	if !ok {
		panic(fmt.Errorf("falta: fmt factories can only be extended by other fmt factories"))
	}

	return newFmtFactory(f.errFmt + " " + v.errFmt)
}

func (f fmtFalta) Error() string {
	return f.errFmt
}

func (f fmtFalta) Is(err error) bool {
	other := Falta{}
	errAs := errors.As(err, &other) && other.errFmt == f.errFmt
	errFmtEq := err.Error() == f.errFmt
	errValueEq := err.Error() == f.Error()
	return errAs || errFmtEq || errValueEq
}

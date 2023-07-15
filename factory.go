package falta

import (
	"fmt"
	"strings"
	"text/template"
)

type Fields map[string]any

type Factory[T any] interface {
	New(vs ...T) error
}

func New[T any](errFmt string) Factory[T] {
	return newTmplFactory[T](errFmt)
}

func Newf[T any](errFmt string) Factory[any] {
	return newFmtFactory(errFmt)
}

type tmplFactory[T any] struct {
	errFmt string
	tmpl   *template.Template
}

func newTmplFactory[T any](errFmt string) tmplFactory[T] {
	return tmplFactory[T]{
		errFmt: errFmt,
		tmpl:   template.Must(template.New("tmplFactoryFmt").Parse(errFmt)),
	}
}

func (f tmplFactory[T]) New(vs ...T) error {
	if len(vs) == 0 {
		return f
	}

	builder := new(strings.Builder)

	if err := f.tmpl.Execute(builder, vs[0]); err != nil {
		panic(err)
	}

	return fmt.Errorf(builder.String())
}

func (f tmplFactory[T]) Error() string {
	return f.errFmt
}

type fmtFactory struct {
	errFmt string
	parent error
}

func newFmtFactory(errFmt string) fmtFactory {
	return fmtFactory{
		errFmt: errFmt,
	}
}

func (f fmtFactory) New(vs ...any) error {
	if len(vs) == 0 {
		return f
	}

	if f.parent != nil {
		return fmt.Errorf(f.errFmt+": %w", append(vs, f.parent)...)
	}

	return fmt.Errorf(f.errFmt, vs...)
}

func (f fmtFactory) Error() string {
	return f.errFmt
}

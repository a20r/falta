package falta_test

import (
	"errors"
	"fmt"

	"github.com/a20r/falta"
)

type Circle struct {
	Radius float64
}

var ErrInvalidCircle = falta.New[Circle]("invalid circle: radius ({{.Radius}}) <= 0")

func IsCircleValid(circle Circle) error {
	if circle.Radius <= 0 {
		return ErrInvalidCircle.New(circle)
	}

	return nil
}

func ExampleFactory() {
	circle := Circle{Radius: -1}

	if err := IsCircleValid(circle); err != nil {
		fmt.Println(err)
	}

	// Output:
	// invalid circle: radius (-1) <= 0
}

func ExampleNewf() {
	errNotFound := falta.Newf("user not found: id=%d")

	err := errNotFound.New(42)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errNotFound))

	// Output:
	// user not found: id=42
	// true
}

func ExampleNewError() {
	errShutdown := falta.NewError("server: shutting down")

	err := fmt.Errorf("worker stopped: %w", errShutdown)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errShutdown))

	// Output:
	// worker stopped: server: shutting down
	// true
}

func ExampleNewM() {
	errCallFailed := falta.NewM("call failed: [code={{.code}}] {{.message}}")

	fmt.Println(errCallFailed.New(falta.M{
		"code":    503,
		"message": "Bad Gateway",
	}))

	// Output:
	// call failed: [code=503] Bad Gateway
}

func ExampleFalta_Wrap() {
	errCannotOpen := falta.Newf("cannot open config %s")
	cause := errors.New("permission denied")

	err := errCannotOpen.New("app.yaml").Wrap(cause)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errCannotOpen), errors.Is(err, cause))

	// Output:
	// cannot open config app.yaml: permission denied
	// true true
}

func ExampleFalta_Annotate() {
	errCannotOpen := falta.Newf("cannot open config %s")

	fmt.Println(errCannotOpen.New("app.yaml").Annotate("run `myapp init` first"))

	// Output:
	// cannot open config app.yaml: run `myapp init` first
}

func ExampleFalta_Capture() {
	errCannotLoad := falta.Newf("load: cannot load %s")

	load := func(name string) (err error) {
		defer errCannotLoad.New(name).Capture(&err)
		return errors.New("file corrupt")
	}

	err := load("app")
	fmt.Println(err)
	fmt.Println(errors.Is(err, errCannotLoad))

	// Output:
	// load: cannot load app: file corrupt
	// true
}

func ExampleExtendableFactory() {
	errCallFailed := falta.NewM("call failed: code={{.code}}")
	errCallFailedBecause := errCallFailed.Extend(falta.NewM("because {{.reason}}"))

	fmt.Println(errCallFailedBecause.New(falta.M{
		"code":   503,
		"reason": "downstream is unavailable",
	}))

	// Output:
	// call failed: code=503 because downstream is unavailable
}

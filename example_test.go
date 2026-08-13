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

// A factory declared once is matchable with errors.Is no matter what data the
// individual error carries.
func ExampleNewf() {
	errUserNotFound := falta.Newf("user store: no user with id %d")

	err := errUserNotFound.New(42)

	fmt.Println(err)
	fmt.Println(errors.Is(err, errUserNotFound))

	// Output:
	// user store: no user with id 42
	// true
}

// NewM builds errors from a map when the fields do not warrant their own struct.
func ExampleNewM() {
	errCallFailed := falta.NewM("call failed: [code={{.code}}] {{.message}}")

	err := errCallFailed.New(falta.M{
		"code":    503,
		"message": "Bad Gateway",
	})

	fmt.Println(err)
	fmt.Println(errors.Is(err, errCallFailed))

	// Output:
	// call failed: [code=503] Bad Gateway
	// true
}

// NewError declares a plain sentinel error with no parameters.
func ExampleNewError() {
	errClosed := falta.NewError("queue: already closed")

	err := fmt.Errorf("publish: %w", errClosed)

	fmt.Println(err)
	fmt.Println(errors.Is(err, errClosed))

	// Output:
	// publish: queue: already closed
	// true
}

// Wrap keeps the cause reachable through errors.Is and errors.As.
func ExampleFalta_Wrap() {
	errDecodeFailed := falta.Newf("config: cannot decode %s")
	cause := errors.New("unexpected end of JSON input")

	err := errDecodeFailed.New("config.json").Wrap(cause)

	fmt.Println(err)
	fmt.Println(errors.Is(err, errDecodeFailed), errors.Is(err, cause))

	// Output:
	// config: cannot decode config.json: unexpected end of JSON input
	// true true
}

// Annotate adds situational context without changing the error's identity.
func ExampleFalta_Annotate() {
	errRequestFailed := falta.Newf("http: request to %s failed")

	err := errRequestFailed.New("/v1/users").Annotate("retry budget exhausted")

	fmt.Println(err)
	fmt.Println(errors.Is(err, errRequestFailed))

	// Output:
	// http: request to /v1/users failed: retry budget exhausted
	// true
}

// Capture wraps every error a function returns from a single deferred line, so
// you do not have to repeat the same wrapping at every return site.
func ExampleFalta_Capture() {
	errLoadFailed := falta.Newf("store: cannot load record %d")

	load := func(id int) (record string, err error) {
		defer errLoadFailed.New(id).Capture(&err)

		return "", errors.New("connection refused")
	}

	_, err := load(7)

	fmt.Println(err)
	fmt.Println(errors.Is(err, errLoadFailed))

	// Output:
	// store: cannot load record 7: connection refused
	// true
}

// Extend composes a general factory with extra detail, keeping one message
// prefix in one place.
func ExampleExtendableFactory() {
	errCallFailed := falta.NewM("call failed: [code={{.code}}] {{.message}}")
	errCallFailedWithReason := errCallFailed.Extend(falta.NewM("because {{.reason}}"))

	err := errCallFailedWithReason.New(falta.M{
		"code":    503,
		"message": "Bad Gateway",
		"reason":  "the upstream is down",
	})

	fmt.Println(err)

	// Output:
	// call failed: [code=503] Bad Gateway because the upstream is down
}

# falta

[![CI](https://github.com/a20r/falta/actions/workflows/ci.yml/badge.svg)](https://github.com/a20r/falta/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/a20r/falta.svg)](https://pkg.go.dev/github.com/a20r/falta)
[![Go Report Card](https://goreportcard.com/badge/github.com/a20r/falta)](https://goreportcard.com/report/github.com/a20r/falta)

**Declare your errors once, instantiate them everywhere.**

Falta is a tiny error package for Go. It gives you *error factories*: you
declare the shape of an error at the package level — where it's reviewable, greppable, and
matchable — and fill in the details at the call site.

```go
var ErrUserNotFound = falta.Newf("user store: no user with id %d")

// ... 40 files later
return ErrUserNotFound.New(id)
```

## The problem

Go makes you pick one of two bad options.

**Option 1: `fmt.Errorf` at the call site.** You get great context, but the error's identity is
a string that exists in exactly one place. The same logical failure gets worded four different
ways across the codebase, nobody can `errors.Is` against it, and callers end up matching on
substrings.

```go
// in three different files, for the same failure
return fmt.Errorf("no user %d", id)
return fmt.Errorf("user %d not found in store", id)
return fmt.Errorf("failed to find user: id=%d", id)
```

**Option 2: sentinel errors.** They're matchable, but they're frozen strings. To attach the ID
you have to wrap, which means the *message* still gets rewritten at every call site — so you're
back to option 1 with an extra step.

```go
var ErrNotFound = errors.New("not found")

return fmt.Errorf("user %d: %w", id, ErrNotFound) // message still ad hoc
```

Falta collapses the two. A factory is a matchable identity **and** a message template. One
declaration gives you consistent wording everywhere and `errors.Is` for free.

## Install

```sh
go get github.com/a20r/falta
```

Requires Go 1.21 or newer. The package itself imports only the standard library — `testify`
is a test-only dependency.

## Quick start

```go
package main

import (
	"errors"
	"fmt"

	"github.com/a20r/falta"
)

// Declare your package's errors in one place.
var (
	ErrUserNotFound = falta.Newf("user store: no user with id %d")
	ErrStoreClosed  = falta.NewError("user store: already closed")
)

type Store struct {
	users  map[int]string
	closed bool
}

func (s *Store) Get(id int) (string, error) {
	if s.closed {
		return "", ErrStoreClosed
	}

	name, ok := s.users[id]

	if !ok {
		return "", ErrUserNotFound.New(id)
	}

	return name, nil
}

func main() {
	store := &Store{users: map[int]string{1: "alex"}}

	_, err := store.Get(42)

	fmt.Println(err)                            // user store: no user with id 42
	fmt.Println(errors.Is(err, ErrUserNotFound)) // true
}
```

The message reads the way it would if you'd written `fmt.Errorf` by hand, and the caller can
still branch on the error without string matching.

## Building errors

Falta has four constructors. Pick based on how the error's data is shaped.

### `falta.Newf` — printf-style

The workhorse. Takes a format string, returns a factory whose `New` takes the arguments.

```go
var ErrCannotOpenFile = falta.Newf("config: cannot open %s (mode %o)")

err := ErrCannotOpenFile.New("app.yaml", 0644)
// config: cannot open app.yaml (mode 644)
```

### `falta.New[T]` — templated over a struct

When the error is about a domain object, hand it the object. The format string is a
[`text/template`](https://pkg.go.dev/text/template), so fields are named at the declaration
instead of positional at the call site.

```go
type Circle struct {
	Radius float64
}

var ErrInvalidCircle = falta.New[Circle]("invalid circle: radius ({{.Radius}}) <= 0")

err := ErrInvalidCircle.New(Circle{Radius: -1})
// invalid circle: radius (-1) <= 0
```

Adding a field to the message later is a one-line change at the declaration — no call site
has to be touched to keep passing the right number of positional arguments.

### `falta.NewM` — templated over a map

Same idea, for when the fields don't justify a struct. `falta.M` is just `map[string]any`.

```go
var ErrCallFailed = falta.NewM("call failed: [code={{.code}}] {{.message}}")

err := ErrCallFailed.New(falta.M{
	"code":    503,
	"message": "Bad Gateway",
})
// call failed: [code=503] Bad Gateway
```

### `falta.NewError` — a plain sentinel

For errors with nothing to interpolate. It returns an error value directly, not a factory.

```go
var ErrStoreClosed = falta.NewError("user store: already closed")

return ErrStoreClosed
```

## Working with errors

### `Wrap` — attach a cause

Both the falta error and the underlying cause stay reachable through `errors.Is`.

```go
err := ErrCannotOpenFile.New("app.yaml").Wrap(osErr)
// config: cannot open app.yaml: open app.yaml: no such file or directory

errors.Is(err, ErrCannotOpenFile) // true
errors.Is(err, os.ErrNotExist)    // true
```

### `Annotate` — attach situational context

For the "why this time" that doesn't belong in the declaration. Annotating does not change the
error's identity.

```go
err := ErrRequestFailed.New("/v1/users").Annotate("retry budget exhausted")
// http: request to /v1/users failed: retry budget exhausted

errors.Is(err, ErrRequestFailed) // true
```

### `Capture` — wrap every return from one line

The pattern falta was really written for. One deferred line at the top of a function wraps
whatever error that function returns, from any return site, without touching them.

```go
func LoadRecord(id int) (record string, err error) {
	defer ErrLoadFailed.New(id).Capture(&err)

	conn, err := pool.Acquire()

	if err != nil {
		return "", err // becomes: store: cannot load record 7: <pool error>
	}

	defer conn.Release()

	return conn.QueryRecord(id) // and so does this one
}
```

Two requirements: the error return must be **named** (`err error`), and `Capture` must be
called with `defer`. It's a no-op when the function returns `nil`.

This is what keeps error messages consistent as a function grows. New early returns get the
same wrapping automatically, so nobody has to remember the house style for this function's
errors — it's stated once, at the top.

### `Extend` — compose a base factory with extra detail

`Newf` and `NewM` factories can be extended, which appends a second template to the first.
Use it when a general error sometimes has a more specific explanation.

```go
var (
	ErrCallFailed           = falta.NewM("call failed: [code={{.code}}] {{.message}}")
	ErrCallFailedWithReason = ErrCallFailed.Extend(falta.NewM("because {{.reason}}"))
)

err := ErrCallFailedWithReason.New(falta.M{
	"code":    503,
	"message": "Bad Gateway",
	"reason":  "the upstream is down",
})
// call failed: [code=503] Bad Gateway because the upstream is down
```

### Inspecting

Falta errors are ordinary Go errors. `errors.Is`, `errors.As`, and `errors.Unwrap` all work,
including through a `fmt.Errorf("...: %w", err)` chain.

```go
var f falta.Falta

if errors.As(err, &f) {
	cause := f.Unwrap() // the error passed to Wrap, or nil
}
```

Matching is by the factory's **declaration string**, not by the data interpolated into it. Two
errors from the same factory match no matter what arguments they were built with, and an error
matches the factory that built it.

```go
errors.Is(ErrUserNotFound.New(1), ErrUserNotFound.New(2)) // true
errors.Is(ErrUserNotFound.New(1), ErrUserNotFound)        // true
errors.Is(ErrUserNotFound.New(1), ErrStoreClosed)         // false
```

Be aware of what that does *not* mean. Falta compares format strings and, failing that, falls
back to comparing rendered messages — it does not compare factory instances. So two separately
declared factories that share a format string are interchangeable, and any error whose message
happens to equal a falta error's message will match it:

```go
a := falta.Newf("boom: %s")
b := falta.Newf("boom: %s") // a distinct factory, identical declaration

errors.Is(a.New("x"), b.New("y"))            // true  — same format string
errors.Is(a.New("x"), errors.New("boom: x")) // true  — same rendered message
```

In practice this is rarely a problem: give each error its own message, the way you would
anyway, and matching behaves the way you expect. It matters if you were counting on two
same-worded errors in different packages staying distinguishable — they won't be.

## Things that will bite you

- **`NewError` and `Annotate` panic on format verbs.** Both take literal strings, so a stray
  `%s` is a bug — falta reports it loudly at init time rather than emitting `%!s(MISSING)`
  into your logs.
- **`falta.New[T]` panics on an invalid template.** Also at init time, by design: a broken
  error message should not first surface during an incident.
- **Calling `New()` with no arguments returns the raw format string** as the error message.
  That's the intended behavior for `NewError`-style use of a factory, but it means a forgotten
  argument shows up as a literal `%s` or `{{.Field}}` rather than a compile error.
- **`Extend` panics on mismatched factory kinds.** A `Newf` factory can only be extended by
  another `Newf` factory, and a `New[T]`/`NewM` factory only by one of the same type.

## Contributing

```sh
go test ./...                          # tests and runnable examples
go test -race -cover ./...             # what CI runs
gofmt -l .                             # must print nothing
golangci-lint run ./...                # config in .golangci.yml
```

CI runs the test suite against Go 1.21 through 1.24, plus lint, `gofmt`, and a `go mod tidy`
check. The examples in this README are backed by runnable
[example tests](./example_test.go), so they're verified on every commit.

## License

[MIT](./LICENSE)

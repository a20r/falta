# falta

[![CI](https://github.com/a20r/falta/actions/workflows/ci.yml/badge.svg)](https://github.com/a20r/falta/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/a20r/falta.svg)](https://pkg.go.dev/github.com/a20r/falta)
[![Go Report Card](https://goreportcard.com/badge/github.com/a20r/falta)](https://goreportcard.com/report/github.com/a20r/falta)

**Declare your errors once, with their message, and get `errors.Is` matching for free.**

Falta is a small error package for Go. You define an error *factory* at the
package level — message template included — and every error it produces is automatically
matchable against that factory with `errors.Is`.

## The problem

Go gives you two ways to build an error, and they pull in opposite directions:

- `errors.New` gives you a **matchable** sentinel, but it can't carry any context.
- `fmt.Errorf` gives you **context**, but the result matches nothing unless you remember `%w`.

So you end up writing both, and saying the same thing twice:

```go
var ErrUserNotFound = errors.New("user not found")

func GetUser(id int) (*User, error) {
    // "user not found" is now written in two places, and the %w is load-bearing.
    return nil, fmt.Errorf("user not found: id=%d: %w", id, ErrUserNotFound)
}
```

This gets worse at scale. The message drifts between call sites, the sentinel and the format
string live far apart, and if anyone drops the `%w` — or writes `%v` — `errors.Is` starts
returning `false` and nothing in the compiler or the test suite tells you.

With falta the message lives in exactly one place, and matching is not something a call site
can forget to do:

```go
var ErrUserNotFound = falta.Newf("user not found: id=%d")

func GetUser(id int) (*User, error) {
    return nil, ErrUserNotFound.New(id)
}
```

```go
_, err := GetUser(42)
fmt.Println(err)                            // user not found: id=42
fmt.Println(errors.Is(err, ErrUserNotFound)) // true
```

## Install

```sh
go get github.com/a20r/falta
```

Falta requires Go 1.21+ and has no runtime dependencies.

## Quick start

Declare a factory per error condition, next to the code that returns it:

```go
package users

import "github.com/a20r/falta"

var (
    ErrUserNotFound = falta.Newf("users: not found: id=%d")
    ErrInvalidEmail = falta.Newf("users: invalid email %q for id=%d")
)

func (s *Store) Get(id int) (*User, error) {
    u, ok := s.byID[id]

    if !ok {
        return nil, ErrUserNotFound.New(id)
    }

    return u, nil
}
```

Callers match on the factory, exactly like a sentinel:

```go
u, err := store.Get(42)

switch {
case errors.Is(err, users.ErrUserNotFound):
    http.Error(w, "no such user", http.StatusNotFound)
case err != nil:
    http.Error(w, "internal error", http.StatusInternalServerError)
}
```

That's the whole idea. Everything below is refinement.

## Building errors

Falta has four constructors, depending on how you want to render the message.

### `falta.Newf` — printf formatting

```go
var ErrCannotOpenConfig = falta.Newf("cannot open config %s")

err := ErrCannotOpenConfig.New("app.yaml")
// cannot open config app.yaml
```

### `falta.New[T]` — a template over your own struct

Best when the error is about a domain value you already have in hand. The message is a
[`text/template`](https://pkg.go.dev/text/template), so fields are named rather than positional
and can't be silently reordered.

```go
type Circle struct {
    Radius float64
}

var ErrInvalidCircle = falta.New[Circle]("invalid circle: radius ({{.Radius}}) <= 0")

err := ErrInvalidCircle.New(Circle{Radius: -1})
// invalid circle: radius (-1) <= 0
```

### `falta.NewM` — a template over a map

The same thing without declaring a type, for one-off structured messages.

```go
var ErrCallFailed = falta.NewM("call failed: [code={{.code}}] {{.message}}")

err := ErrCallFailed.New(falta.M{
    "code":    503,
    "message": "Bad Gateway",
})
// call failed: [code=503] Bad Gateway
```

### `falta.NewError` — a plain sentinel

For errors that carry no context at all. It returns a `falta.Falta` directly rather than a
factory, so there is no `.New(...)` call.

```go
var ErrShutdown = falta.NewError("server: shutting down")

err := fmt.Errorf("worker stopped: %w", ErrShutdown)
// worker stopped: server: shutting down
errors.Is(err, ErrShutdown) // true
```

> **Note:** `NewError` and `Annotate` panic at construction time if the string contains a
> printf verb (`%s`, `%d`, …). That's deliberate — it catches "I meant to format this" bugs
> at init instead of shipping a literal `%s` to your users.

## Adding context

### `Wrap` — attach a cause

The wrapped error stays reachable through `errors.Is` / `errors.As` / `errors.Unwrap`, so you
can match on either layer.

```go
_, ioErr := os.Open("app.yaml")
err := ErrCannotOpenConfig.New("app.yaml").Wrap(ioErr)

fmt.Println(err)
// cannot open config app.yaml: open app.yaml: no such file or directory

errors.Is(err, ErrCannotOpenConfig) // true — your error
errors.Is(err, os.ErrNotExist)      // true — the cause
```

### `Annotate` — add a note without a cause

```go
err := ErrCannotOpenConfig.New("app.yaml").
    Annotate("run `myapp init` first").
    Wrap(ioErr)
// cannot open config app.yaml: run `myapp init` first: open app.yaml: no such file or directory
```

### `Capture` — wrap every error a function returns

`Capture` takes a pointer to a named return value and wraps it on the way out, if it's non-nil.
Deferred at the top of a function, it guarantees that every error path is tagged — including
the ones added later by someone who wasn't thinking about your sentinel.

```go
func LoadConfig(name string) (cfg []byte, err error) {
    defer ErrCannotOpenConfig.New(name).Capture(&err)

    // Any error returned below comes back wrapped by ErrCannotOpenConfig.
    return os.ReadFile(name)
}
```

```go
_, err := LoadConfig("app.yaml")
// cannot open config app.yaml: open app.yaml: no such file or directory

errors.Is(err, ErrCannotOpenConfig) // true
errors.Is(err, os.ErrNotExist)      // true
```

This requires a **named** error return value — that's what gives `Capture` something to point
at. A `nil` error is left alone.

## Extending factories

`Newf` and `NewM` return an `ExtendableFactory`, which can be composed into a more specific
variant of the same error. The message templates are joined with a space.

```go
var (
    ErrQueryFailed       = falta.NewM("query failed: [code={{.code}}] {{.message}}")
    ErrQueryFailedReason = ErrQueryFailed.Extend(falta.NewM("because {{.reason}}"))
)

err := ErrQueryFailedReason.New(falta.M{
    "code":    503,
    "message": "Bad Gateway",
    "reason":  "primary is down",
})
// query failed: [code=503] Bad Gateway because primary is down
```

An extended factory is a **distinct** error identity, not a subtype:

```go
errors.Is(err, ErrQueryFailedReason) // true
errors.Is(err, ErrQueryFailed)       // false
```

If you want callers to be able to match the general case, return the base factory's error and
`Wrap` the specific one — or match on the extended factory directly.

A factory can only be extended by another factory of the same kind: a printf factory by
another printf factory, a template factory by another template factory. Mixing the two panics
at init.

## API at a glance

| Function | Returns | Message style |
| --- | --- | --- |
| `falta.Newf(format)` | `ExtendableFactory[any]` | printf verbs |
| `falta.New[T](tmpl)` | `Factory[T]` | `text/template` over `T` |
| `falta.NewM(tmpl)` | `ExtendableFactory[M]` | `text/template` over `falta.M` |
| `falta.NewError(msg)` | `Falta` | literal, no formatting |

| Method on `Falta` | Effect |
| --- | --- |
| `Wrap(err)` | Appends `err` as the cause; reachable via `errors.Is`/`Unwrap` |
| `Annotate(note)` | Appends a note to the message |
| `Capture(&err)` | `defer`-friendly: wraps a non-nil named return value |
| `Unwrap()` | Returns the wrapped cause, or `nil` |
| `Is(target)` | Reports whether `target` came from the same factory |

Factories implement `error` themselves, which is why they can be passed straight to
`errors.Is` as a target.

## Development

```sh
go test -race -cover ./...   # tests (statement coverage is currently 98.3%)
golangci-lint run            # lint (config in .golangci.yml)
go mod tidy                  # deps
```

CI runs the same gates on every pull request and on `main`, against the oldest supported Go
release and the current one, and fails any run whose statement coverage drops below 95%. The
suite includes an `errors.Is` truth table, fuzz targets for the construction invariants, a
race-detector concurrency test, and runnable examples for every constructor. See
[`.github/workflows/ci.yml`](.github/workflows/ci.yml).

## License

[MIT](LICENSE)

package falta_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/a20r/falta"
	"github.com/stretchr/testify/assert"
)

// verbPattern mirrors the guard inside falta so the fuzz targets can predict which inputs must panic.
var verbPattern = regexp.MustCompile(`\%\w`)

// FuzzNewError asserts NewError's two contracts over arbitrary strings: messages containing printf verbs panic at
// construction, and everything else round-trips literally and stays matchable through wrapping. The seed corpus runs
// as part of the normal test suite in CI.
func FuzzNewError(f *testing.F) {
	for _, seed := range []string{"", "plain", "50%% done", "100%", "%s", "a: b: c", "café ☕"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, msg string) {
		if verbPattern.MatchString(msg) {
			assert.Panics(t, func() { _ = falta.NewError(msg) })
			return
		}

		err := falta.NewError(msg)
		assert.Equal(t, msg, err.Error())
		assert.ErrorIs(t, err, err)
		assert.ErrorIs(t, fmt.Errorf("ctx: %w", err), err)
	})
}

// FuzzWrap asserts the wrapping invariants over arbitrary message/cause pairs: the message is always
// "<msg>: <cause>", the cause is always reachable through errors.Is and errors.Unwrap.
func FuzzWrap(f *testing.F) {
	f.Add("op failed", "cause")
	f.Add("", "")
	f.Add("50%% done", "disk 90% full")

	f.Fuzz(func(t *testing.T, msg, causeMsg string) {
		if verbPattern.MatchString(msg) {
			t.Skip("verb strings panic at construction; covered by FuzzNewError")
		}

		cause := errors.New(causeMsg)
		err := falta.NewError(msg).Wrap(cause)

		assert.Equal(t, msg+": "+causeMsg, err.Error())
		assert.ErrorIs(t, err, cause)
		assert.Equal(t, cause, errors.Unwrap(err))
	})
}

// FuzzTemplateData asserts that data flowing through a template lands in the message verbatim — no printf
// re-interpretation, no escaping — for arbitrary strings.
func FuzzTemplateData(f *testing.F) {
	for _, seed := range []string{"plain", "100%s", "50%% off", "<html>&", "{{.v}}", "🎉"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, v string) {
		err := falta.NewM("value {{.v}}").New(falta.M{"v": v})
		assert.Equal(t, "value "+v, err.Error())
	})
}

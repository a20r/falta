package falta_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/a20r/falta"
	"github.com/stretchr/testify/assert"
)

// TestErrorsIsMatrix pins the errors.Is matching semantics in both directions for every pairing users are likely to
// rely on. Matching is by identity of the factory's format string: two errors from the same factory match regardless
// of their arguments, and a factory matches every error it produced.
func TestErrorsIsMatrix(t *testing.T) {
	errNotFound := falta.Newf("users: not found: id=%d")
	errTimeout := falta.Newf("users: timeout after %dms")
	errRejected := falta.NewM("orders: rejected: {{.reason}}")

	err := errNotFound.New(42)
	tmplErr := errRejected.New(falta.M{"reason": "empty cart"})

	cases := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{"error matches its own factory", err, errNotFound, true},
		{"factory matches its own error", errNotFound, err, true},
		{"errors from the same factory match regardless of args", errNotFound.New(1), errNotFound.New(2), true},
		{"a second factory with the identical format matches", err, falta.Newf("users: not found: id=%d"), true},
		{"different factory does not match", err, errTimeout, false},
		{"error from a different factory does not match", err, errTimeout.New(9), false},
		{"template error matches its factory", tmplErr, errRejected, true},
		{"template factory matches its error", errRejected, tmplErr, true},
		{"template errors with different data match", errRejected.New(falta.M{"reason": "a"}), errRejected.New(falta.M{"reason": "b"}), true},
		{"template error does not match a printf factory", tmplErr, errNotFound, false},
		{"plain error carrying the rendered message matches", err, errors.New("users: not found: id=42"), true},
		{"plain error carrying the raw format string matches", err, errors.New("users: not found: id=%d"), true},
		{"unrelated plain error does not match", err, errors.New("boom"), false},
		{"nil target does not match", err, nil, false},
		{"wrapped cause matches", errNotFound.New(1).Wrap(io.EOF), io.EOF, true},
		{"unrelated sentinel does not match a wrapped error", errNotFound.New(1).Wrap(io.EOF), io.ErrUnexpectedEOF, false},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, errors.Is(tc.err, tc.target))
		})
	}
}

func TestErrorsAs(t *testing.T) {
	errNotFound := falta.Newf("users: not found: id=%d")

	t.Run("extracts the Falta from a deep chain", func(t *testing.T) {
		err := errNotFound.New(7)
		deep := fmt.Errorf("a: %w", fmt.Errorf("b: %w", err))

		var f falta.Falta
		assert.ErrorAs(t, deep, &f)
		assert.Equal(t, err, f)
	})

	t.Run("reports false for chains without a Falta", func(t *testing.T) {
		var f falta.Falta
		assert.False(t, errors.As(fmt.Errorf("a: %w", io.EOF), &f))
	})
}

func TestExtendedFactoryIdentity(t *testing.T) {
	base := falta.NewM("call failed: code={{.code}}")
	ext := base.Extend(falta.NewM("because {{.reason}}"))

	err := ext.New(falta.M{"code": 503, "reason": "downstream"})

	assert.ErrorIs(t, err, ext)
	assert.NotErrorIs(t, err, base, "an extended factory is a distinct identity")
	assert.NotErrorIs(t, base.New(falta.M{"code": 503}), ext)
}

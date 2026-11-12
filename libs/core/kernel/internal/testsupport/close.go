package testsupport

import (
	"io"
	"testing"
)

// Close closes c and fails the test when closing returns an error.
func Close(tb testing.TB, c io.Closer) {
	tb.Helper()
	if err := c.Close(); err != nil {
		tb.Errorf("close: %v", err)
	}
}

// CloseAfter closes c when the test ends and fails the test when closing returns an error.
func CloseAfter(tb testing.TB, c io.Closer) {
	tb.Helper()
	tb.Cleanup(func() { Close(tb, c) })
}

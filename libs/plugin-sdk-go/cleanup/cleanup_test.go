package cleanup

import (
	"errors"
	"testing"
)

// recorder is a test whose failures are counted and do not fail the test that holds it.
type recorder struct {
	testing.TB
	failures int
}

func (r *recorder) Errorf(string, ...any) { r.failures++ }
func (r *recorder) Helper()               {}

type closer struct{ err error }

func (c closer) Close() error { return c.err }

func TestClose_ReportsTheError(t *testing.T) {
	r := &recorder{TB: t}
	Close(r, closer{err: errors.New("busy")})
	if r.failures != 1 {
		t.Fatalf("a failed close reported %d failures, want 1", r.failures)
	}
}

func TestClose_AcceptsASuccessfulClose(t *testing.T) {
	r := &recorder{TB: t}
	Close(r, closer{})
	if r.failures != 0 {
		t.Fatalf("a successful close reported %d failures, want 0", r.failures)
	}
}

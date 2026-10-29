// Package enumtest checks that every value of a wire enumeration is handled.
package enumtest

import "testing"

// CheckHandled fails for every value that reading does not accept.
func CheckHandled[E comparable](t *testing.T, values []E, reading func(E) bool) {
	t.Helper()
	for _, value := range values {
		if !reading(value) {
			t.Errorf("%v reaches nothing", value)
		}
	}
}

// CheckProduced fails for every value that has no entry in from or that writing does not produce
// from its entry.
func CheckProduced[E comparable, C any](t *testing.T, values []E, from map[E]C, writing func(C) E) {
	t.Helper()
	for _, value := range values {
		entry, isPresent := from[value]
		if !isPresent {
			t.Errorf("%v has no counterpart", value)
			continue
		}
		if got := writing(entry); got != value {
			t.Errorf("%v is written as %v", entry, got)
		}
	}
}

// RoundTrip fails for every value that does not come back as itself.
func RoundTrip[E comparable, C any](t *testing.T, values []E, reading func(E) C, writing func(C) E) {
	t.Helper()
	for _, value := range values {
		if got := writing(reading(value)); got != value {
			t.Errorf("%v comes back as %v", value, got)
		}
	}
}

package testsupport

import (
	"testing"

	"github.com/jiva-studio/numen/libs/protocol/schematest"
)

// The protocol module's schematest checks, for the core and the plugins alike,
// that every value of a schema enum is handled.

// SchemaEnum is one enum of the schema, as the generated code declares it.
type SchemaEnum = schematest.SchemaEnum

// CheckHandled fails for every value of the enum that is not handled on input.
func CheckHandled[E SchemaEnum](t *testing.T, reading func(E) bool) {
	t.Helper()
	schematest.CheckHandled(t, reading)
}

// CheckProduced fails for every value of the enum nothing on this side is written as.
func CheckProduced[E SchemaEnum, C any](t *testing.T, from map[E]C, writing func(C) E) {
	t.Helper()
	schematest.CheckProduced(t, from, writing)
}

// RoundTrip fails for every value of the enum that does not come back as itself.
func RoundTrip[E SchemaEnum, C any](t *testing.T, reading func(E) C, writing func(C) E) {
	t.Helper()
	schematest.RoundTrip(t, reading, writing)
}

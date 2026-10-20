package testsupport

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/protocol/schematest"
)

// Whether every value of a schema enum reaches something is checked by the protocol module's
// schematest, for the core and the plugins alike.

// SchemaEnum is one enum of the schema, as the generated code declares it.
type SchemaEnum = schematest.SchemaEnum

// CheckHandled fails for every value of the enum that reaches nothing on the way in.
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

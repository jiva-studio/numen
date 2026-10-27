// Package schematest checks that every value of an enum of the schema reaches something.
package schematest

import (
	"testing"

	protoschematest "github.com/jiva-studio/numen/libs/protocol/schematest"
)

// SchemaEnum is one enum of the schema, as the generated code declares it.
type SchemaEnum = protoschematest.SchemaEnum

// CheckHandled fails for every value of the enum that reaches nothing on the way in.
func CheckHandled[E SchemaEnum](t *testing.T, reading func(E) bool) {
	t.Helper()
	protoschematest.CheckHandled(t, reading)
}

// CheckProduced fails for every value of the enum nothing on this side is written as.
func CheckProduced[E SchemaEnum, C any](t *testing.T, from map[E]C, writing func(C) E) {
	t.Helper()
	protoschematest.CheckProduced(t, from, writing)
}

// RoundTrip fails for every value of the enum that does not come back as itself.
func RoundTrip[E SchemaEnum, C any](t *testing.T, reading func(E) C, writing func(C) E) {
	t.Helper()
	protoschematest.RoundTrip(t, reading, writing)
}

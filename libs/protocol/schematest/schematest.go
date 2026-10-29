// Package schematest checks, for a test, that every value of a schema enum is handled.
package schematest

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Whether every value of a schema enum is handled.
//
// The linter reads a switch and stops at its default, and every switch over a
// schema enum here writes one; a map keyed by an enum it does not read at all.
// So a value added to the schema compiles, runs, and does nothing. What each
// check below walks comes from the descriptor the generated code carries, so
// the schema is the only list of values there is.

// SchemaEnum is one enum of the schema, as the generated code declares it.
type SchemaEnum interface {
	~int32
	protoreflect.Enum
}

// CheckHandled fails for every value of the enum that has no handler on the
// way in. Reading reports whether the value was acted on.
func CheckHandled[E SchemaEnum](t *testing.T, reading func(E) bool) {
	t.Helper()
	checkEachValue(t, func(t *testing.T, value protoreflect.EnumValueDescriptor, one E) {
		if !reading(one) {
			t.Errorf("%s reaches nothing on the way in", value.Name())
		}
	})
}

// CheckProduced fails for every value of the enum that nothing on this side
// produces. From names what each value is written from, and writing does the
// writing, so a value no one names and a value named for the wrong thing both
// fail.
func CheckProduced[E SchemaEnum, C any](t *testing.T, from map[E]C, writing func(C) E) {
	t.Helper()
	checkEachValue(t, func(t *testing.T, value protoreflect.EnumValueDescriptor, one E) {
		was, named := from[one]
		if !named {
			t.Errorf("%s is written from nothing", value.Name())
			return
		}
		if got := writing(was); got != one {
			t.Errorf("%s is written from %v, which is written as %v", value.Name(), was, got)
		}
	})
}

// RoundTrip fails for every value of the enum that does not come back as
// itself. A value the reading does not name lands on its default, and the
// writing sends that default back in its place.
func RoundTrip[E SchemaEnum, C any](t *testing.T, reading func(E) C, writing func(C) E) {
	t.Helper()
	checkEachValue(t, func(t *testing.T, value protoreflect.EnumValueDescriptor, one E) {
		if got := writing(reading(one)); got != one {
			t.Errorf("%s comes back as %v", value.Name(), got)
		}
	})
}

// checkEachValue runs check over every value of the enum but the unspecified zero, which
// is what a field carries when nothing was set and has no action.
func checkEachValue[E SchemaEnum](t *testing.T, check func(*testing.T, protoreflect.EnumValueDescriptor, E)) {
	t.Helper()
	var zero E
	values := zero.Descriptor().Values()
	for i := range values.Len() {
		value := values.Get(i)
		if value.Number() == 0 {
			continue
		}
		check(t, value, E(value.Number()))
	}
}

package service

import "testing"

// getField returns object[key] as a T and fails the test when object is not a JSON object or holds
// no T under key.
func getField[T any](t *testing.T, object any, key string) T {
	t.Helper()
	var zero T
	fields, ok := object.(map[string]any)
	if !ok {
		t.Fatalf("%T is not a JSON object", object)
		return zero
	}
	value, ok := fields[key].(T)
	if !ok {
		t.Fatalf("%q holds %T, want %T", key, fields[key], zero)
		return zero
	}
	return value
}

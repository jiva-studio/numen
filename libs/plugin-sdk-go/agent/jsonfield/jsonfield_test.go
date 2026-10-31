package jsonfield

import "testing"

func TestGetStringReturnsEmptyForAbsentAndForeignTypes(t *testing.T) {
	object := map[string]any{"a": "x", "n": 1}
	if got := GetString(object, "a"); got != "x" {
		t.Errorf("a = %q, want x", got)
	}
	if got := GetString(object, "n"); got != "" {
		t.Errorf("n = %q, want empty", got)
	}
	if got := GetString(object, "missing"); got != "" {
		t.Errorf("missing = %q, want empty", got)
	}
}

func TestGetMapReturnsNilForAbsentAndForeignTypes(t *testing.T) {
	object := map[string]any{"m": map[string]any{"k": "v"}, "s": "text"}
	if got := GetMap(object, "m"); got["k"] != "v" {
		t.Errorf("m = %v, want the nested object", got)
	}
	if got := GetMap(object, "s"); got != nil {
		t.Errorf("s = %v, want nil", got)
	}
	if got := GetMap(object, "missing"); got != nil {
		t.Errorf("missing = %v, want nil", got)
	}
}

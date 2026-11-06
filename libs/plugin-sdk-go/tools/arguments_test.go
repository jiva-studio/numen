package tools

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
)

type shelveIn struct {
	Title string `json:"title"`
	Count int    `json:"count,omitempty"`
}

type shelveOut struct {
	Shelved string `json:"shelved"`
}

type countIn struct {
	Count int `json:"count,omitempty"`
}

// newShelvingRegistry adds shelve_book, whose title is required, and shelf_count, which requires
// nothing. runs counts the calls of either run function.
func newShelvingRegistry(t *testing.T) (registry *Registry, runs *atomic.Int32) {
	t.Helper()
	registry = New()
	runs = new(atomic.Int32)
	if err := Add(registry, Spec{Name: "shelve_book", Effect: ReadWrite}, func(_ context.Context, in shelveIn) (shelveOut, error) {
		runs.Add(1)
		return shelveOut{Shelved: in.Title}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := Add(registry, Spec{Name: "shelf_count", Effect: ReadOnly}, func(_ context.Context, in countIn) (countIn, error) {
		runs.Add(1)
		return in, nil
	}); err != nil {
		t.Fatal(err)
	}
	return registry, runs
}

func invoke(t *testing.T, registry *Registry, name string, arguments []byte) *CallResponse {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), Call{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// assertRefusedNaming fails the test unless the call was answered with an error result whose text
// names property, and the run function was not called.
func assertRefusedNaming(t *testing.T, res *CallResponse, runs *atomic.Int32, property string) {
	t.Helper()
	if res.GetError() != ErrorCodeUnspecified {
		t.Errorf("the call was answered with the error code %v, want an error result", res.GetError())
	}
	if !res.GetResult().GetIsError() || !strings.Contains(res.GetResult().GetText(), property) {
		t.Errorf("the call was answered %+v, want an error result naming %q", res.GetResult(), property)
	}
	if got := runs.Load(); got != 0 {
		t.Errorf("the run function was called %d times", got)
	}
}

func TestAnArgumentOfTheWrongTypeIsRefusedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"title":"Ferns of the valley","count":"three"}`))

	assertRefusedNaming(t, res, runs, "count")
}

func TestAMissingRequiredArgumentIsRefusedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"count":2}`))

	assertRefusedNaming(t, res, runs, "title")
}

func TestNoArgumentsForAToolWithARequiredArgumentAreRefusedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", nil)

	assertRefusedNaming(t, res, runs, "title")
}

func TestAnUndeclaredArgumentIsRefusedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"title":"Ferns of the valley","colour":"green"}`))

	assertRefusedNaming(t, res, runs, "colour")
}

func TestValidArgumentsRunTheTool(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"title":"Ferns of the valley","count":2}`))

	if res.GetResult().GetIsError() || string(res.GetResult().GetStructured()) != `{"shelved":"Ferns of the valley"}` {
		t.Errorf("the call was answered %+v", res.GetResult())
	}
	if got := runs.Load(); got != 1 {
		t.Errorf("the run function was called %d times, want once", got)
	}
}

func TestNoArgumentsRunAToolWithoutRequiredArguments(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelf_count", nil)

	if res.GetError() != ErrorCodeUnspecified || res.GetResult().GetIsError() {
		t.Errorf("the call was answered %+v with the error code %v", res.GetResult(), res.GetError())
	}
	if got := runs.Load(); got != 1 {
		t.Errorf("the run function was called %d times, want once", got)
	}
}

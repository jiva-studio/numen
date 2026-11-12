package tools

import (
	"context"
	"errors"
	"reflect"
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

func assertRefusedNaming(t *testing.T, res *CallResponse, runs *atomic.Int32, property string) {
	t.Helper()
	if res.GetError() != ErrorCodeUnspecified {
		t.Errorf("the call returned the error code %v, want an error result", res.GetError())
	}
	if !res.GetResult().GetIsError() || !strings.Contains(res.GetResult().GetText(), property) {
		t.Errorf("the call returned %+v, want an error result naming %q", res.GetResult(), property)
	}
	if got := runs.Load(); got != 0 {
		t.Errorf("the run function was called %d times", got)
	}
}

func TestArgumentOfWrongTypeIsRejectedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"title":"Ferns of the valley","count":"three"}`))

	assertRefusedNaming(t, res, runs, "count")
}

func TestMissingRequiredArgumentIsRejectedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"count":2}`))

	assertRefusedNaming(t, res, runs, "title")
}

func TestNoArgumentsForRequiredArgumentAreRejectedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", nil)

	assertRefusedNaming(t, res, runs, "title")
}

func TestUndeclaredArgumentIsRejectedNamingIt(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"title":"Ferns of the valley","color":"green"}`))

	assertRefusedNaming(t, res, runs, "color")
}

func TestValidArgumentsRunTheTool(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelve_book", []byte(`{"title":"Ferns of the valley","count":2}`))

	if res.GetResult().GetIsError() || string(res.GetResult().GetStructured()) != `{"shelved":"Ferns of the valley"}` {
		t.Errorf("the call returned %+v", res.GetResult())
	}
	if got := runs.Load(); got != 1 {
		t.Errorf("the run function was called %d times, want once", got)
	}
}

func TestNoArgumentsRunAToolWithoutRequiredArguments(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	res := invoke(t, registry, "shelf_count", nil)

	if res.GetError() != ErrorCodeUnspecified || res.GetResult().GetIsError() {
		t.Errorf("the call returned %+v with the error code %v", res.GetResult(), res.GetError())
	}
	if got := runs.Load(); got != 1 {
		t.Errorf("the run function was called %d times, want once", got)
	}
}

type linkInput struct {
	Target string `json:"target"`
	Role   string `json:"role,omitempty"`
	Rank   int    `json:"rank,omitempty"`
	Weight uint   `json:"weight,omitempty"`
}

type placeInput struct {
	Title string `json:"title"`
	Row   int    `json:"row,omitempty"`
}

type groupInput struct {
	Links []linkInput `json:"links"`
}

type catalogInput struct {
	Place  placeInput   `json:"place"`
	Links  []linkInput  `json:"links"`
	Groups []groupInput `json:"groups,omitempty"`
	Limit  int          `json:"limit,omitempty"`
	Note   string       `json:"note,omitempty"`
	Shelf  uint8        `json:"shelf,omitempty"`
	Pair   [2]string    `json:"pair,omitempty"`
}

// catalogRuns records the arguments catalog_update ran with.
type catalogRuns struct {
	atomic.Int32

	last atomic.Pointer[catalogInput]
}

func newCatalogRegistry(t *testing.T) (*Registry, *catalogRuns) {
	t.Helper()
	registry := New()
	runs := &catalogRuns{}
	if err := Add(registry, Spec{Name: "catalog_update", Effect: ReadWrite}, func(_ context.Context, in catalogInput) (shelveOut, error) {
		runs.Add(1)
		runs.last.Store(&in)
		return shelveOut{Shelved: in.Place.Title}, nil
	}); err != nil {
		t.Fatal(err)
	}
	return registry, runs
}

func runCatalog(t *testing.T, arguments string) catalogInput {
	t.Helper()
	registry, runs := newCatalogRegistry(t)
	res := invoke(t, registry, "catalog_update", []byte(arguments))
	if res.GetError() != ErrorCodeUnspecified || res.GetResult().GetIsError() || runs.Load() != 1 {
		t.Fatalf("the arguments %s were returned %+v with the error code %v", arguments, res.GetResult(), res.GetError())
	}
	return *runs.last.Load()
}

func TestNullForAnOptionalArgumentIsReadAsAbsent(t *testing.T) {
	for _, tc := range []struct {
		name      string
		arguments string
		want      catalogInput
	}{
		{
			name:      "top level",
			arguments: `{"place":{"title":"Ferns"},"links":[],"limit":null,"note":null,"groups":null,"shelf":null,"pair":null}`,
			want:      catalogInput{Place: placeInput{Title: "Ferns"}, Links: []linkInput{}},
		},
		{
			name:      "nested object",
			arguments: `{"place":{"title":"Ferns","row":null},"links":[]}`,
			want:      catalogInput{Place: placeInput{Title: "Ferns"}, Links: []linkInput{}},
		},
		{
			name:      "item of an array",
			arguments: `{"place":{"title":"Ferns"},"links":[{"target":"a.md","role":null},{"target":"b.md","role":"source","rank":null,"weight":null}],"limit":null}`,
			want: catalogInput{
				Place: placeInput{Title: "Ferns"},
				Links: []linkInput{{Target: "a.md"}, {Target: "b.md", Role: "source"}},
			},
		},
		{
			name:      "item of an array inside an item of an array",
			arguments: `{"place":{"title":"Ferns"},"links":[],"groups":[{"links":[{"target":"c.md","role":null}]}]}`,
			want: catalogInput{
				Place:  placeInput{Title: "Ferns"},
				Links:  []linkInput{},
				Groups: []groupInput{{Links: []linkInput{{Target: "c.md"}}}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := runCatalog(t, tc.arguments); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("catalog_update ran with %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestAWholeNumberIsAcceptedForAnIntegerArgument(t *testing.T) {
	for _, arguments := range []string{
		`{"place":{"title":"Ferns","row":20.0},"links":[{"target":"a.md","rank":20.0,"weight":20.0}],"limit":20.0,"shelf":20.0}`,
		`{"place":{"title":"Ferns","row":2e1},"links":[{"target":"a.md","rank":2e1,"weight":2E1}],"limit":2e1,"shelf":2.0e1}`,
		`{"place":{"title":"Ferns","row":20},"links":[{"target":"a.md","rank":20,"weight":20}],"limit":20,"shelf":20}`,
	} {
		got := runCatalog(t, arguments)
		if got.Limit != 20 || got.Place.Row != 20 || got.Shelf != 20 || len(got.Links) != 1 ||
			got.Links[0].Rank != 20 || got.Links[0].Weight != 20 {
			t.Errorf("the arguments %s ran catalog_update with %+v, want 20 for every integer", arguments, got)
		}
	}
}

func TestAFractionIsRejectedForAnIntegerArgument(t *testing.T) {
	registry, runs := newCatalogRegistry(t)

	for arguments, want := range map[string]string{
		`{"place":{"title":"Ferns"},"links":[],"limit":20.5}`:                                  `catalog_update: the argument limit must be an integer, not a number`,
		`{"place":{"title":"Ferns","row":0.5},"links":[]}`:                                     `catalog_update: the argument place.row must be an integer, not a number`,
		`{"place":{"title":"Ferns"},"links":[{"target":"a.md"},{"target":"b.md","rank":1.5}]}`: `catalog_update: the argument links[1].rank must be an integer, not a number`,
	} {
		res := invoke(t, registry, "catalog_update", []byte(arguments))
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != want {
			t.Errorf("the arguments %s were returned %+v, want the error %q", arguments, res.GetResult(), want)
		}
	}
	if got := runs.Load(); got != 0 {
		t.Errorf("catalog_update ran %d times on a fraction for an integer", got)
	}
}

func TestToolAddPrefixesTheToolNameToAFailureOnce(t *testing.T) {
	registry := New()
	if err := Add(registry, Spec{Name: "shelf_count", Effect: ReadOnly}, func(context.Context, countIn) (countIn, error) {
		return countIn{}, errors.New("the shelf is gone")
	}); err != nil {
		t.Fatal(err)
	}
	if err := Add(registry, Spec{Name: "shelve_book", Effect: ReadWrite}, func(context.Context, shelveIn) (shelveOut, error) {
		return shelveOut{}, nil
	}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ tool, arguments, want string }{
		{"shelf_count", `{}`, "shelf_count: the shelf is gone"},
		{"shelve_book", `{}`, "shelve_book: the argument title is missing"},
		{"shelve_book", `{"title":"Ferns","count":"three"}`, "shelve_book: the argument count must be an integer, not a string"},
	} {
		res := invoke(t, registry, tc.tool, []byte(tc.arguments))
		if res.GetError() != ErrorCodeUnspecified || !res.GetResult().GetIsError() || res.GetResult().GetText() != tc.want {
			t.Errorf("%s %s returned %+v with the error code %v, want the error %q",
				tc.tool, tc.arguments, res.GetResult(), res.GetError(), tc.want)
		}
	}
}

func TestErrorCodesOfACallCarryNoText(t *testing.T) {
	registry, _ := newShelvingRegistry(t)

	for name, arguments := range map[string]string{"shelf_drop": `{}`, "shelve_book": `[1]`} {
		res := invoke(t, registry, name, []byte(arguments))
		if res.GetError() == ErrorCodeUnspecified || res.GetResult() != nil {
			t.Errorf("%s %s returned %+v with the error code %v, want an error code and no result",
				name, arguments, res.GetResult(), res.GetError())
		}
	}
}

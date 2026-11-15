package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

type locationInput struct {
	Title string `json:"title"`
}

type bookInput struct {
	Shelf  locationInput `json:"location"`
	Pages  int           `json:"pages,omitempty"`
	Tags   []string      `json:"tags,omitempty"`
	Author string        `json:"author"`
}

func newBookRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	if err := Add(registry, Spec{Name: "book_create", Effect: ReadWrite}, func(_ context.Context, in bookInput) (bookInput, error) {
		return in, nil
	}); err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestValidationMessagesNameTheArgumentInPlainWords(t *testing.T) {
	registry := newBookRegistry(t)

	for arguments, want := range map[string]string{
		`{"location":{"title":"Ferns"},"author":"Mira","pages":"many"}`:  `book_create: the argument pages must be an integer, not a string`,
		`{"location":{"title":"Ferns"},"author":"Mira","tags":[3]}`:      `book_create: the argument tags[0] must be a string, not an integer`,
		`{"location":{"title":"Ferns"}}`:                                 `book_create: the argument author is missing`,
		`{}`:                                                             `book_create: the arguments location, author are missing`,
		`{"location":{},"author":"Mira"}`:                                `book_create: the argument location.title is missing`,
		`{"location":{"title":"Ferns","row":2},"author":"Mira"}`:         `book_create: the argument location.row is not known`,
		`{"location":{"title":"Ferns"},"author":"Mira","color":"green"}`: `book_create: the argument color is not known`,
	} {
		res := invoke(t, registry, "book_create", []byte(arguments))
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != want {
			t.Errorf("the arguments %s were returned %+v, want the error %q", arguments, res.GetResult(), want)
		}
	}
}

func TestATypeFailureNamesTheExpectedAndTheGivenType(t *testing.T) {
	books := newBookRegistry(t)
	catalog, _ := newCatalogRegistry(t)

	for _, tc := range []struct {
		tool, arguments, want string
	}{
		{"book_create", `{"location":"Ferns","author":"Mira"}`, `book_create: the argument location must be an object, not a string`},
		{"book_create", `{"location":{"title":"Ferns"},"author":["Mira"]}`, `book_create: the argument author must be a string, not an array`},
		{"book_create", `{"location":{"title":"Ferns"},"author":{"name":"Mira"}}`, `book_create: the argument author must be a string, not an object`},
		{"book_create", `{"location":{"title":"Ferns"},"author":true}`, `book_create: the argument author must be a string, not a boolean`},
		{"book_create", `{"location":{"title":"Ferns"},"author":2.5}`, `book_create: the argument author must be a string, not a number`},
		{"book_create", `{"location":{"title":"Ferns"},"author":"Mira","tags":"moss"}`, `book_create: the argument tags must be an array, not a string`},
		{"book_create", `{"location":{"title":"Ferns"},"author":"Mira","tags":["moss",7]}`, `book_create: the argument tags[1] must be a string, not an integer`},
		{"catalog_update", `{"place":{"title":"Ferns"},"links":[{"target":"a.md"},{"target":"b.md","role":7}]}`, `catalog_update: the argument links[1].role must be a string, not an integer`},
		{"catalog_update", `{"place":{"title":"Ferns"},"links":[],` +
			`"groups":[{"links":[]},{"links":[]},{"links":[{"target":true}]}]}`, `catalog_update: the argument groups[2].links[0].target must be a string, not a boolean`},
		{"catalog_update", `{"place":{"title":"Ferns","row":"third"},"links":[]}`, `catalog_update: the argument place.row must be an integer, not a string`},
	} {
		registry := books
		if tc.tool == "catalog_update" {
			registry = catalog
		}
		res := invoke(t, registry, tc.tool, []byte(tc.arguments))
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != tc.want {
			t.Errorf("the arguments %s were returned %+v, want the error %q", tc.arguments, res.GetResult(), tc.want)
		}
	}
}

func TestANullRequiredArgumentIsRejectedNamingItsType(t *testing.T) {
	books := newBookRegistry(t)
	catalog, runs := newCatalogRegistry(t)

	for _, tc := range []struct {
		registry              *Registry
		tool, arguments, want string
	}{
		{books, "book_create", `{"location":{"title":"Ferns"},"author":null}`, `book_create: the argument author must be a string, not null`},
		{books, "book_create", `{"location":{"title":null},"author":"Mira"}`, `book_create: the argument location.title must be a string, not null`},
		{books, "book_create", `{"location":null,"author":"Mira"}`, `book_create: the argument location must be an object, not null`},
		{catalog, "catalog_update", `{"place":{"title":"Ferns"},"links":[{"target":"a.md"},{"target":null}]}`, `catalog_update: the argument links[1].target must be a string, not null`},
	} {
		res := invoke(t, tc.registry, tc.tool, []byte(tc.arguments))
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != tc.want {
			t.Errorf("the arguments %s were returned %+v, want the error %q", tc.arguments, res.GetResult(), tc.want)
		}
	}
	if got := runs.Load(); got != 0 {
		t.Errorf("catalog_update ran %d times on a null required argument", got)
	}
}

func TestAnArgumentFailingAnotherKeywordIsInvalidInPlainWords(t *testing.T) {
	catalog, _ := newCatalogRegistry(t)

	for arguments, want := range map[string]string{
		`{"place":{"title":"Ferns"},"links":[],"shelf":300}`:                                             "catalog_update: the argument shelf is invalid: it must be at most 255",
		`{"place":{"title":"Ferns"},"links":[],"shelf":-1}`:                                              "catalog_update: the argument shelf is invalid: it must be at least 0",
		`{"place":{"title":"Ferns"},"links":[],"pair":["a.md"]}`:                                         "catalog_update: the argument pair is invalid: it must have exactly 2 items",
		`{"place":{"title":"Ferns"},"links":[{"target":"a.md","rank":2},{"target":"b.md","weight":-4}]}`: "catalog_update: the argument links[1].weight is invalid: it must be at least 0",
	} {
		res := invoke(t, catalog, "catalog_update", []byte(arguments))
		text := res.GetResult().GetText()
		if !res.GetResult().GetIsError() || text != want {
			t.Errorf("the arguments %s were returned %+v, want %q", arguments, res.GetResult(), want)
		}
		assertPlainFailureText(t, text)
	}
	for _, valid := range []string{
		`{"place":{"title":"Ferns"},"links":[],"shelf":0,"pair":["a.md","b.md"]}`,
		`{"place":{"title":"Ferns"},"links":[],"shelf":255}`,
	} {
		_ = runCatalog(t, valid)
	}

	if err := checkType(&jsonschema.Schema{Type: typeNull}, json.Number("1"), "x"); err == nil || err.Error() != "the argument x must be null, not an integer" {
		t.Errorf("checkType(null, 1) = %v", err)
	}

	intPtr := func(v int) *int { return &v }
	for _, tc := range []struct {
		schema *jsonschema.Schema
		length int
		want   string
	}{
		{&jsonschema.Schema{MinItems: intPtr(2)}, 1, "it must have at least 2 items"},
		{&jsonschema.Schema{MinItems: intPtr(2)}, 2, ""},
		{&jsonschema.Schema{MinItems: intPtr(2)}, 3, ""},
		{&jsonschema.Schema{MaxItems: intPtr(2)}, 1, ""},
		{&jsonschema.Schema{MaxItems: intPtr(2)}, 2, ""},
		{&jsonschema.Schema{MaxItems: intPtr(2)}, 3, "it must have at most 2 items"},
		{&jsonschema.Schema{MinItems: intPtr(2), MaxItems: intPtr(2)}, 1, "it must have exactly 2 items"},
		{&jsonschema.Schema{MinItems: intPtr(2), MaxItems: intPtr(2)}, 2, ""},
		{&jsonschema.Schema{MinItems: intPtr(2), MaxItems: intPtr(2)}, 3, "it must have exactly 2 items"},
		{&jsonschema.Schema{MinItems: intPtr(1), MaxItems: intPtr(3)}, 0, "it must have at least 1 items"},
		{&jsonschema.Schema{MinItems: intPtr(1), MaxItems: intPtr(3)}, 1, ""},
		{&jsonschema.Schema{MinItems: intPtr(1), MaxItems: intPtr(3)}, 3, ""},
		{&jsonschema.Schema{MinItems: intPtr(1), MaxItems: intPtr(3)}, 4, "it must have at most 3 items"},
	} {
		if got := checkLength(tc.schema, tc.length); got != tc.want {
			t.Errorf("checkLength(%+v, %d) = %q, want %q", tc.schema, tc.length, got, tc.want)
		}
	}
}

// Every failure text is one sentence about the JSON arguments: it names no Go type, struct or field
// and carries no formatting artifact or keyword of the schema validator.
func TestAFailureTextNamesNoGoTypeOrValidatorKeyword(t *testing.T) {
	catalog, _ := newCatalogRegistry(t)
	books := newBookRegistry(t)

	for _, arguments := range []string{
		`{"place":{"title":"Ferns"},"links":[{"target":"a.md","rank":1.5}]}`,
		`{"place":{"title":"Ferns"},"links":[{"target":"a.md","role":null,"rank":"high"}]}`,
		`{"place":{"title":"Ferns","row":2.5},"links":[]}`,
		`{"place":{"title":"Ferns"},"links":[],"limit":20.5}`,
		`{"place":{"title":"Ferns"},"links":[],"limit":"twenty"}`,
		`{"place":{"title":"Ferns"},"links":[],"shelf":300}`,
		`{"place":{"title":"Ferns"},"links":[],"pair":["a.md","b.md","c.md"]}`,
		`{"place":{"title":"Ferns"},"links":[{"target":null}]}`,
		`{"place":{"title":"Ferns"},"links":"a.md"}`,
		`{"place":{},"links":[{}]}`,
		`{"place":{"title":"Ferns"},"links":[{"target":"a.md","color":"green"}]}`,
	} {
		res := invoke(t, catalog, "catalog_update", []byte(arguments))
		if !res.GetResult().GetIsError() {
			t.Errorf("the arguments %s ran the tool: %+v", arguments, res.GetResult())
			continue
		}
		assertPlainFailureText(t, res.GetResult().GetText())
	}
	for _, arguments := range []string{
		`{"location":{"title":"Ferns"},"author":null}`,
		`{"location":{"title":"Ferns"},"author":"Mira","pages":"many"}`,
		`{"location":{"title":"Ferns"},"author":"Mira","tags":[3]}`,
	} {
		assertPlainFailureText(t, invoke(t, books, "book_create", []byte(arguments)).GetResult().GetText())
	}
}

func assertPlainFailureText(t *testing.T, text string) {
	t.Helper()
	for _, fragment := range []string{
		"<invalid reflect.Value>", "%!", "type:", "required:", "minimum:", "maximum:", "minItems:", "maxItems:",
		"catalogInput", "linkInput", "placeInput", "groupInput", "bookInput", "locationInput",
		"Go struct", "Go value", "json:", "float64", "uint8", "int8", "[]string", "map[",
		"Place", "Links", "Groups", "Limit", "Shelf", "Pair", "Target", "Role", "Rank", "Weight", "Row", "Title", "Author",
	} {
		if strings.Contains(text, fragment) {
			t.Errorf("the failure text %q contains %q", text, fragment)
		}
	}
	if strings.Count(text, "catalog_update: ")+strings.Count(text, "book_create: ") != 1 {
		t.Errorf("the failure text %q does not start with the tool name exactly once", text)
	}
	if strings.Contains(text, "\n") {
		t.Errorf("the failure text %q is more than one line", text)
	}
}

func TestNullArgumentsAreTheEmptyObject(t *testing.T) {
	registry, runs := newShelvingRegistry(t)

	counted := invoke(t, registry, "shelf_count", []byte(`null`))
	if counted.GetError() != ErrorCodeUnspecified || counted.GetResult().GetIsError() || runs.Load() != 1 {
		t.Errorf("shelf_count with null arguments returned %+v with the error code %v after %d runs",
			counted.GetResult(), counted.GetError(), runs.Load())
	}

	shelved := invoke(t, registry, "shelve_book", []byte(`null`))
	if text := shelved.GetResult().GetText(); !shelved.GetResult().GetIsError() || text != "shelve_book: the argument title is missing" {
		t.Errorf("shelve_book with null arguments returned %+v, want the error %q", shelved.GetResult(), "shelve_book: the argument title is missing")
	}
	if got := runs.Load(); got != 1 {
		t.Errorf("the run functions were called %d times, want once", got)
	}
}

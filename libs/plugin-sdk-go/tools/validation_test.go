package tools

import (
	"context"
	"testing"
)

type shelfIn struct {
	Title string `json:"title"`
}

type cataloguedIn struct {
	Shelf  shelfIn  `json:"shelf"`
	Pages  int      `json:"pages,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Author string   `json:"author"`
}

func newCatalogueRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	if err := Add(registry, Spec{Name: "catalogue_book", Effect: ReadWrite}, func(_ context.Context, in cataloguedIn) (cataloguedIn, error) {
		return in, nil
	}); err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestValidationMessagesNameTheArgumentInPlainWords(t *testing.T) {
	registry := newCatalogueRegistry(t)

	for arguments, want := range map[string]string{
		`{"shelf":{"title":"Ferns"},"author":"Mira","pages":"many"}`:   `the argument pages is invalid: many has type "string", want "integer"`,
		`{"shelf":{"title":"Ferns"},"author":"Mira","tags":[3]}`:       `the argument tags is invalid: 3 has type "integer", want "string"`,
		`{"shelf":{"title":"Ferns"}}`:                                  `the argument author is missing`,
		`{}`:                                                           `the arguments shelf, author are missing`,
		`{"shelf":{},"author":"Mira"}`:                                 `the argument shelf.title is missing`,
		`{"shelf":{"title":"Ferns","row":2},"author":"Mira"}`:          `the argument shelf.row is not known`,
		`{"shelf":{"title":"Ferns"},"author":"Mira","colour":"green"}`: `the argument colour is not known`,
	} {
		res := invoke(t, registry, "catalogue_book", []byte(arguments))
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != want {
			t.Errorf("the arguments %s were answered %+v, want the error %q", arguments, res.GetResult(), want)
		}
	}
}

func TestReadPropertyPathReadsThePropertyNamesOfAPointer(t *testing.T) {
	for pointer, want := range map[string]string{
		"root":                               "",
		"/properties/shelf/properties/title": "shelf.title",
		"/properties/properties":             "properties",
		"/properties/a~1b/items":             "a/b",
	} {
		got := ""
		for i, name := range readPropertyPath(pointer) {
			if i > 0 {
				got += "."
			}
			got += name
		}
		if got != want {
			t.Errorf("readPropertyPath(%q) = %q, want %q", pointer, got, want)
		}
	}
}

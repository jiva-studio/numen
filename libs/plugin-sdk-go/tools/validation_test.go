package tools

import (
	"context"
	"testing"
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
		`{"location":{"title":"Ferns"},"author":"Mira","pages":"many"}`:   `the argument pages is invalid: many has type "string", want "integer"`,
		`{"location":{"title":"Ferns"},"author":"Mira","tags":[3]}`:       `the argument tags is invalid: 3 has type "integer", want "string"`,
		`{"location":{"title":"Ferns"}}`:                                  `the argument author is missing`,
		`{}`:                                                              `the arguments location, author are missing`,
		`{"location":{},"author":"Mira"}`:                                 `the argument location.title is missing`,
		`{"location":{"title":"Ferns","row":2},"author":"Mira"}`:          `the argument location.row is not known`,
		`{"location":{"title":"Ferns"},"author":"Mira","colour":"green"}`: `the argument colour is not known`,
	} {
		res := invoke(t, registry, "book_create", []byte(arguments))
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != want {
			t.Errorf("the arguments %s were answered %+v, want the error %q", arguments, res.GetResult(), want)
		}
	}
}

func TestReadPropertyPathReadsThePropertyNamesOfAPointer(t *testing.T) {
	for pointer, want := range map[string]string{
		"root":                               "",
		"/properties/location/properties/title": "location.title",
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

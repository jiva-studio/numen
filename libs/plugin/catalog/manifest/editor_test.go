package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const editorHead = `{"id":"md.example.todos","name":"A","version":"1.0.0","frontend":{},"contributes":{"editor":{"extensions":`

func TestReadManifest_EditorExtensions(t *testing.T) {
	cases := map[string]struct {
		extensions string
		want       error
	}{
		"one extension":          {`[{"id":"tasks","title":"Tasks","description":"Draws checkboxes"}]}}}`, nil},
		"no title":               {`[{"id":"tasks","title":"","description":"d"}]}}}`, manifest.ErrInvalidContribution},
		"no description":         {`[{"id":"tasks","title":"T","description":" "}]}}}`, manifest.ErrInvalidContribution},
		"an id not in camelCase": {`[{"id":"Task-list","title":"T","description":"d"}]}}}`, manifest.ErrInvalidContribution},
		"an id twice":            {`[{"id":"tasks","title":"T","description":"d"},{"id":"tasks","title":"U","description":"e"}]}}}`, manifest.ErrInvalidContribution},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(editorHead + one.extensions))

			if !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

const editorBackendOnly = `{"id":"md.example.todos","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},` +
	`"contributes":{"editor":{"extensions":[{"id":"tasks","title":"T","description":"d"}]}}}`

func TestReadManifest_EditorExtensionsNeedAFrontend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(editorBackendOnly))

	if !errors.Is(err, manifest.ErrInvalidContribution) {
		t.Errorf("error = %v, want %v", err, manifest.ErrInvalidContribution)
	}
}

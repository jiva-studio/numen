package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const dropHandlerHead = `{"id":"md.example.links","name":"A","version":"1.0.0","frontend":{},"contributes":{"dropHandlers":`

func TestReadManifest_DropHandlers(t *testing.T) {
	cases := map[string]struct {
		handlers string
		want     error
	}{
		"one handler with one type":     {`[{"types":["text/uri-list"]}]}}`, nil},
		"two handlers with other types": {`[{"types":["text/uri-list"]},{"types":["text/plain"]}]}}`, nil},
		"a handler with no types":       {`[{"types":[]}]}}`, manifest.ErrInvalidContribution},
		"an empty type":                 {`[{"types":[""]}]}}`, manifest.ErrInvalidContribution},
		"a type with space around it":   {`[{"types":[" text/plain"]}]}}`, manifest.ErrInvalidContribution},
		"a type twice in one handler":   {`[{"types":["text/plain","text/plain"]}]}}`, manifest.ErrInvalidContribution},
		"a type in two handlers":        {`[{"types":["text/plain"]},{"types":["text/plain"]}]}}`, manifest.ErrInvalidContribution},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(dropHandlerHead + one.handlers))

			if !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

func TestReadManifest_DropHandlersNeedAFrontend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"md.example.links","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"contributes":{"dropHandlers":[{"types":["text/plain"]}]}}`))

	if !errors.Is(err, manifest.ErrInvalidContribution) {
		t.Errorf("error = %v, want %v", err, manifest.ErrInvalidContribution)
	}
}

func TestReadManifest_DropHandlersAreKept(t *testing.T) {
	read, err := manifestjson.Read(strings.NewReader(dropHandlerHead + `[{"types":["text/uri-list","text/html"]}]}}`))

	if err != nil {
		t.Fatal(err)
	}
	got := read.Contributes.DropHandlers
	if len(got) != 1 || strings.Join(got[0].Types, ",") != "text/uri-list,text/html" {
		t.Errorf("drop handlers = %v", got)
	}
}

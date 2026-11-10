package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const viewCapabilityHead = `{"id":"md.example.chat","name":"A","version":"1.0.0","frontend":{},"contributes":{"viewCapabilities":`

func TestReadManifest_ViewCapabilities(t *testing.T) {
	cases := map[string]struct {
		capabilities string
		want         error
	}{
		"a kind the application defines":             {`["conversation"]}}`, nil},
		"all three kinds of the application":         {`["conversation","note-editor","vault-summary"]}}`, nil},
		"a kind named after the plugin":              {`["md.example.chat.thread"]}}`, nil},
		"a kind of no list":                          {`["graph"]}}`, manifest.ErrInvalidContribution},
		"a kind named after another plugin":          {`["md.example.other.thread"]}}`, manifest.ErrInvalidContribution},
		"the id of the plugin alone":                 {`["md.example.chat."]}}`, manifest.ErrInvalidContribution},
		"a kind that is not a plain name":            {`["md.example.chat.Thread Pane"]}}`, manifest.ErrInvalidContribution},
		"the same kind twice":                        {`["conversation","conversation"]}}`, manifest.ErrInvalidContribution},
		"a kind the application defines in capitals": {`["Conversation"]}}`, manifest.ErrInvalidContribution},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(viewCapabilityHead + one.capabilities))

			if !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

func TestReadManifest_ViewCapabilitiesNeedAFrontend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"md.example.chat","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"contributes":{"viewCapabilities":["conversation"]}}`))

	if !errors.Is(err, manifest.ErrInvalidContribution) {
		t.Errorf("error = %v, want %v", err, manifest.ErrInvalidContribution)
	}
}

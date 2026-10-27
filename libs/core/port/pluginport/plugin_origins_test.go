package pluginport_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const originManifestHead = `{"id":"a.b","name":"A","version":"1.0.0","origins":`

func TestReadManifestKeepsDeclaredOrigins(t *testing.T) {
	m, err := pluginport.ReadManifest(strings.NewReader(originManifestHead + `["https://tiles.example.org","https://example.org:8443"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Origins, []string{"https://tiles.example.org", "https://example.org:8443"}) {
		t.Errorf("the origins came out as %q", m.Origins)
	}
}

func TestReadManifestRejectsAnOriginOutsideHTTPS(t *testing.T) {
	for _, origin := range []string{
		`"http://example.org"`,
		`"https://example.org/path"`,
		`"https://example.org?x"`,
		`"https://user@example.org"`,
		`"https://example.org; script-src *"`,
		`"https://*.example.org 'unsafe-eval'"`,
		`"example.org"`,
		`"*"`,
	} {
		_, err := pluginport.ReadManifest(strings.NewReader(originManifestHead + `[` + origin + `]}`))
		if !errors.Is(err, pluginport.ErrInvalidOrigin) {
			t.Errorf("%s was read with %v", origin, err)
		}
	}
}

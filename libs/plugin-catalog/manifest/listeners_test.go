package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const listenersManifestHead = `{"id":"org.example.listeners","name":"A","version":"1.0.0",` +
	`"backend": {"entrypoint": "bin/listeners"}`

func TestReadManifestReadsTheListenersOfThePlugin(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(listenersManifestHead + `,
		"contributes": {"listeners": [
			{"name": "mcp", "port": 3333, "protocol": "tcp"},
			{"name": "debug-console", "port": 65535, "protocol": "tcp"}
		]}}`))
	if err != nil {
		t.Fatal(err)
	}

	want := []manifest.Listener{
		{Name: "mcp", Port: 3333, Protocol: "tcp"},
		{Name: "debug-console", Port: 65535, Protocol: "tcp"},
	}
	if !slices.Equal(m.Contributes.Listeners, want) {
		t.Errorf("listeners = %+v, want %+v", m.Contributes.Listeners, want)
	}
}

func TestReadManifestReadsNoListenersWhenNoneAreDeclared(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(listenersManifestHead + `}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(m.Contributes.Listeners) != 0 {
		t.Errorf("listeners = %+v, want none", m.Contributes.Listeners)
	}
}

func TestReadManifestAcceptsTheLowestAndHighestListenerPort(t *testing.T) {
	for _, port := range []string{"1024", "65535"} {
		_, err := manifestjson.Read(strings.NewReader(listenersManifestHead +
			`, "contributes": {"listeners": [{"name": "mcp", "port": ` + port + `, "protocol": "tcp"}]}}`))
		if err != nil {
			t.Errorf("port %s: error = %v, want none", port, err)
		}
	}
}

func TestReadManifestRejectsAListenerTheHostCannotBind(t *testing.T) {
	cases := map[string]string{
		"a name declared twice":      `[{"name":"mcp","port":3333,"protocol":"tcp"},{"name":"mcp","port":3334,"protocol":"tcp"}]`,
		"a port below 1024":          `[{"name":"mcp","port":1023,"protocol":"tcp"}]`,
		"port 0":                     `[{"name":"mcp","port":0,"protocol":"tcp"}]`,
		"a port above 65535":         `[{"name":"mcp","port":65536,"protocol":"tcp"}]`,
		"protocol udp":               `[{"name":"mcp","port":3333,"protocol":"udp"}]`,
		"no protocol":                `[{"name":"mcp","port":3333}]`,
		"protocol in capitals":       `[{"name":"mcp","port":3333,"protocol":"TCP"}]`,
		"an empty name":              `[{"name":"","port":3333,"protocol":"tcp"}]`,
		"a name in capitals":         `[{"name":"Mcp","port":3333,"protocol":"tcp"}]`,
		"a name with a space":        `[{"name":"mcp server","port":3333,"protocol":"tcp"}]`,
		"a name with a dot":          `[{"name":"mcp.http","port":3333,"protocol":"tcp"}]`,
		"a name starting with digit": `[{"name":"3mcp","port":3333,"protocol":"tcp"}]`,
		"a name ending in a hyphen":  `[{"name":"mcp-","port":3333,"protocol":"tcp"}]`,
	}
	for name, listeners := range cases {
		_, err := manifestjson.Read(strings.NewReader(listenersManifestHead +
			`, "contributes": {"listeners": ` + listeners + `}}`))

		if !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: error = %v, want an invalid contribution", name, err)
		}
	}
}

func TestReadManifestRejectsAListenerOnAPluginWithoutABackend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"org.example.listeners","name":"A","version":"1.0.0",
		"frontend": {}, "contributes": {"listeners": [{"name": "mcp", "port": 3333, "protocol": "tcp"}]}}`))

	if !errors.Is(err, manifest.ErrInvalidContribution) {
		t.Errorf("error = %v, want an invalid contribution", err)
	}
}

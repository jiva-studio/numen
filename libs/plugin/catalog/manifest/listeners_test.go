package manifest_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const listenersManifestHead = `{"id":"org.example.listeners","name":"A","version":"1.0.0",` +
	`"backend": {"entrypoint": "bin/listeners"},` +
	`"settings": [` +
	`{"key": "port", "type": "number", "title": "Port", "default": 4100, "min": 1024, "max": 65535, "step": 1},` +
	`{"key": "console-port", "type": "number", "title": "Console port", "default": 4101},` +
	`{"key": "host", "type": "string", "title": "Host", "default": "relay.example"}]`

func TestReadManifestReadsTheListenersOfThePlugin(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(listenersManifestHead + `,
		"contributes": {"listeners": [
			{"name": "mcp", "portSetting": "port", "protocol": "tcp"},
			{"name": "debug-console", "portSetting": "console-port", "protocol": "tcp"},
			{"name": "metrics", "port": 4200, "protocol": "tcp"}
		]}}`))
	if err != nil {
		t.Fatal(err)
	}

	want := []manifest.Listener{
		{Name: "mcp", PortSetting: "port", Protocol: "tcp"},
		{Name: "debug-console", PortSetting: "console-port", Protocol: "tcp"},
		{Name: "metrics", Port: 4200, Protocol: "tcp"},
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

func TestReadManifestRejectsAListenerTheHostCannotBind(t *testing.T) {
	cases := map[string]string{
		"a name declared twice":         `[{"name":"mcp","portSetting":"port","protocol":"tcp"},{"name":"mcp","portSetting":"console-port","protocol":"tcp"}]`,
		"protocol udp":                  `[{"name":"mcp","portSetting":"port","protocol":"udp"}]`,
		"no protocol":                   `[{"name":"mcp","portSetting":"port"}]`,
		"protocol in capitals":          `[{"name":"mcp","portSetting":"port","protocol":"TCP"}]`,
		"an empty name":                 `[{"name":"","portSetting":"port","protocol":"tcp"}]`,
		"a name in capitals":            `[{"name":"Mcp","portSetting":"port","protocol":"tcp"}]`,
		"a name with a space":           `[{"name":"mcp server","portSetting":"port","protocol":"tcp"}]`,
		"a name with a dot":             `[{"name":"mcp.http","portSetting":"port","protocol":"tcp"}]`,
		"a name starting with digit":    `[{"name":"3mcp","portSetting":"port","protocol":"tcp"}]`,
		"a name ending in a hyphen":     `[{"name":"mcp-","portSetting":"port","protocol":"tcp"}]`,
		"neither port nor port setting": `[{"name":"mcp","protocol":"tcp"}]`,
		"both port and port setting":    `[{"name":"mcp","port":4200,"portSetting":"port","protocol":"tcp"}]`,
		"a port below 1024":             `[{"name":"mcp","port":1023,"protocol":"tcp"}]`,
		"a port above 65535":            `[{"name":"mcp","port":65536,"protocol":"tcp"}]`,
		"a negative port":               `[{"name":"mcp","port":-4200,"protocol":"tcp"}]`,
		"a port setting not declared":   `[{"name":"mcp","portSetting":"listen-port","protocol":"tcp"}]`,
		"a port setting of type string": `[{"name":"mcp","portSetting":"host","protocol":"tcp"}]`,
	}
	for name, listeners := range cases {
		_, err := manifestjson.Read(strings.NewReader(listenersManifestHead +
			`, "contributes": {"listeners": ` + listeners + `}}`))

		if !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: error = %v, want an invalid contribution", name, err)
		}
	}
}

func TestReadManifestAcceptsTheLowestAndHighestFixedPort(t *testing.T) {
	if manifest.MinListenerPort != 1024 || manifest.MaxListenerPort != 65535 {
		t.Errorf("listener ports %d..%d, want 1024..65535", manifest.MinListenerPort, manifest.MaxListenerPort)
	}
	for _, port := range []string{strconv.Itoa(manifest.MinListenerPort), strconv.Itoa(manifest.MaxListenerPort)} {
		_, err := manifestjson.Read(strings.NewReader(listenersManifestHead +
			`, "contributes": {"listeners": [{"name": "mcp", "port": ` + port + `, "protocol": "tcp"}]}}`))
		if err != nil {
			t.Errorf("port %s: error = %v, want none", port, err)
		}
	}
}

func TestReadManifestRejectsAListenerOnAPluginWithoutABackend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"org.example.listeners","name":"A","version":"1.0.0",
		"frontend": {},
		"settings": [{"key": "port", "type": "number", "title": "Port", "default": 4100}],
		"contributes": {"listeners": [{"name": "mcp", "portSetting": "port", "protocol": "tcp"}]}}`))

	if !errors.Is(err, manifest.ErrInvalidContribution) {
		t.Errorf("error = %v, want an invalid contribution", err)
	}
}

package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const connectionHead = `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"}`

const openRouterType = `{"id":"openrouter","title":"OpenRouter","description":"One key for many models.",
	"operations":["complete","embed"],
	"fields":[
		{"key":"baseUrl","type":"string","title":"Address","isAddress":true,"fixedValue":"https://openrouter.ai/api/v1"},
		{"key":"apiKey","type":"secret","title":"API key","required":true},
		{"key":"parallelRequests","type":"number","title":"Parallel requests","default":4,"min":1,"max":32,"step":1}]}`

func readConnectionTypes(types string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(connectionHead + `,"contributes":{"connectionTypes":[` + types + `]}}`))
}

func TestAManifestDeclaresTheConnectionTypesOfThePlugin(t *testing.T) {
	m, err := readConnectionTypes(openRouterType)
	if err != nil {
		t.Fatal(err)
	}

	declared, ok := m.Contributes.GetConnectionType("openrouter")
	if !ok || declared.Title != "OpenRouter" || !declared.HasOperation("embed") || declared.HasOperation("chat") {
		t.Fatalf("the openrouter type = %+v, %v", declared, ok)
	}
	address, _ := declared.GetField("baseUrl")
	if !address.IsAddress || address.FixedValue == nil || *address.FixedValue != "https://openrouter.ai/api/v1" {
		t.Errorf("the address field = %+v", address)
	}
	key, _ := declared.GetField("apiKey")
	if key.Type != manifest.ConnectionFieldSecret || !key.IsRequired {
		t.Errorf("the key field = %+v", key)
	}
	if _, ok := m.Contributes.GetConnectionType("missing"); ok {
		t.Error("a type the manifest does not declare was found")
	}
}

func TestAConnectionTypeIDJoinsThePluginIDAndTheLocalID(t *testing.T) {
	id := manifest.GetConnectionTypeID("md.numen.connections.llm", "openrouter")
	if id != "md.numen.connections.llm/openrouter" {
		t.Fatalf("id = %q", id)
	}
	plugin, local, ok := manifest.SplitConnectionTypeID(id)
	if !ok || plugin != "md.numen.connections.llm" || local != "openrouter" {
		t.Errorf("split = %q, %q, %v", plugin, local, ok)
	}
	for _, bad := range []string{"", "openrouter", "/openrouter", "a.b/", "a.b/c/d"} {
		if _, _, ok := manifest.SplitConnectionTypeID(bad); ok {
			t.Errorf("%q was split", bad)
		}
	}
}

func TestAConnectionTypeThatTheApplicationCannotUseIsRefused(t *testing.T) {
	const head = `{"id":"x","title":"T","operations":["complete"]`
	cases := map[string]string{
		"an id in capitals":              `{"id":"OpenRouter","title":"T","operations":["complete"]}`,
		"an id with a slash":             `{"id":"a/b","title":"T","operations":["complete"]}`,
		"a type declared twice":          head + `},` + head + `}`,
		"no title":                       `{"id":"x","title":" ","operations":["complete"]}`,
		"no operation":                   `{"id":"x","title":"T","operations":[]}`,
		"an operation in capitals":       `{"id":"x","title":"T","operations":["Complete"]}`,
		"an operation twice":             `{"id":"x","title":"T","operations":["complete","complete"]}`,
		"a reserved operation":           `{"id":"x","title":"T","operations":["submit"]}`,
		"a field key with a dash":        head + `,"fields":[{"key":"base-url","type":"string","title":"A"}]}`,
		"a field key twice":              head + `,"fields":[{"key":"a","type":"string","title":"A"},{"key":"a","type":"string","title":"B"}]}`,
		"a reserved field key":           head + `,"fields":[{"key":"name","type":"string","title":"A"}]}`,
		"an unknown field type":          head + `,"fields":[{"key":"a","type":"range","title":"A"}]}`,
		"an enum with no options":        head + `,"fields":[{"key":"a","type":"enum","title":"A"}]}`,
		"a secret with a default":        head + `,"fields":[{"key":"a","type":"secret","title":"A","default":"k"}]}`,
		"a secret with a fixed value":    head + `,"fields":[{"key":"a","type":"secret","title":"A","fixedValue":"k"}]}`,
		"a secret that is the address":   head + `,"fields":[{"key":"a","type":"secret","title":"A","isAddress":true}]}`,
		"a secret with options":          head + `,"fields":[{"key":"a","type":"secret","title":"A","options":["x"]}]}`,
		"a fixed value on a number":      head + `,"fields":[{"key":"a","type":"number","title":"A","fixedValue":"1"}]}`,
		"a required fixed value":         head + `,"fields":[{"key":"a","type":"string","title":"A","fixedValue":"u","required":true}]}`,
		"a fixed value with a default":   head + `,"fields":[{"key":"a","type":"string","title":"A","fixedValue":"u","default":"v"}]}`,
		"an address that is a number":    head + `,"fields":[{"key":"a","type":"number","title":"A","isAddress":true}]}`,
		"two addresses":                  head + `,"fields":[{"key":"a","type":"string","title":"A","isAddress":true},{"key":"b","type":"string","title":"B","isAddress":true}]}`,
		"parallel requests as a string":  head + `,"fields":[{"key":"parallelRequests","type":"string","title":"P"}]}`,
		"parallel requests below one":    head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":0,"step":1}]}`,
		"parallel requests with a step":  head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":1,"step":0.5}]}`,
		"parallel requests default low":  head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":2,"step":1,"default":1}]}`,
		"parallel requests default part": head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":1,"step":1,"default":1.5}]}`,
	}
	for name, types := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := readConnectionTypes(types); !errors.Is(err, manifest.ErrInvalidContribution) {
				t.Errorf("err = %v, want ErrInvalidContribution", err)
			}
		})
	}
}

func TestAPluginThatDeclaresConnectionTypesDeclaresABackend(t *testing.T) {
	text := `{"id":"a.b","name":"A","version":"1.0.0","contributes":{"connectionTypes":[
		{"id":"x","title":"T","operations":["complete"]}]}}`
	if _, err := manifestjson.Read(strings.NewReader(text)); !errors.Is(err, manifest.ErrNoBackend) {
		t.Errorf("err = %v, want ErrNoBackend", err)
	}
}

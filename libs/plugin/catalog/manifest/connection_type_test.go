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
	"operations":["llm.complete","llm.embed","llm.models"],
	"rowFields":[{"key":"model","type":"string","title":"Model","required":true,"optionsFrom":"llm.models"}],
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
	if !ok || declared.Title != "OpenRouter" || !declared.HasOperation("llm.embed") || declared.HasOperation("llm.chat") {
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
	model, _ := declared.GetRowField("model")
	if model.OptionsFrom != "llm.models" || !model.IsRequired {
		t.Errorf("the model row field = %+v", model)
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
	const head = `{"id":"x","title":"T","operations":["llm.complete"]`
	cases := map[string]string{
		"an id in capitals":              `{"id":"OpenRouter","title":"T","operations":["llm.complete"]}`,
		"an id with a slash":             `{"id":"a/b","title":"T","operations":["llm.complete"]}`,
		"a type declared twice":          head + `},` + head + `}`,
		"no title":                       `{"id":"x","title":" ","operations":["llm.complete"]}`,
		"no operation":                   `{"id":"x","title":"T","operations":[]}`,
		"an operation in capitals":       `{"id":"x","title":"T","operations":["Complete"]}`,
		"an operation twice":             `{"id":"x","title":"T","operations":["llm.complete","llm.complete"]}`,
		"a field key with a dash":        head + `,"fields":[{"key":"base-url","type":"string","title":"A"}]}`,
		"a row field of numbers":         head + `,"rowFields":[{"key":"a","type":"number","title":"A"}]}`,
		"a row field keyed connection":   head + `,"rowFields":[{"key":"connection","type":"string","title":"A"}]}`,
		"options from an unserved call":  head + `,"rowFields":[{"key":"model","type":"string","title":"Model","optionsFrom":"llm.models"}]}`,
		"options from on a field":        `{"id":"x","title":"T","operations":["llm.complete","llm.models"],"fields":[{"key":"a","type":"string","title":"A","optionsFrom":"llm.models"}]}`,
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
		"built in with a required field": head + `,"builtIn":true,"fields":[{"key":"a","type":"secret","title":"A","required":true}]}`,
	}
	for name, types := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := readConnectionTypes(types); !errors.Is(err, manifest.ErrInvalidContribution) {
				t.Errorf("err = %v, want ErrInvalidContribution", err)
			}
		})
	}
}

func TestATypeWithNoRequiredFieldIsBuiltIn(t *testing.T) {
	m, err := readConnectionTypes(`{"id":"cli","title":"CLI","operations":["acp.task"],"builtIn":true,
		"rowFields":[{"key":"model","type":"string","title":"Model"}],
		"fields":[
			{"key":"baseUrl","type":"string","title":"Address","isAddress":true,"fixedValue":"https://cli.example"},
			{"key":"parallelRequests","type":"number","title":"Parallel requests","default":1,"min":1,"step":1}]},` +
		openRouterType)
	if err != nil {
		t.Fatal(err)
	}

	cli, _ := m.Contributes.GetConnectionType("cli")
	openRouter, _ := m.Contributes.GetConnectionType("openrouter")
	if !cli.IsBuiltIn {
		t.Error("the type that declares builtIn is not built in")
	}
	if openRouter.IsBuiltIn {
		t.Error("the type that does not declare builtIn is built in")
	}
}

func TestAPluginThatDeclaresConnectionTypesDeclaresABackend(t *testing.T) {
	text := `{"id":"a.b","name":"A","version":"1.0.0","contributes":{"connectionTypes":[
		{"id":"x","title":"T","operations":["llm.complete"]}]}}`
	if _, err := manifestjson.Read(strings.NewReader(text)); !errors.Is(err, manifest.ErrNoBackend) {
		t.Errorf("err = %v, want ErrNoBackend", err)
	}
}

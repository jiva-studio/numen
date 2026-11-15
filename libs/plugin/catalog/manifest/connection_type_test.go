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
	"rowFields":[
		{"key":"model","type":"string","title":"Model","required":true,"optionsFrom":"llm.models"},
		{"key":"tier","type":"enum","title":"Tier","options":["free","paid"]},
		{"key":"voice","type":"enum","title":"Voice","optionsFrom":"llm.models"}
	],
	"fields":[
		{"key":"baseUrl","type":"string","title":"Address","isAddress":true,"fixedValue":"https://openrouter.ai/api/v1"},
		{"key":"apiKey","type":"secret","title":"API key","required":true},
		{"key":"mode","type":"enum","title":"Mode","options":["fast","accurate"]},
		{"key":"parallelRequests","type":"number","title":"Parallel requests","default":1,"min":1,"max":32,"step":1}]}`

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
	address, ok := declared.GetField("baseUrl")
	if !ok || !address.IsAddress || address.FixedValue == nil || *address.FixedValue != "https://openrouter.ai/api/v1" {
		t.Errorf("the address field = %+v, %v", address, ok)
	}
	if _, ok := declared.GetField("missing"); ok {
		t.Error("a field the type does not declare was found")
	}
	key, ok := declared.GetField("apiKey")
	if !ok || key.Type != manifest.ConnectionFieldSecret || !key.IsRequired {
		t.Errorf("the key field = %+v, %v", key, ok)
	}
	model, ok := declared.GetRowField("model")
	if !ok || model.OptionsFrom != "llm.models" || !model.IsRequired {
		t.Errorf("the model row field = %+v, %v", model, ok)
	}
	if _, ok := declared.GetRowField("missing"); ok {
		t.Error("a row field the type does not declare was found")
	}
	if _, ok := m.Contributes.GetConnectionType("missing"); ok {
		t.Error("a type the manifest does not declare was found")
	}

	noDefaultParallel := `{"id":"local","title":"Local","operations":["llm.complete"],
		"fields":[{"key":"parallelRequests","type":"number","title":"Parallel","min":1,"step":1}]}`
	if _, err := readConnectionTypes(noDefaultParallel); err != nil {
		t.Errorf("parallelRequests without default failed: %v", err)
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
	cases := map[string]struct {
		types string
		want  string
	}{
		"an id in capitals":             {`{"id":"OpenRouter","title":"T","operations":["llm.complete"]}`, "is not lowercase words"},
		"an id with a slash":            {`{"id":"a/b","title":"T","operations":["llm.complete"]}`, "is not lowercase words"},
		"a type declared twice":         {head + `},` + head + `}`, "is declared twice"},
		"no title":                      {`{"id":"x","title":" ","operations":["llm.complete"]}`, "has no title"},
		"no operation":                  {`{"id":"x","title":"T","operations":[]}`, "declares no operation"},
		"an operation in capitals":      {`{"id":"x","title":"T","operations":["Complete"]}`, "is not lowercase words joined by dots"},
		"an operation twice":            {`{"id":"x","title":"T","operations":["llm.complete","llm.complete"]}`, "is listed twice"},
		"a field key with a dash":       {head + `,"fields":[{"key":"base-url","type":"string","title":"A"}]}`, "field key"},
		"a row field key with a dash":   {head + `,"rowFields":[{"key":"base-url","type":"string","title":"A"}]}`, "row field key"},
		"a row field of numbers":        {head + `,"rowFields":[{"key":"a","type":"number","title":"A"}]}`, "is a row field and is not a string or an enum"},
		"a row field enum without list": {head + `,"rowFields":[{"key":"a","type":"enum","title":"A"}]}`, "is an enum with neither options nor optionsFrom"},
		"a row field keyed connection":  {head + `,"rowFields":[{"key":"connection","type":"string","title":"A"}]}`, "is reserved"},
		"a row field declared twice":    {head + `,"rowFields":[{"key":"a","type":"string","title":"A"},{"key":"a","type":"string","title":"B"}]}`, "row field \"a\" is declared twice"},
		"a row field with fixed value":  {head + `,"rowFields":[{"key":"a","type":"string","title":"A","fixedValue":"u"}]}`, "is a row field and has a fixed value or the address mark"},
		"a row field with address mark": {head + `,"rowFields":[{"key":"a","type":"string","title":"A","isAddress":true}]}`, "is a row field and has a fixed value or the address mark"},
		"options from an unserved call": {head + `,"rowFields":[{"key":"model","type":"string","title":"Model","optionsFrom":"llm.models"}]}`, "which the type does not serve"},
		"options from on a field": {
			`{"id":"x","title":"T","operations":["llm.complete","llm.models"],"fields":[{"key":"a","type":"string","title":"A","optionsFrom":"llm.models"}]}`,
			"names optionsFrom and is no row field",
		},
		"a field key twice":            {head + `,"fields":[{"key":"a","type":"string","title":"A"},{"key":"a","type":"string","title":"B"}]}`, "field \"a\" is declared twice"},
		"a reserved field key":         {head + `,"fields":[{"key":"name","type":"string","title":"A"}]}`, "is reserved"},
		"an unknown field type":        {head + `,"fields":[{"key":"a","type":"range","title":"A"}]}`, "which is not a field type"},
		"an enum with no options":      {head + `,"fields":[{"key":"a","type":"enum","title":"A"}]}`, "is an enum with no options"},
		"a secret with a default":      {head + `,"fields":[{"key":"a","type":"secret","title":"A","default":"k"}]}`, "is a secret and has a default"},
		"a secret with a fixed value":  {head + `,"fields":[{"key":"a","type":"secret","title":"A","fixedValue":"k"}]}`, "is a secret and has a default"},
		"a secret that is the address": {head + `,"fields":[{"key":"a","type":"secret","title":"A","isAddress":true}]}`, "is a secret and has a default"},
		"a secret with options":        {head + `,"fields":[{"key":"a","type":"secret","title":"A","options":["x"]}]}`, "is a secret and has a default"},
		"a fixed value on a number":    {head + `,"fields":[{"key":"a","type":"number","title":"A","fixedValue":"1"}]}`, "has a fixed value"},
		"a required fixed value":       {head + `,"fields":[{"key":"a","type":"string","title":"A","fixedValue":"u","required":true}]}`, "has a fixed value"},
		"a fixed value with a default": {head + `,"fields":[{"key":"a","type":"string","title":"A","fixedValue":"u","default":"v"}]}`, "has a fixed value"},
		"an address that is a number":  {head + `,"fields":[{"key":"a","type":"number","title":"A","isAddress":true}]}`, "is the address and is not a string"},
		"two addresses": {
			head + `,"fields":[{"key":"a","type":"string","title":"A","isAddress":true},{"key":"b","type":"string","title":"B","isAddress":true}]}`,
			"marks 2 fields as the address",
		},
		"parallel requests as a string":  {head + `,"fields":[{"key":"parallelRequests","type":"string","title":"P","min":1,"step":1}]}`, "parallelRequests is a number"},
		"parallel requests without min":  {head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","step":1}]}`, "parallelRequests is a number"},
		"parallel requests without step": {head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":1}]}`, "parallelRequests is a number"},
		"parallel requests below one":    {head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":0,"step":1}]}`, "parallelRequests is a number"},
		"parallel requests with a step":  {head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":1,"step":0.5}]}`, "parallelRequests is a number"},
		"parallel requests default text": {head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":1,"step":1,"default":"2"}]}`, "the default of parallelRequests"},
		"parallel requests default low":  {head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":2,"step":1,"default":1}]}`, "the default of parallelRequests"},
		"parallel requests default part": {head + `,"fields":[{"key":"parallelRequests","type":"number","title":"P","min":1,"step":1,"default":1.5}]}`, "the default of parallelRequests"},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := readConnectionTypes(one.types)
			if !errors.Is(err, manifest.ErrInvalidContribution) || !strings.Contains(err.Error(), one.want) {
				t.Errorf("err = %v, want ErrInvalidContribution containing %q", err, one.want)
			}
		})
	}
}

func TestAPluginThatDeclaresConnectionTypesDeclaresABackend(t *testing.T) {
	text := `{"id":"a.b","name":"A","version":"1.0.0","contributes":{"connectionTypes":[
		{"id":"x","title":"T","operations":["llm.complete"]}]}}`
	if _, err := manifestjson.Read(strings.NewReader(text)); !errors.Is(err, manifest.ErrNoBackend) {
		t.Errorf("err = %v, want ErrNoBackend", err)
	}
}

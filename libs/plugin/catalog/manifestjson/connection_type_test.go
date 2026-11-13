package manifestjson_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestAConnectionTypeIsReadWithItsFields(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{"id":"a.b","contributes":{"connectionTypes":[
		{"id":"local","title":"Local","description":"d","operations":["llm.complete","llm.embed"],"fields":[
			{"key":"baseUrl","type":"string","title":"Address","description":"where","isAddress":true,"required":true,"default":"http://localhost:11434/v1"},
			{"key":"mode","type":"enum","title":"Mode","options":["a",{"value":"b","title":"B"}]},
			{"key":"parallelRequests","type":"number","title":"Parallel","default":2,"min":1,"max":8,"step":1}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(m.Contributes.ConnectionTypes) != 1 {
		t.Fatalf("types = %+v", m.Contributes.ConnectionTypes)
	}
	one := m.Contributes.ConnectionTypes[0]
	if one.ID != "local" || one.Title != "Local" || one.Description != "d" || !slices.Equal(one.Operations, []string{"llm.complete", "llm.embed"}) {
		t.Errorf("type = %+v", one)
	}
	address, mode, parallel := one.Fields[0], one.Fields[1], one.Fields[2]
	if address.Key != "baseUrl" || address.Description != "where" || !address.IsAddress || !address.IsRequired || address.Default != "http://localhost:11434/v1" {
		t.Errorf("address = %+v", address)
	}
	if len(mode.Options) != 2 || mode.Options[1].Value != "b" || mode.Options[1].Title != "B" {
		t.Errorf("mode = %+v", mode)
	}
	if parallel.Default != float64(2) || *parallel.Min != 1 || *parallel.Max != 8 || *parallel.Step != 1 {
		t.Errorf("parallel = %+v", parallel)
	}
}

func TestAnUnknownKeyInAConnectionTypeIsAWarning(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{"id":"a.b","contributes":{"connectionTypes":[
		{"id":"x","extra":1,"fields":[{"key":"k","other":2}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{`unknown key "contributes.connectionTypes[0].extra"`, `unknown key "contributes.connectionTypes[0].fields[0].other"`} {
		if !slices.Contains(m.Warnings, want) {
			t.Errorf("warnings = %v, want %s", m.Warnings, want)
		}
	}
}

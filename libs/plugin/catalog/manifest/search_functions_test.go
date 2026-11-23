package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const rhymesFunction = `{"name":"rhymes","args":[{"name":"word","type":"string"}],"level":"document",` +
	`"description":"notes that rhyme with a word","example":"rhymes(\"moon\")"}`

func readSearchFunctions(backend, functions string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",` + backend +
		`"contributes":{"searchFunctions":` + functions + `}}`))
}

func TestAManifestDeclaresTheSearchFunctionsItsBackendAnswers(t *testing.T) {
	m, err := readSearchFunctions(`"backend":{"entrypoint":"bin/a"},`, `{"version":1,"functions":[`+rhymesFunction+`]}`)
	if err != nil {
		t.Fatal(err)
	}

	got := m.Contributes.SearchFunctions
	if got.Version != 1 || len(got.Functions) != 1 {
		t.Fatalf("search functions = %+v", got)
	}
	f := got.Functions[0]
	if f.Name != "rhymes" || f.Level != manifest.SearchFunctionLevelDocument || f.Description != "notes that rhyme with a word" ||
		f.Example != `rhymes("moon")` || len(f.Args) != 1 || f.Args[0] != (manifest.SearchFunctionArg{Name: "word", Type: manifest.SearchFunctionArgString}) {
		t.Errorf("function = %+v", f)
	}
}

func TestASearchFunctionDeclaredWronglyIsRefusedNamingTheField(t *testing.T) {
	backend := `"backend":{"entrypoint":"bin/a"},`
	for name, c := range map[string]struct{ backend, functions, field string }{
		"another schema version":      {backend, `{"version":2,"functions":[` + rhymesFunction + `]}`, "version 2"},
		"a name with a dash":          {backend, `{"version":1,"functions":[` + strings.Replace(rhymesFunction, `"rhymes"`, `"rhy-mes"`, 1) + `]}`, "name"},
		"a name declared twice":       {backend, `{"version":1,"functions":[` + rhymesFunction + `,` + rhymesFunction + `]}`, "twice"},
		"an unknown arg type":         {backend, `{"version":1,"functions":[` + strings.Replace(rhymesFunction, `"string"`, `"float"`, 1) + `]}`, "type"},
		"a chunk level":               {backend, `{"version":1,"functions":[` + strings.Replace(rhymesFunction, `"document"`, `"chunk"`, 1) + `]}`, "chunk-level"},
		"no description":              {backend, `{"version":1,"functions":[` + strings.Replace(rhymesFunction, `notes that rhyme with a word`, ``, 1) + `]}`, "description"},
		"an example without the call": {backend, `{"version":1,"functions":[` + strings.Replace(rhymesFunction, `rhymes(\"moon\")`, `moon`, 1) + `]}`, "example"},
		"no backend":                  {"", `{"version":1,"functions":[` + rhymesFunction + `]}`, "contributes.searchFunctions"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := readSearchFunctions(c.backend, c.functions)
			if err == nil || !strings.Contains(err.Error(), c.field) {
				t.Errorf("err = %v, want an issue naming %q", err, c.field)
			}
			if !errors.Is(err, manifest.ErrInvalidContribution) && !errors.Is(err, manifest.ErrNoBackend) {
				t.Errorf("err = %v, want an invalid contribution", err)
			}
		})
	}
}

func searchPlugin(id string, names ...string) manifest.DiscoveredPlugin {
	var functions []manifest.SearchFunction
	for _, name := range names {
		functions = append(functions, manifest.SearchFunction{Name: name})
	}
	return manifest.DiscoveredPlugin{Manifest: manifest.Manifest{
		ID: id, Contributes: manifest.Contributes{SearchFunctions: manifest.SearchFunctions{Version: 1, Functions: functions}},
	}}
}

func TestTheSecondPluginToDeclareASearchFunctionIsRefusedWhole(t *testing.T) {
	conflicts := manifest.ListContributionConflicts([]manifest.DiscoveredPlugin{
		searchPlugin("md.example.a", "rhymes"), searchPlugin("md.example.b", "rhymes", "echoes"),
	}, manifest.CoreClaims{})

	if len(conflicts) != 1 || conflicts[0].PluginID != "md.example.b" || conflicts[0].OwnerID != "md.example.a" ||
		conflicts[0].What != "search function" || conflicts[0].ID != "rhymes" {
		t.Errorf("conflicts = %+v", conflicts)
	}
}

func TestAPluginThatDeclaresABuiltinSearchFunctionIsRefusedWithTheCoreAsOwner(t *testing.T) {
	conflicts := manifest.ListContributionConflicts([]manifest.DiscoveredPlugin{searchPlugin("md.example.a", "words")},
		manifest.CoreClaims{SearchFunctions: []string{"words", "meaning"}})

	if len(conflicts) != 1 || conflicts[0].PluginID != "md.example.a" || conflicts[0].OwnerID != manifest.CoreOwnerID {
		t.Errorf("conflicts = %+v", conflicts)
	}
}

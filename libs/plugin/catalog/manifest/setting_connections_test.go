package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readSettings(settings string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","settings":[` + settings + `]}`))
}

func TestASettingOfRowsIsReadFromAManifest(t *testing.T) {
	m, err := readSettings(`{"key":"connections","type":"connections","title":"Connections","description":"Services tried in order.","operations":["llm.complete"],"max":5}`)
	if err != nil {
		t.Fatal(err)
	}

	declared, ok := m.GetSetting("connections")
	if !ok || declared.Type != manifest.SettingTypeConnections || !slices.Equal(declared.Operations, []string{"llm.complete"}) {
		t.Fatalf("the connections setting = %+v, %v", declared, ok)
	}
	if declared.Max == nil || *declared.Max != 5 {
		t.Errorf("Max = %v, want 5", declared.Max)
	}
}

func TestASettingOfRowsFollowsTheConditionOfAnotherSetting(t *testing.T) {
	m, err := readSettings(`{"key":"source","type":"enum","title":"Source","default":"local","options":["local","connection"]},
		{"key":"connections","type":"connections","title":"Connections","operations":["llm.embed"],"max":1,"when":{"setting":"source","is":["connection"]}}`)
	if err != nil {
		t.Fatal(err)
	}

	local := m.ResolveSettings(map[string]any{"connections": []any{}})
	if _, isKept := local["connections"]; isKept {
		t.Errorf("under source local the rows are in force: %v", local)
	}
	remote := m.ResolveSettings(map[string]any{"source": "connection", "connections": []any{}})
	if _, isKept := remote["connections"]; !isKept {
		t.Errorf("under source connection the rows are left out: %v", remote)
	}
}

func TestAPluginMayDeclareASettingOfRowsForEachOperation(t *testing.T) {
	_, err := readSettings(`{"key":"completions","type":"connections","title":"Completions","operations":["llm.complete"],"max":3},
		{"key":"embeddings","type":"connections","title":"Embeddings","operations":["llm.embed"],"max":1}`)

	if err != nil {
		t.Errorf("err = %v, want none", err)
	}
}

func TestASettingOfRowsThatBreaksARuleIsRejected(t *testing.T) {
	for name, settings := range map[string]string{
		"no operation":             `{"key":"a","type":"connections","title":"A","max":1}`,
		"an operation in capitals": `{"key":"a","type":"connections","title":"A","operations":["Complete"],"max":1}`,
		"an operation with a dash": `{"key":"a","type":"connections","title":"A","operations":["co-mplete"],"max":1}`,
		"no max":                   `{"key":"a","type":"connections","title":"A","operations":["llm.complete"]}`,
		"a max of zero":            `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":0}`,
		"a negative max":           `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":-2}`,
		"a max that is not whole":  `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1.5}`,
		"a default":                `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"default":[]}`,
		"options":                  `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"options":["x"]}`,
		"options from a registry":  `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"optionsFrom":"themes"}`,
		"a min":                    `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"min":1}`,
		"a step":                   `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"step":1}`,
		"extensions":               `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"accept":[".txt"]}`,
		"several paths":            `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"multiple":true}`,
		"a preview":                `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"preview":true}`,
		"a generated value":        `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1,"generate":"token"}`,
		"operations on a string":   `{"key":"a","type":"string","title":"A","operations":["llm.complete"]}`,
		"operations on a number":   `{"key":"a","type":"number","title":"A","operations":["llm.complete"]}`,
		"discovery on a boolean":   `{"key":"a","type":"boolean","title":"A","discover":true}`,
		"two settings for one operation": `{"key":"a","type":"connections","title":"A","operations":["llm.complete"],"max":1},` +
			`{"key":"b","type":"connections","title":"B","operations":["llm.complete"],"max":2}`,
	} {
		_, err := readSettings(settings)
		if !errors.Is(err, manifest.ErrInvalidSetting) {
			t.Errorf("%s: err = %v, want ErrInvalidSetting", name, err)
		}
	}
}

func TestASettingOfRowsThatDiscoversIsReadFromAManifest(t *testing.T) {
	m, err := readSettings(`{"key":"agents","type":"connections","title":"Agents","operations":["acp.task"],"max":5,"discover":true},
		{"key":"completions","type":"connections","title":"Completions","operations":["llm.complete"],"max":1}`)
	if err != nil {
		t.Fatal(err)
	}

	agents, _ := m.GetSetting("agents")
	completions, _ := m.GetSetting("completions")
	if !agents.Discover {
		t.Error("the setting that declares discover does not discover")
	}
	if completions.Discover {
		t.Error("the setting that does not declare discover discovers")
	}
}

func TestASettingOfRowsThatNamesAnOperationTwiceIsRejectedOnce(t *testing.T) {
	_, err := readSettings(`{"key":"completions","type":"connections","title":"Completions","operations":["llm.complete","llm.complete"],"max":1}`)

	if !errors.Is(err, manifest.ErrInvalidSetting) || strings.Count(err.Error(), "llm.complete") != 1 || !strings.Contains(err.Error(), "twice") {
		t.Errorf("err = %v, want one issue that names the operation twice", err)
	}
}

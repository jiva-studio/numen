package settings_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const rowsPluginID = "md.numen.tools.proofreading"

func declaredWithRows(owner string) ([]manifest.Setting, bool) {
	switch owner {
	case rowsPluginID:
		return []manifest.Setting{
			{Key: "connections", Type: "connections", Title: "Connections", Operations: []string{"llm.complete"}, Max: new(float64(2)), Discover: true},
			{Key: "label", Type: "string", Title: "Label"},
		}, true
	case "md.numen.engines.embeddings":
		return []manifest.Setting{
			{Key: "source", Type: "enum", Title: "Source", Options: manifest.SettingOptions{{Value: "local"}, {Value: "connection"}}},
			{Key: "connections", Type: "connections", Title: "Connections", Operations: []string{"llm.embed"}, Max: new(float64(1))},
		}, true
	}
	return nil, false
}

func knownConnections(name string) ([]string, bool) {
	if name == settings.RegistryConnections {
		return []string{"c1f0a9e20", "c77b01d40"}, true
	}
	return nil, false
}

func checkRows(t *testing.T, rows string) error {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(rows), &value); err != nil {
		t.Fatal(err)
	}
	return settings.CheckWrite(declaredWithRows, knownConnections, []string{rowsPluginID, "connections"}, value)
}

func TestRowsOfExistingConnectionsAreAccepted(t *testing.T) {
	for what, rows := range map[string]string{
		"no row":                       `[]`,
		"a row":                        `[{"connection":"c1f0a9e20","model":"google/gemini-2.5-flash"}]`,
		"a row with a reasoning level": `[{"connection":"c1f0a9e20","model":"m","reasoning":"low"}]`,
		"two connections":              `[{"connection":"c1f0a9e20","model":"m"},{"connection":"c77b01d40","model":"m"}]`,
		"two models of one connection": `[{"connection":"c1f0a9e20","model":"a"},{"connection":"c1f0a9e20","model":"b"}]`,
		"a row with no option":         `[{"connection":"c1f0a9e20"}]`,
		"an option the core reads as a name only": `[{"connection":"c1f0a9e20","model":"m","temperature":"1"}]`,
		"one model at two reasoning levels":       `[{"connection":"c1f0a9e20","model":"m"},{"connection":"c1f0a9e20","model":"m","reasoning":"low"}]`,
	} {
		if err := checkRows(t, rows); err != nil {
			t.Errorf("%s was rejected: %v", what, err)
		}
	}
}

func TestRowsThatBreakTheRulesOfTheSettingAreRejected(t *testing.T) {
	for what, c := range map[string]struct {
		rows string
		want error
	}{
		"a text":                        {`"c1f0a9e20"`, settings.ErrWrongType},
		"an object":                     {`{"connection":"c1f0a9e20","model":"m"}`, settings.ErrWrongType},
		"a row that is a text":          {`["c1f0a9e20"]`, settings.ErrWrongType},
		"a connection that is a number": {`[{"connection":7,"model":"m"}]`, settings.ErrWrongType},
		"a model that is a number":      {`[{"connection":"c1f0a9e20","model":7}]`, settings.ErrWrongType},
		"a reasoning that is a flag":    {`[{"connection":"c1f0a9e20","model":"m","reasoning":true}]`, settings.ErrWrongType},
		"no connection":                 {`[{"model":"m"}]`, settings.ErrWrongType},
		"an empty connection":           {`[{"connection":"","model":"m"}]`, settings.ErrWrongType},
		"a connection that is gone":     {`[{"connection":"c00000000","model":"m"}]`, settings.ErrNotAnOption},
		"the same connection and options twice": {
			`[{"connection":"c1f0a9e20","model":"m","reasoning":"low"},{"reasoning":"low","connection":"c1f0a9e20","model":"m"}]`, settings.ErrWrongType,
		},
		"more rows than max": {
			`[{"connection":"c1f0a9e20","model":"a"},{"connection":"c1f0a9e20","model":"b"},{"connection":"c77b01d40","model":"a"}]`, settings.ErrOutOfRange,
		},
	} {
		if err := checkRows(t, c.rows); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", what, err, c.want)
		}
	}
}

func TestAnEmptyValueClearsTheRows(t *testing.T) {
	err := settings.CheckWrite(declaredWithRows, knownConnections, []string{rowsPluginID, "connections"}, nil)

	if err != nil {
		t.Errorf("err = %v, want none", err)
	}
}

func TestARowsSettingIsCheckedWithoutAListOfConnectionsWhenNoneIsGiven(t *testing.T) {
	var value any
	if err := json.Unmarshal([]byte(`[{"connection":"c00000000","model":"m"}]`), &value); err != nil {
		t.Fatal(err)
	}

	err := settings.CheckWrite(declaredWithRows, installedThemes, []string{rowsPluginID, "connections"}, value)

	if err != nil {
		t.Errorf("err = %v, want none for a registry that does not list connections", err)
	}
}

const rowsFile = `{
	"connections": [
		{"id":"c1f0a9e20","type":"md.numen.connections.llm/openrouter","name":"OpenRouter","revision":1},
		{"id":"c77b01d40","type":"md.numen.connections.llm/openai-compatible","name":"Ollama","revision":1}
	],
	"md.numen.tools.proofreading": {"connections": %s}
}`

func openRows(t *testing.T, rows string) settings.File {
	t.Helper()
	cfg, err := openAt(write(t, fmt.Sprintf(rowsFile, rows)))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredWithRows, nil)
	return cfg
}

func listRowsInForce(t *testing.T, cfg settings.File) []map[string]any {
	t.Helper()
	got, _ := cfg.GetValue(rowsPluginID, "connections")
	list, isList := got.([]any)
	if !isList {
		t.Fatalf("the rows in force are %v, want a list", got)
	}
	rows := make([]map[string]any, 0, len(list))
	for _, one := range list {
		row, isRow := one.(map[string]any)
		if !isRow {
			t.Fatalf("a row in force is %v, want an object", one)
		}
		rows = append(rows, row)
	}
	return rows
}

func TestOnlyTheRowsThatFailOnReadAreLeftOutWithANotice(t *testing.T) {
	cfg := openRows(t, `[
		{"connection":"c1f0a9e20","model":"a"},
		{"connection":"c00000000","model":"b"},
		{"connection":"c1f0a9e20","model":"a"},
		{"connection":"c77b01d40","model":"c","temperature":1},
		{"connection":"c77b01d40","model":"d"}
	]`)

	rows := listRowsInForce(t, cfg)
	if len(rows) != 2 {
		t.Fatalf("rows in force = %v, want the first and the last", rows)
	}
	if rows[0]["model"] != "a" || rows[1]["model"] != "d" {
		t.Errorf("rows in force = %v", rows)
	}
	if !hasNotice(cfg.Notices, rowsPluginID, "connections", "c00000000") {
		t.Errorf("no notice names the row of the missing connection: %q", cfg.Notices)
	}
	if len(cfg.Notices) != 3 {
		t.Errorf("notices = %q, want one for each of the 3 rows left out", cfg.Notices)
	}
}

func TestRowsBeyondMaxAreLeftOutOnRead(t *testing.T) {
	cfg := openRows(t, `[
		{"connection":"c1f0a9e20","model":"a"},
		{"connection":"c1f0a9e20","model":"b"},
		{"connection":"c1f0a9e20","model":"c"}
	]`)

	if rows := listRowsInForce(t, cfg); len(rows) != 2 {
		t.Errorf("rows in force = %v, want the first 2", rows)
	}
}

func TestAValueThatIsNoListIsLeftOutWholeOnRead(t *testing.T) {
	cfg := openRows(t, `"c1f0a9e20"`)

	if got, found := cfg.GetValue(rowsPluginID, "connections"); found {
		t.Errorf("a text is in force as %v", got)
	}
	if !hasNotice(cfg.Notices, rowsPluginID, "connections") {
		t.Errorf("notices = %q", cfg.Notices)
	}
}

func TestRowsOfConnectionsThatExistProduceNoNotice(t *testing.T) {
	cfg := openRows(t, `[{"connection":"c1f0a9e20","model":"a"},{"connection":"c77b01d40","model":"b","reasoning":"low"}]`)

	if len(cfg.Notices) != 0 {
		t.Errorf("notices = %q", cfg.Notices)
	}
}

func checkDiscover(t *testing.T, owner, choices string) error {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(choices), &value); err != nil {
		t.Fatal(err)
	}
	return settings.CheckWrite(declaredWithRows, knownConnections, []string{owner, "discover"}, value)
}

func TestTheChoiceToDiscoverIsAcceptedForASettingThatDiscovers(t *testing.T) {
	for what, choices := range map[string]string{"off": `{"connections":false}`, "on": `{"connections":true}`, "no choice": `{}`} {
		if err := checkDiscover(t, rowsPluginID, choices); err != nil {
			t.Errorf("%s was rejected: %v", what, err)
		}
	}
}

func TestAChoiceToDiscoverThatBreaksTheRulesIsRejected(t *testing.T) {
	for what, c := range map[string]struct {
		owner, choices string
		want           error
	}{
		"a flag":                           {rowsPluginID, `false`, settings.ErrWrongType},
		"a text for the setting":           {rowsPluginID, `{"connections":"off"}`, settings.ErrWrongType},
		"a setting that is no rows":        {rowsPluginID, `{"label":false}`, settings.ErrUndeclared},
		"a setting that does not discover": {"md.numen.engines.embeddings", `{"connections":false}`, settings.ErrUndeclared},
	} {
		if err := checkDiscover(t, c.owner, c.choices); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", what, err, c.want)
		}
	}
}

func listConnectionsWithDetected(name string) ([]string, bool) {
	switch name {
	case settings.RegistryDetectedConnections:
		return settings.ListDetectedConnectionIDs(fixedDetection{}), true
	case settings.RegistryPendingDetections:
		return []string{heronTypeID}, true
	}
	return knownConnections(name)
}

func TestARowOfADetectedConnectionIsAcceptedOnWrite(t *testing.T) {
	for _, id := range []string{kiteAgentID, heronAgentID} {
		var value any
		if err := json.Unmarshal([]byte(`[{"connection":"`+id+`","model":"m"}]`), &value); err != nil {
			t.Fatal(err)
		}

		err := settings.CheckWrite(declaredWithRows, listConnectionsWithDetected, []string{rowsPluginID, "connections"}, value)

		if err != nil {
			t.Errorf("%s: err = %v, want none", id, err)
		}
	}
}

func TestARowOfADetectedConnectionStaysInForceOnRead(t *testing.T) {
	path := write(t, fmt.Sprintf(rowsFile, `[
		{"connection":"`+kiteAgentID+`","model":"m"},
		{"connection":"`+wrenAgentID+`","model":"m"}
	]`))
	store := settings.NewStore(settings.NewFiles(), func() (string, error) { return path, nil }, declaredWithRows, listConnectionsWithDetected)

	cfg, err := store.ReadCheckedFile()
	if err != nil {
		t.Fatal(err)
	}

	rows := listRowsInForce(t, cfg)
	if len(rows) != 1 || rows[0]["connection"] != kiteAgentID {
		t.Errorf("rows in force = %v, want the row of Kite agent only", rows)
	}
	if !hasNotice(cfg.Notices, rowsPluginID, "connections", wrenAgentID) {
		t.Errorf("no notice names the row of the connection that is not detected: %q", cfg.Notices)
	}
}

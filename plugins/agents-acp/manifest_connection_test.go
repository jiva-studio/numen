package agentacp

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

type connectionField struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	IsRequired  bool     `json:"required"`
	OptionsFrom string   `json:"optionsFrom"`
	Default     *float64 `json:"default"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Step        *float64 `json:"step"`
}

type connectionShape struct {
	Services     []string `json:"services"`
	Capabilities []string `json:"capabilities"`
	Targets      []string `json:"targets"`
	Contributes  struct {
		Operations []struct {
			Name  string `json:"name"`
			Title string `json:"title"`
		} `json:"operations"`
		ConnectionTypes []struct {
			ID         string            `json:"id"`
			Title      string            `json:"title"`
			Operations []string          `json:"operations"`
			Fields     []connectionField `json:"fields"`
			RowFields  []connectionField `json:"rowFields"`
		} `json:"connectionTypes"`
	} `json:"contributes"`
	Settings []struct {
		Key  string   `json:"key"`
		Type string   `json:"type"`
		Min  *float64 `json:"min"`
	} `json:"settings"`
}

func readConnectionShape(t *testing.T) connectionShape {
	t.Helper()
	var shape connectionShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

func TestManifestProvidesTheTypesOfACPProgramsThatCompleteAndRunTasks(t *testing.T) {
	types := readConnectionShape(t).Contributes.ConnectionTypes
	var ids []string
	for _, one := range types {
		ids = append(ids, one.ID+"="+one.Title)
		if !slices.Equal(one.Operations, []string{"llm.complete", "llm.models", "llm.reasoning", "acp.task"}) {
			t.Errorf("%s: operations = %v, want llm.complete, the lists of models and reasoning levels, and acp.task", one.ID, one.Operations)
		}
		var keys []string
		for _, field := range one.Fields {
			keys = append(keys, field.Key)
			if field.Type == "secret" {
				t.Errorf("%s: field %q is a secret: the type holds no key", one.ID, field.Key)
			}
		}
		if !slices.Equal(keys, []string{"program", "parallelRequests"}) {
			t.Errorf("%s: fields = %v, want the program and then the parallel requests", one.ID, keys)
		}
	}
	want := []string{"acp=ACP agent", "claude-code=Claude Code", "codex=OpenAI Codex", "antigravity=Antigravity"}
	if !slices.Equal(ids, want) {
		t.Fatalf("types = %v, want %v", ids, want)
	}
}

// The program is a file. The type acp names no program of its own, so a person chooses one; the others
// find their known program where the field is empty.
func TestTheProgramIsAFileThatOnlyTheTypeACPRequires(t *testing.T) {
	for _, one := range readConnectionShape(t).Contributes.ConnectionTypes {
		if len(one.Fields) == 0 {
			t.Fatalf("%s has no fields", one.ID)
		}
		program := one.Fields[0]
		if program.Key != "program" || program.Type != "file" || program.IsRequired != (one.ID == "acp") {
			t.Errorf("%s: first field = %+v, want the file field program, required for acp alone", one.ID, program)
		}
	}
}

func TestOneRequestRunsAtATimeAndAtMostEight(t *testing.T) {
	for _, one := range readConnectionShape(t).Contributes.ConnectionTypes {
		if len(one.Fields) != 2 {
			t.Fatalf("%s: fields = %+v, want two", one.ID, one.Fields)
		}
		parallel := one.Fields[1]
		if parallel.Type != "number" || parallel.Default == nil || *parallel.Default != 1 || parallel.Min == nil || *parallel.Min != 1 ||
			parallel.Max == nil || *parallel.Max != 8 || parallel.Step == nil || *parallel.Step != 1 {
			t.Errorf("%s: parallelRequests = %+v, want a number defaulting to 1 from 1 to 8 by 1", one.ID, parallel)
		}
	}
}

// The plugin serves its connection types and nothing else: it is no agent provider of the host.
func TestManifestDeclaresTheProviderServiceAlone(t *testing.T) {
	shape := readConnectionShape(t)
	if !slices.Equal(shape.Services, []string{"numen.v1.ProviderService"}) || len(shape.Capabilities) != 0 {
		t.Errorf("services = %v, capabilities = %v, want the provider service alone", shape.Services, shape.Capabilities)
	}
}

func TestManifestTitlesTheOperationOfTasks(t *testing.T) {
	operations := readConnectionShape(t).Contributes.Operations
	if len(operations) != 1 || operations[0].Name != "acp.task" || operations[0].Title != "run agent tasks" {
		t.Errorf("operations = %+v, want acp.task titled run agent tasks", operations)
	}
}

func TestManifestDeclaresTheSettingRequestTimeout(t *testing.T) {
	for _, one := range readConnectionShape(t).Settings {
		if one.Key == "requestTimeout" {
			if one.Type != "number" || one.Min == nil || *one.Min < 1 {
				t.Errorf("requestTimeout = %+v, want a number of seconds of at least 1", one)
			}
			return
		}
	}
	t.Error("the manifest has no setting requestTimeout")
}

// An agent program cannot start on a phone, so the plugin runs in desktop windows alone and its type
// is not listed anywhere else.
func TestThePluginRunsInDesktopWindowsAlone(t *testing.T) {
	targets := readConnectionShape(t).Targets
	if len(targets) == 0 {
		t.Fatal("the manifest names no target")
	}
	for _, target := range targets {
		if !strings.HasPrefix(target, "desktop.") {
			t.Errorf("target %q is not a desktop window", target)
		}
	}
}

// A row names the model, from the list the program offers, and a reasoning level of that model.
func TestARowStoresAModelAndAReasoningLevelListedByTheProvider(t *testing.T) {
	for _, one := range readConnectionShape(t).Contributes.ConnectionTypes {
		if len(one.RowFields) != 2 {
			t.Fatalf("%s: row fields = %+v, want two", one.ID, one.RowFields)
		}
		model, reasoning := one.RowFields[0], one.RowFields[1]
		if model.Key != "model" || !model.IsRequired || model.OptionsFrom != "llm.models" {
			t.Errorf("%s: first row field = %+v, want the required model listed by llm.models", one.ID, model)
		}
		if reasoning.Key != "reasoning" || reasoning.IsRequired || reasoning.OptionsFrom != "llm.reasoning" {
			t.Errorf("%s: second row field = %+v, want the optional reasoning listed by llm.reasoning", one.ID, reasoning)
		}
	}
}

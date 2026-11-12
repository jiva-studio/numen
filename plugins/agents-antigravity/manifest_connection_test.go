package agentantigravity

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
	Services    []string `json:"services"`
	Targets     []string `json:"targets"`
	Contributes struct {
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

func TestManifestProvidesOneTypeThatCompletesWithNoKeyAndNoProgramField(t *testing.T) {
	types := readConnectionShape(t).Contributes.ConnectionTypes
	if len(types) != 1 || types[0].ID != "antigravity" || types[0].Title != "Antigravity" {
		t.Fatalf("types = %+v, want the type antigravity alone", types)
	}
	if !slices.Equal(types[0].Operations, []string{"llm.complete", "llm.models", "llm.reasoning"}) {
		t.Errorf("operations = %v, want llm.complete with the lists of models and reasoning levels", types[0].Operations)
	}
	for _, field := range types[0].Fields {
		if field.Key != "parallelRequests" {
			t.Errorf("field %q: the type holds no field but parallelRequests", field.Key)
		}
	}
}

func TestOneRequestRunsAtATimeAndAtMostEight(t *testing.T) {
	types := readConnectionShape(t).Contributes.ConnectionTypes
	if len(types) != 1 || len(types[0].Fields) != 1 {
		t.Fatalf("types = %+v, want one type with one field", types)
	}
	fields := types[0].Fields
	parallel := fields[0]
	if parallel.Type != "number" || parallel.Default == nil || *parallel.Default != 1 || parallel.Min == nil || *parallel.Min != 1 ||
		parallel.Max == nil || *parallel.Max != 8 || parallel.Step == nil || *parallel.Step != 1 {
		t.Errorf("parallelRequests = %+v, want a number defaulting to 1 from 1 to 8 by 1", parallel)
	}
}

func TestManifestDeclaresTheProviderServiceAndTheAgentService(t *testing.T) {
	services := readConnectionShape(t).Services
	for _, want := range []string{"numen.v1.AgentService", "numen.v1.ProviderService"} {
		if !slices.Contains(services, want) {
			t.Errorf("services = %v, want %s among them", services, want)
		}
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
func TestARowHoldsAModelAndAReasoningLevelListedByTheProvider(t *testing.T) {
	types := readConnectionShape(t).Contributes.ConnectionTypes
	if len(types) != 1 || len(types[0].RowFields) != 2 {
		t.Fatalf("types = %+v, want one type with two row fields", types)
	}
	model, reasoning := types[0].RowFields[0], types[0].RowFields[1]
	if model.Key != "model" || !model.IsRequired || model.OptionsFrom != "llm.models" {
		t.Errorf("first row field = %+v, want the required model listed by llm.models", model)
	}
	if reasoning.Key != "reasoning" || reasoning.IsRequired || reasoning.OptionsFrom != "llm.reasoning" {
		t.Errorf("second row field = %+v, want the optional reasoning listed by llm.reasoning", reasoning)
	}
}

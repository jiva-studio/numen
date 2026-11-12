package settings_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func listedPrograms() []agent.Program {
	return []agent.Program{
		{
			Provider: "p.one", Name: "alpha", Title: "Alpha", IsInstalled: true, IsDefault: true,
			Models: []agent.Model{{Name: "big", Title: "Big"}, {Name: "small"}},
		},
		{Provider: "p.two", Name: "beta", Title: "Beta", IsInstalled: true},
		{Provider: "p.two", Name: "gamma", Title: "Gamma", Models: []agent.Model{{Name: "g1"}}},
	}
}

func getAgentModelAt() []string { return []string{"agent", "alpha", "model"} }

func getAgentUseAt() []string { return []string{"agent", "use"} }

// A model is usable by the window only if it provides the path it is read from
// and the value that selecting it writes.
func TestEveryModelReturnsWhereItIsReadAndWhatItWrites(t *testing.T) {
	for _, one := range settings.AgentModels("agent", settings.Agent{}, listedPrograms()) {
		if len(one.Path) == 0 {
			t.Errorf("%q is read from nowhere", one.Title)
		}
		if one.Title == "" {
			t.Errorf("the model named %q is drawn as nothing", one.Name)
		}
		if len(one.Writes) == 0 {
			t.Errorf("choosing %q writes nothing", one.Title)
		}
		for _, wrote := range one.Writes {
			if len(wrote.Path) == 0 || wrote.JSON == "" {
				t.Errorf("choosing %q writes %+v", one.Title, wrote)
			}
		}
	}
}

// One setting has one model that an unconfigured installation runs on.
func TestOneModelToASettingIsTheDefault(t *testing.T) {
	for name, programs := range map[string][]agent.Program{
		"programs that mark one": listedPrograms(),
		"programs that mark none": {
			{Name: "alpha", Title: "Alpha", IsInstalled: true, Models: []agent.Model{{Name: "m"}}},
			{Name: "beta", Title: "Beta", IsInstalled: true},
		},
		"no programs": nil,
	} {
		byDefault := map[string]int{}
		for _, one := range settings.AgentModels("agent", settings.Agent{}, programs) {
			if one.IsDefault {
				byDefault[formatPath(one.Path)]++
			}
		}
		if byDefault[formatPath(getAgentUseAt())] != 1 {
			t.Errorf("%s: %s has %d models by default", name, formatPath(getAgentUseAt()), byDefault[formatPath(getAgentUseAt())])
		}
		for _, one := range programs {
			if !one.IsInstalled {
				continue
			}
			pos := formatPath(modelAt(one.Name))
			if byDefault[pos] != 1 {
				t.Errorf("%s: %s has %d models by default", name, pos, byDefault[pos])
			}
		}
	}
}

// The programs the setting can name are the ones the plugins list, with the default one marked, a
// program that is not installed not fetched, and the choice of none last.
func TestTheProgramsTheSettingCanNameAreTheOnesThePluginsList(t *testing.T) {
	var uses []settingsdomain.Model
	for _, one := range settings.AgentModels("agent", settings.Agent{}, listedPrograms()) {
		if formatPath(one.Path) == formatPath(getAgentUseAt()) {
			uses = append(uses, one)
		}
	}

	var names []string
	for _, one := range uses {
		names = append(names, one.Name)
	}
	if want := []string{"alpha", "beta", "gamma", ""}; !slices.Equal(names, want) {
		t.Fatalf("the programs are %q, want %q", names, want)
	}
	if !uses[0].IsDefault || uses[1].IsDefault || uses[2].IsDefault || uses[3].IsDefault {
		t.Errorf("the default is %v", uses)
	}
	if uses[0].Presence != settingsdomain.Present || uses[2].Presence != settingsdomain.NotFetched || uses[3].Presence != settingsdomain.NothingToFetch {
		t.Errorf("the presence is %v, %v, %v", uses[0].Presence, uses[2].Presence, uses[3].Presence)
	}
	if got := uses[1].Writes; len(got) != 1 || formatPath(got[0].Path) != "agent.use" || got[0].JSON != `"beta"` {
		t.Errorf("choosing beta writes %+v", got)
	}
	if got := uses[3].Writes; len(got) != 1 || got[0].JSON != `""` {
		t.Errorf("choosing none writes %+v", got)
	}
}

// With no plugin listing a program, the only choice is none, and it is the one the installation
// runs on.
func TestWithNoProgramsNothingAnswersByDefault(t *testing.T) {
	models := settings.AgentModels("agent", settings.Agent{}, nil)
	if len(models) != 1 || models[0].Name != "" || !models[0].IsDefault {
		t.Errorf("the models are %+v", models)
	}
}

// The models of a program are those it lists, the one it marks or else the first the default, each
// writing where the program's section keeps its model.
func TestAgentModels_ListedModelsWithFirstAsDefault(t *testing.T) {
	target := modelsAt(t, settings.Agent{}, listedPrograms(), getAgentModelAt())
	if len(target) != 2 || target[0].Name != "big" || target[1].Name != "small" {
		t.Fatalf("the program offers %+v", target)
	}
	if !target[0].IsDefault || target[1].IsDefault {
		t.Errorf("the default is %+v", target)
	}
	if target[1].Title != "small" || target[0].Title != "Big" {
		t.Errorf("the titles are %q and %q", target[0].Title, target[1].Title)
	}
	if got := target[1].Writes; len(got) != 1 || formatPath(got[0].Path) != "agent.alpha.model" || got[0].JSON != `"small"` {
		t.Errorf("choosing small writes %+v", got)
	}
	if target[0].Presence != settingsdomain.NothingToFetch {
		t.Errorf("the presence is %v", target[0].Presence)
	}
}

// The model a program marks as its own is the default where the settings name none.
func TestTheModelAProgramMarksIsTheDefault(t *testing.T) {
	programs := []agent.Program{{
		Name: "alpha", Title: "Alpha", IsInstalled: true,
		Models: []agent.Model{{Name: "one"}, {Name: "two", IsDefault: true}},
	}}
	target := modelsAt(t, settings.Agent{}, programs, getAgentModelAt())
	if target[0].IsDefault || !target[1].IsDefault {
		t.Errorf("the default is %+v", target)
	}
}

// The model the settings name is the one in force when the program lists it, and goes last under
// its own group where it does not.
func TestTheModelTheSettingsNameIsCached(t *testing.T) {
	named := settings.Agent{}
	named.Programs = map[string]settings.AgentProgram{"alpha": {Model: "small"}}
	target := modelsAt(t, named, listedPrograms(), getAgentModelAt())
	if target[0].IsDefault || !target[1].IsDefault {
		t.Errorf("the model in force is %+v", target)
	}

	named.Programs = map[string]settings.AgentProgram{"alpha": {Model: "custom-1"}}
	target = modelsAt(t, named, listedPrograms(), getAgentModelAt())
	last := target[len(target)-1]
	if len(target) != 3 || last.Name != "custom-1" || last.Group != "Named in the settings" || last.IsDefault {
		t.Errorf("the models are %+v", target)
	}
	if target[0].Group != "In full" || !target[0].IsDefault {
		t.Errorf("the models are %+v", target)
	}
}

// A program that lists no models offers only the default model until the user
// names one.
func TestAgentModels_NoListedModelsYieldsDefaultOnly(t *testing.T) {
	target := modelsAt(t, settings.Agent{}, listedPrograms(), []string{"agent", "beta", "model"})
	if len(target) != 1 || target[0].Name != "" || !target[0].IsDefault {
		t.Errorf("the program opens on %+v", target)
	}

	named := settings.Agent{}
	named.Programs = map[string]settings.AgentProgram{"beta": {Model: "mine"}}
	target = modelsAt(t, named, listedPrograms(), []string{"agent", "beta", "model"})
	if len(target) != 1 || target[0].Name != "mine" || target[0].Title != "mine" || !target[0].IsDefault {
		t.Errorf("the program opens on %+v", target)
	}
}

// A program that is not installed offers no models unless the settings name it.
func TestAgentModels_UninstalledProgramYieldsConfiguredOnly(t *testing.T) {
	pos := formatPath([]string{"agent", "gamma", "model"})
	for _, one := range settings.AgentModels("agent", settings.Agent{}, listedPrograms()) {
		if formatPath(one.Path) == pos {
			t.Errorf("a program that is not installed offers %+v", one)
		}
	}

	named := settings.Agent{}
	named.Use = new("gamma")
	target := modelsAt(t, named, listedPrograms(), []string{"agent", "gamma", "model"})
	if len(target) != 1 || target[0].Name != "g1" {
		t.Errorf("the named program offers %+v", target)
	}
}

func modelsAt(t *testing.T, target settings.Agent, programs []agent.Program, setting []string) []settingsdomain.Model {
	t.Helper()
	found := []settingsdomain.Model{}
	for _, one := range settings.AgentModels("agent", target, programs) {
		if formatPath(one.Path) == formatPath(setting) {
			found = append(found, one)
		}
	}
	if len(found) == 0 {
		t.Fatalf("%s can be set to nothing", formatPath(setting))
	}
	return found
}

func modelAt(program string) []string { return []string{"agent", program, "model"} }

func formatPath(path []string) string { return strings.Join(path, ".") }

// The levels of reasoning a model lists, and the one it works at unasked, stay on its row.
func TestAModelKeepsTheLevelsOfReasoningItLists(t *testing.T) {
	programs := []agent.Program{{
		Name: "alpha", Title: "Alpha", IsInstalled: true,
		Models: []agent.Model{
			{Name: "deep", ReasoningLevels: []domain.ReasoningLevel{{Name: "low", Title: "Low"}, {Name: "high", Title: "High"}}, DefaultReasoning: "low"},
			{Name: "plain"},
		},
	}}
	target := modelsAt(t, settings.Agent{}, programs, getAgentModelAt())
	if len(target[0].ReasoningLevels) != 2 || target[0].ReasoningLevels[1].Name != "high" || target[0].DefaultReasoning != "low" {
		t.Errorf("the model offers %+v", target[0])
	}
	if len(target[1].ReasoningLevels) != 0 {
		t.Errorf("a model listing none offers %+v", target[1].ReasoningLevels)
	}
}

package reasoning

import (
	"slices"
	"testing"
)

func TestTheTitleIsSplitIntoNameAndLevel(t *testing.T) {
	cases := []struct{ title, name, level string }{
		{"Gemini 3.8 Flash (High)", "Gemini 3.8 Flash", "high"},
		{"  Model (Extra) (XHigh) ", "Model (Extra)", "xhigh"},
		{"M(High)", "M", "high"},
		{"Gemini (Highx", "Gemini (Highx", ""},
		{"Model (Preview)", "Model (Preview)", ""},
		{"(High)", "(High)", ""},
		{"Plain", "Plain", ""},
	}
	for _, one := range cases {
		name, level := SplitTitle(one.title)
		if name != one.name || level != one.level {
			t.Errorf("%q split into %q and %q, want %q and %q", one.title, name, level, one.name, one.level)
		}
	}
}

func listing() []Listed {
	return []Listed{
		{"flash-high", "Flash 3 (High)"},
		{"flash-medium", "Flash 3 (Medium)"},
		{"flash-low", "Flash 3 (Low)"},
		{"oss-medium", "OSS 120B (Medium)"},
		{"solo", "Solo"},
	}
}

func TestModelsListedOncePerLevelFoldIntoOneFamily(t *testing.T) {
	got := Fold(listing())
	if len(got) != 3 {
		t.Fatalf("got %d families, want 3: %v", len(got), got)
	}
	flash := got[0]
	if flash.Name != "flash" || flash.Title != "Flash 3" || flash.Default != "medium" {
		t.Errorf("the family is %+v", flash)
	}
	names := []string{}
	for _, one := range flash.Levels {
		names = append(names, one.Name+":"+one.Title)
	}
	if want := []string{"low:Low", "medium:Medium", "high:High"}; !slices.Equal(names, want) {
		t.Errorf("the levels are %v, want %v", names, want)
	}
	if got[1].Name != "oss-medium" || got[1].Title != "OSS 120B (Medium)" || len(got[1].Levels) != 0 {
		t.Errorf("a model of one level stays as listed: %+v", got[1])
	}
	if got[2].Name != "solo" || len(got[2].Levels) != 0 {
		t.Errorf("a model of no level stays as listed: %+v", got[2])
	}
}

func TestTheFamilyNameFallsBackToTheFirstListedName(t *testing.T) {
	got := Fold([]Listed{{"alpha", "M (Low)"}, {"beta", "M (High)"}})
	if got[0].Name != "alpha" || got[0].GetVariant("high") != "beta" {
		t.Errorf("got %+v", got[0])
	}
	mismatched := Fold([]Listed{{"alpha-low", "M (Low)"}, {"beta-high", "M (High)"}})
	if mismatched[0].Name != "alpha-low" {
		t.Errorf("mismatched prefix family name = %q, want alpha-low", mismatched[0].Name)
	}
	emptyPrefix := Fold([]Listed{{"-low", "M (Low)"}, {"-high", "M (High)"}})
	if emptyPrefix[0].Name != "-low" {
		t.Errorf("empty prefix family name = %q, want -low", emptyPrefix[0].Name)
	}
	unleveled := Fold([]Listed{{"a", "Plain"}, {"b", "Plain"}})
	if len(unleveled) != 2 || unleveled[0].Name != "a" || unleveled[1].Name != "b" {
		t.Errorf("unleveled models folded into %+v, want two separate families", unleveled)
	}
}

func TestAChosenLevelResolvesToTheListedModel(t *testing.T) {
	families := Fold(listing())
	cases := []struct{ model, level, want string }{
		{"flash", "low", "flash-low"},
		{"flash", "high", "flash-high"},
		{"flash", "", "flash-medium"},
		{"flash", "ultra", "flash-medium"},
		{"flash-high", "low", "flash-high"},
		{"solo", "high", "solo"},
		{"oss-medium", "low", "oss-medium"},
		{"unknown", "low", "unknown"},
	}
	for _, one := range cases {
		if got := ResolveModel(families, one.model, one.level); got != one.want {
			t.Errorf("model %q at %q resolves to %q, want %q", one.model, one.level, got, one.want)
		}
	}
}

func TestTheFamilyIsAvailableWithItsLevelsAndDefault(t *testing.T) {
	got := Fold(listing())[0].GetModel()
	if len(got.ReasoningLevels) != 3 || got.DefaultReasoning != "medium" || got.Name != "flash" {
		t.Errorf("got %v", got)
	}
}

func TestLevelHelpers(t *testing.T) {
	levels := NewLevels("max", "minimal", "low", "xhigh", "weird")
	Sort(levels)
	if levels[0].Name != "minimal" || levels[1].Name != "low" || levels[2].Name != "xhigh" || levels[4].Name != "weird" {
		t.Errorf("sorted %v", levels)
	}
	if levels[2].Title != "Extra high" {
		t.Errorf("title %q", levels[2].Title)
	}
	if NewLevel("  XHigh ").Name != "xhigh" || NewLevel("  High ").Title != "High" {
		t.Error("NewLevel did not normalize case and whitespace")
	}
	if GetDefault(levels) != "minimal" || GetDefault(nil) != "" || GetDefault(NewLevels("high", "medium")) != "medium" {
		t.Error("the default is wrong")
	}
	if Normalize(levels, "max") != "max" || Normalize(levels, "none") != "minimal" {
		t.Error("the normalized level is wrong")
	}
	if NewModel("m", "M", NewLevels("low", "high"), "bogus").DefaultReasoning != "" || Pick(levels, "max") != "max" || Pick(levels, "none") != "" {
		t.Error("a default outside the levels is not kept")
	}
	if NewLevel("").Title != "" {
		t.Error("empty input")
	}
}

func TestResolveModel_FamilyWithoutLevelUsesDefaultLevel(t *testing.T) {
	families := Fold([]Listed{
		{"gemini-3.8-flash-high", "Gemini 3.8 Flash (High)"},
		{"gemini-3.8-flash-medium", "Gemini 3.8 Flash (Medium)"},
		{"gemini-3.8-flash-low", "Gemini 3.8 Flash (Low)"},
	})
	if got := ResolveModel(families, "gemini-3.8-flash", ""); got != "gemini-3.8-flash-medium" {
		t.Errorf("resolved to %q", got)
	}
}

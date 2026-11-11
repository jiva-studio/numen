package model

import (
	"slices"
	"testing"
)

func TestTheModelsAreTheLinesThatHoldANameAndATitle(t *testing.T) {
	got := Parse([]byte(
		"Fetching available models...\n" +
			"\r⠋ Fetching available models...\rgemini-4-argon-high     Gemini 4 Argon (High)\n" +
			"m-high\tModel (High)\n" +
			"bare\t\n" +
			"\tno name\n" +
			"m-low\tModel (Low)\n"))
	want := []Model{
		{"gemini-4-argon-high", "Gemini 4 Argon (High)"},
		{"m-high", "Model (High)"},
		{"bare", "bare"},
		{"m-low", "Model (Low)"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("the models are %v, want %v", got, want)
	}
}

func TestParseKeepsOnlyLinesWithAModelId(t *testing.T) {
	// A carriage return at the start, a double space with no name, a name with a space, and no title.
	got := Parse([]byte(
		"\rcr-at-zero\tCR Zero\n" +
			"   leading-spaces-only\n" +
			"has space  Not A Model ID\n" +
			"space-bare  \n"))
	want := []Model{
		{Name: "cr-at-zero", Title: "CR Zero"},
		{Name: "space-bare", Title: "space-bare"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("the models are %v, want %v", got, want)
	}
}

func TestFoldGroupsTheLevelsOfAModel(t *testing.T) {
	families := Fold([]Model{{Name: "m-high", Title: "Model (High)"}, {Name: "m-low", Title: "Model (Low)"}, {Name: "solo", Title: "Solo"}})
	if len(families) != 2 {
		t.Fatalf("the families are %v", families)
	}
}

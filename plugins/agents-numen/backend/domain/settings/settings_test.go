package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
)

func TestAnswerLengthIsReadFromTheCamelCaseKey(t *testing.T) {
	store := settings.NewStore()
	store.Configure(map[string]string{"maxTokens": "1024"})

	if got := store.Get().MaxTokens; got == nil || *got != 1024 {
		t.Errorf("max tokens = %v, want 1024", got)
	}
}

func TestAnswerLengthIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	store := settings.NewStore()
	store.Configure(map[string]string{"max_tokens": "1024"})

	if got := store.Get().MaxTokens; got != nil {
		t.Errorf("max tokens = %d, want none", *got)
	}
}

func TestASettingAbsentFromTheValuesKeepsItsValue(t *testing.T) {
	store := settings.NewStore()
	store.Configure(map[string]string{"connections": `[{"connection":"c1","model":"m1","reasoning":"low"}]`, "temperature": "0.5"})
	store.Configure(map[string]string{"maxSteps": "7"})

	got := store.Get()
	want := settings.Row{Connection: "c1", Model: "m1", Reasoning: "low"}
	if !got.HasRow || got.Row != want || got.Temperature == nil || *got.Temperature != 0.5 || got.MaxSteps != 7 {
		t.Errorf("the settings are %+v", got)
	}
}

func TestTheStepLimitIsTwentyFiveUntilOneIsConfigured(t *testing.T) {
	store := settings.NewStore()
	before := store.Get().GetMaxSteps()
	store.Configure(map[string]string{"maxSteps": "5"})

	if after := store.Get().GetMaxSteps(); before != 25 || after != 5 {
		t.Errorf("the limits are %d and %d", before, after)
	}
}

func TestTheFirstRowNamesTheConnection(t *testing.T) {
	store := settings.NewStore()
	if store.Get().HasRow {
		t.Error("a new store holds a row")
	}
	store.Configure(map[string]string{"connections": `[{"connection":"c1","model":"m1"},{"connection":"c2","model":"m2"}]`})
	if got := store.Get(); !got.HasRow || got.Row.Connection != "c1" || got.Row.Model != "m1" {
		t.Errorf("the row is %+v", got.Row)
	}
	store.Configure(map[string]string{"connections": "[]"})
	if store.Get().HasRow {
		t.Error("an empty list of rows left the row")
	}
}

func TestAValueThatIsNoListOfRowsKeepsTheRow(t *testing.T) {
	store := settings.NewStore()
	store.Configure(map[string]string{"connections": `[{"connection":"c1","model":"m1"}]`})
	store.Configure(map[string]string{"connections": "not rows"})

	if got := store.Get(); !got.HasRow || got.Row.Connection != "c1" {
		t.Errorf("the row is %+v", got.Row)
	}
}

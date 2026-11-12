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
	store.Configure(map[string]string{"baseUrl": "http://localhost:1234", "model": "m1", "temperature": "0.5"})
	store.Configure(map[string]string{"model": "m2"})

	got := store.Get()
	if got.BaseURL != "http://localhost:1234" || got.Model != "m2" || got.Temperature == nil || *got.Temperature != 0.5 {
		t.Errorf("the settings are %+v", got)
	}
}

func TestATaskLimitOverridesTheConfiguredOne(t *testing.T) {
	store := settings.NewStore()
	store.Configure(map[string]string{"maxSteps": "5"})
	got := store.Get()

	if got.GetMaxSteps(2) != 2 || got.GetMaxSteps(0) != 5 {
		t.Errorf("the limits are %d and %d", got.GetMaxSteps(2), got.GetMaxSteps(0))
	}
}

func TestTheModelIsTheTaskOnesThenTheConfiguredOneThenTheDefault(t *testing.T) {
	store := settings.NewStore()
	if got := store.Get().GetModel(""); got != settings.DefaultModel {
		t.Errorf("the model is %q", got)
	}
	store.Configure(map[string]string{"model": "m1"})
	if got := store.Get().GetModel(""); got != "m1" {
		t.Errorf("the model is %q", got)
	}
	if got := store.Get().GetModel("routed"); got != "routed" {
		t.Errorf("the model is %q", got)
	}
}

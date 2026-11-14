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
	store.Configure(map[string]string{"temperature": "0.5"})
	store.Configure(map[string]string{"maxSteps": "7"})

	got := store.Get()
	if got.Temperature == nil || *got.Temperature != 0.5 || got.MaxSteps != 7 {
		t.Errorf("the settings are %+v", got)
	}
}

func TestTheStepLimitIsTwentyFiveUntilOneIsConfigured(t *testing.T) {
	store := settings.NewStore()
	if store.Get().MaxSteps != 0 || store.Get().GetMaxTokens() != 0 {
		t.Fatalf("initial store = %+v, want zero MaxSteps and GetMaxTokens", store.Get())
	}
	before := store.Get().GetMaxSteps()
	store.Configure(map[string]string{"maxSteps": "1"})
	one := store.Get().GetMaxSteps()
	store.Configure(map[string]string{"maxSteps": "5"})
	for _, invalid := range []string{"0", "-3", "bad", "999999999999999999999"} {
		store.Configure(map[string]string{"maxSteps": invalid})
	}

	if after := store.Get().GetMaxSteps(); before != 25 || one != 1 || after != 5 {
		t.Errorf("the limits are %d, %d and %d, want 25, 1 and 5", before, one, after)
	}
}

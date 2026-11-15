package settings

import "testing"

func TestTheRunConfigCarriesTheLanguageAndPromptTheHostPassed(t *testing.T) {
	store := NewStore()

	store.Update(map[string]string{"language": "Sanskrit", "prompt": "Keep the verse numbers."})

	got := store.GetRunConfig()
	if got.Language != "Sanskrit" || got.Prompt != "Keep the verse numbers." {
		t.Errorf("language %q, prompt %q", got.Language, got.Prompt)
	}
}

func TestAnUpdateWithoutLanguageAndPromptKeepsTheOnesPassedBefore(t *testing.T) {
	store := NewStore()
	store.Update(map[string]string{"language": "Sanskrit", "prompt": "Keep the verse numbers."})

	store.Update(map[string]string{"batchSize": "12"})

	got := store.GetRunConfig()
	if got.Language != "Sanskrit" || got.Prompt != "Keep the verse numbers." || got.BatchSize != 12 {
		t.Errorf("config = %+v", got)
	}
}

func TestAnUpdateWithEmptyLanguageAndPromptClearsThem(t *testing.T) {
	store := NewStore()
	store.Update(map[string]string{"language": "Sanskrit", "prompt": "Keep the verse numbers."})

	store.Update(map[string]string{"language": "", "prompt": ""})

	got := store.GetRunConfig()
	if got.Language != "" || got.Prompt != "" {
		t.Errorf("language %q, prompt %q, want both cleared", got.Language, got.Prompt)
	}
}

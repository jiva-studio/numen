package servicetype

import "testing"

func TestOnlyTheTypesOfThePluginAreKnown(t *testing.T) {
	for typeID, want := range map[string]bool{OpenRouter: true, OpenAI: true, Compatible: true, "anthropic": false, "": false} {
		if got := IsKnown(typeID); got != want {
			t.Errorf("IsKnown(%q) = %v, want %v", typeID, got, want)
		}
	}
}

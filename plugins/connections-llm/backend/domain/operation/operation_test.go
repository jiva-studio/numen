package operation

import "testing"

func TestOnlyACompletionAndAChatGiveText(t *testing.T) {
	for operation, want := range map[string]bool{Complete: true, Chat: true, Embed: false, List: false, "": false} {
		if got := IsText(operation); got != want {
			t.Errorf("IsText(%q) = %v, want %v", operation, got, want)
		}
	}
}

package loops_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
)

func TestText_IsEmptyUntilItIsSetAndKeepsTheLastText(t *testing.T) {
	text := loops.NewText()
	if got := text.Get(); got != "" {
		t.Errorf("text before it was set = %q, want none", got)
	}

	text.Set("/shared")
	text.Set("/other")

	if got := text.Get(); got != "/other" {
		t.Errorf("text = %q, want the last one", got)
	}
}

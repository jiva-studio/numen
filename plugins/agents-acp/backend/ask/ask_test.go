package ask_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-acp/backend/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/ask"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
)

func TestAskerNeedsAFinderAndALauncher(t *testing.T) {
	if _, err := ask.New(nil, process.Launcher{}); err == nil {
		t.Error("no finder was refused")
	}
	if _, err := ask.New(agents.NewRegistry(nil), nil); err == nil {
		t.Error("no launcher was refused")
	}
}

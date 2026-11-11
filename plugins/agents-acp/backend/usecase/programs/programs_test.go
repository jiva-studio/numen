package programs

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
)

func TestAnAgentHasThirtySecondsOfBackoffAfterFifteenToList(t *testing.T) {
	if listLimit != 15*time.Second || failureBackoff != 30*time.Second {
		t.Fatalf("listLimit = %v, failureBackoff = %v", listLimit, failureBackoff)
	}
}

func TestAModelWithNoNameIsTitledByItsID(t *testing.T) {
	program := createProgram(
		agents.Agent{Program: "unnamed", Title: "Unnamed", Command: []string{"/bin/unnamed"}},
		acp.Listing{Models: []acp.Model{{ID: "m-id", Name: ""}}, Current: "m-id"},
	)

	if program.Name != "acp:unnamed" || len(program.Models) != 1 || program.Models[0].Title != "m-id" || !program.Models[0].IsDefault {
		t.Errorf("program = %+v", program)
	}
}

func TestListerNeedsARegistryALauncherAndAClock(t *testing.T) {
	registry := agents.NewRegistry(nil)
	if _, err := New(nil, process.Launcher{}, clock.System{}, false); err == nil {
		t.Error("no registry was refused")
	}
	if _, err := New(registry, nil, clock.System{}, false); err == nil {
		t.Error("no launcher was refused")
	}
	if _, err := New(registry, process.Launcher{}, nil, false); err == nil {
		t.Error("no clock was refused")
	}
}

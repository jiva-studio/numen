package programs

import (
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/testsupport/fakeacp"
)

func TestMain(m *testing.M) { fakeacp.Main(m) }

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

func TestAListingThatCannotCloseItsSessionFails(t *testing.T) {
	command, _ := fakeacp.Start(t, "fail-close")
	lister, err := New(agents.NewRegistry(nil), process.Launcher{}, clock.System{}, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = lister.readListing(t.Context(), agents.Agent{Program: "fail-close", Title: "Fail close", Command: command})

	if err == nil || !strings.Contains(err.Error(), "close rejected") {
		t.Errorf("readListing with failing close = %v", err)
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

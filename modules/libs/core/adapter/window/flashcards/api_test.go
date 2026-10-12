package flashcards

import (
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// A window built over no list of work keeps one of its own, and one built over
// a list keeps that list.
func TestAWindowKeepsTheListOfWorkItWasBuiltOver(t *testing.T) {
	if NewWindow(nil).Tasking == nil {
		t.Error("a window built over no list keeps none")
	}

	tasks := task.New()
	if NewWindow(tasks).Tasking != tasks {
		t.Error("a window built over a list keeps another")
	}
}

// refusingStores is a shelf that cannot be opened.
type refusingStores struct{}

func (refusingStores) Open(domain.Vault) (port.DerivedStore, error) {
	return nil, errors.New("the shelf is not there")
}

// A session whose run cannot be opened is an internal failure.
func TestASessionWhoseRunCannotBeOpenedIsAnInternalFailure(t *testing.T) {
	isolate(t)
	api, held := newAPI(t, deck)
	api.Log = flashcards.Log{Stores: refusingStores{}}

	_, err := api.StartSession(t.Context(),
		connect.NewRequest(&v1.StartSessionRequest{Vault: string(held[0].ID)}))
	if code := connect.CodeOf(err); code != connect.CodeInternal {
		t.Errorf("starting answered %v", err)
	}
}

// A window naming no clock opens its runs by this machine's.
func TestAWindowNamingNoClockOpensARun(t *testing.T) {
	isolate(t)
	api, held := newAPI(t, deck)
	api.Now = nil

	out := startSession(t, api, held[0])
	now, err := ulid.New(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimPrefix(out.GetRun(), flashcards.Area+"/")
	if len(name) < 5 || name[:5] != now[:5] {
		t.Errorf("the run %q was not opened at this machine's time", out.GetRun())
	}
}

// isolate points every location a mutated path could fall to at the test's own
// directory.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

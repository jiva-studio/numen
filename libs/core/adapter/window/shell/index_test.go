package shell

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// unread is a vault the index does not carry.
var unread = domain.Vault{ID: "unread", Name: "Sanskrit", Path: "/vaults/Sanskrit"}

// A vault being read is work the window reports, named by the vault it is on
// and by how far it has got. A window session on a blank frame reads as broken.
func TestIndexingAVaultIsReportedAsWork(t *testing.T) {
	api := newAPI(t)
	blockCh := make(chan struct{})
	api.SetVaultIndexer(t.Context(), func(context.Context, domain.Vault) error {
		<-blockCh
		return nil
	})

	state, err := api.EnsureIndexed(t.Context(), unread)
	if err != nil || state.Status != pluginport.PluginIndexing || state.Reason != "" {
		t.Fatalf("the vault came back %+v (%v)", state, err)
	}
	testsupport.WaitFor(t, func() bool {
		for _, item := range api.Window.Tasking.List() {
			if item.Title == "Indexing the vault" && item.About == "Sanskrit" && item.IsUserStarted {
				return true
			}
		}
		return false
	})

	// The indexing ends, and the work goes with it.
	close(blockCh)
	testsupport.WaitFor(t, func() bool { return len(api.Window.Tasking.List()) == 0 })
}

// A vault the index already carries is drawn from what it holds, so indexing it
// again is work drawn only once it has lasted. An instant walk says nothing.
func TestIndexingAVaultTheIndexCarriesIsNotWorkAPersonAskedFor(t *testing.T) {
	target := domain.Vault{ID: "held", Name: "Held"}
	api := newAPI(t, target)
	blockCh := make(chan struct{})
	api.SetVaultIndexer(t.Context(), func(context.Context, domain.Vault) error {
		<-blockCh
		return nil
	})

	if _, err := api.EnsureIndexed(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	testsupport.WaitFor(t, func() bool { return len(api.Window.Tasking.List()) == 1 })

	if pos := api.Window.Tasking.List()[0]; pos.IsUserStarted {
		t.Errorf("indexing a vault the index carries came back as work asked for: %+v", pos)
	}
	close(blockCh)
}

// An indexing that failed says why in the vault's own state, and is not begun over and over by the
// counts that follow.
func TestAVaultThatCouldNotBeReadReturnsWhyAndIsLetAlone(t *testing.T) {
	api := newAPI(t)
	var tried atomic.Int64
	api.SetVaultIndexer(t.Context(), func(context.Context, domain.Vault) error {
		tried.Add(1)
		return errors.New("the folder is not there")
	})
	ask := func() pluginport.PluginIndexState {
		state, err := api.EnsureIndexed(t.Context(), unread)
		if err != nil {
			t.Fatal(err)
		}
		return state
	}

	testsupport.WaitFor(t, func() bool { return ask().Reason == "the folder is not there" })
	if state := ask(); state.Status != pluginport.PluginIndexFailed {
		t.Errorf("a vault that could not be read stands as %+v", state)
	}
	if got := tried.Load(); got != 1 {
		t.Errorf("the vault was read %d times", got)
	}

	// The vault moved underneath the window, which is what lets it be tried again.
	api.ClearIndexFailure(unread.ID)
	ask()
	testsupport.WaitFor(t, func() bool { return tried.Load() == 2 })
}

// A window that reads no vault says that the index does not carry a vault it lacks.
func TestAVaultTheIndexDoesNotCarryIsUnreadWhereNothingReadsIt(t *testing.T) {
	api := newAPI(t)

	state, err := api.EnsureIndexed(t.Context(), unread)

	if err != nil || state.Status != pluginport.PluginIndexFailed || state.Reason == "" {
		t.Errorf("a vault nothing reads stands as %+v (%v)", state, err)
	}
}

// A vault forgotten after a clean indexing is claimable again.
func TestAVaultThatWasReadIsClaimedAgainOnceForgotten(t *testing.T) {
	r := &indexRuns{}
	r.setIndexer(t.Context(), func(context.Context, domain.Vault) error { return nil })

	r.record(unread, nil)
	if read, _, _, _ := r.tryStart(unread); read != nil {
		t.Fatalf("a vault indexed clean was claimed before it was forgotten")
	}

	r.clear(unread.ID)

	read, _, _, _ := r.tryStart(unread)
	if read == nil {
		t.Fatalf("a vault forgotten after a clean indexing was not claimable again")
	}
}

// The list of work is answered whether or not the window keeps one, so the page
// has one thing to listen to.
func TestAWindowDoingNothingPriorItselfStillAnswers(t *testing.T) {
	api := newAPI(t)
	api.Window.Tasking = nil

	stream, err := newWindowClient(t, api).WatchTasks(
		t.Context(), connect.NewRequest(&v1.WatchTasksRequest{Window: testWindow}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	if !stream.Receive() {
		t.Fatalf("nothing was said: %v", stream.Err())
	}
	if response := stream.Msg().GetTasks(); len(response) != 0 {
		t.Errorf("a window doing nothing said %+v", response)
	}
}

// A vault is claimed by one indexer at a time: the claim that begins an indexing
// says it is underway, a claim while it runs says so and begins nothing, an
// indexing that failed returns why and begins nothing, and a vault indexed clean
// begins nothing and is not underway.
func TestAVaultIsClaimedForOneIndexingAtATime(t *testing.T) {
	r := &indexRuns{}
	r.setIndexer(t.Context(), func(context.Context, domain.Vault) error { return nil })
	v := domain.Vault{ID: "one"}

	read, limit, underway, reason := r.tryStart(v)
	if read == nil || limit == nil || !underway || reason != "" {
		t.Fatalf("the first claim came back %v %v %v %q", read != nil, limit != nil, underway, reason)
	}

	read, _, underway, reason = r.tryStart(v)
	if read != nil || !underway || reason != "" {
		t.Errorf("a claim while it ran came back %v %v %q", read != nil, underway, reason)
	}

	r.record(v, errors.New("the folder is not there"))
	read, _, underway, reason = r.tryStart(v)
	if read != nil || underway || reason != "the folder is not there" {
		t.Errorf("a claim after a failure came back %v %v %q", read != nil, underway, reason)
	}

	r.clear(v.ID)
	r.record(v, nil)
	read, _, underway, reason = r.tryStart(v)
	if read != nil || underway || reason != "" {
		t.Errorf("a claim after a clean indexing came back %v %v %q", read != nil, underway, reason)
	}
}

// A window that reads nothing claims nothing.
func TestAWindowThatReadsNothingClaimsNothing(t *testing.T) {
	r := &indexRuns{}
	read, _, underway, reason := r.tryStart(domain.Vault{ID: "one"})
	if read != nil || underway || reason != "" {
		t.Errorf("the claim came back %v %v %q", read != nil, underway, reason)
	}
}

// An indexing that failed stays in the list of work with why it failed.
func TestAnIndexingThatFailedStaysAsWorkWithWhy(t *testing.T) {
	api := newAPI(t)
	api.SetVaultIndexer(t.Context(), func(context.Context, domain.Vault) error {
		return errors.New("the folder is not there")
	})

	if _, err := api.EnsureIndexed(t.Context(), unread); err != nil {
		t.Fatal(err)
	}
	testsupport.WaitFor(t, func() bool {
		for _, item := range api.Window.Tasking.List() {
			if item.Title == "Indexing the vault" && item.Error == "the folder is not there" {
				return true
			}
		}
		return false
	})
}

// A vault read wakes everyone waiting to hear that something moved.
func TestAVaultReadWakesTheListeners(t *testing.T) {
	api := newAPI(t)
	api.SetVaultIndexer(t.Context(), func(context.Context, domain.Vault) error { return nil })
	moves, err := api.WatchMoves(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := api.EnsureIndexed(t.Context(), unread); err != nil {
		t.Fatal(err)
	}

	select {
	case <-moves:
	case <-time.After(testsupport.Patience):
		t.Fatal("nobody received the vault was read")
	}
}

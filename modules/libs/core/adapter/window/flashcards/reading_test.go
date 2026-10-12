package flashcards

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// A vault being read is work the window reports, named by the vault it is on
// and by how far it has got. A window session on a blank frame reads as broken.
func TestReadingAVaultIsReportedAsWork(t *testing.T) {
	api, _ := newAPI(t)
	unread := testsupport.NewVault(t, deck)
	unread.Name = "Sanskrit"
	api.Registry = registry{held: []domain.Vault{unread}}

	holding := make(chan struct{})
	api.Reading(t.Context(), func(context.Context, domain.Vault) error {
		<-holding
		return nil
	})

	if one := api.countVault(t.Context(), unread); !one.GetIsReading() || one.GetUnread() != "" {
		t.Fatalf("the vault came back %+v", one)
	}
	testsupport.WaitFor(t, func() bool {
		for _, at := range api.Window.Tasking.List() {
			if at.Doing == "Reading the vault" && at.About == "Sanskrit" && at.IsAsked {
				return true
			}
		}
		return false
	})

	// The reading ends, and the work goes with it.
	close(holding)
	testsupport.WaitFor(t, func() bool { return len(api.Window.Tasking.List()) == 0 })
}

// A vault the index already carries is drawn from what it holds, so reading it
// again is work drawn only once it has lasted. An instant walk says nothing.
func TestReadingAVaultTheIndexCarriesIsNotWorkAPersonAskedFor(t *testing.T) {
	api, held := newAPI(t, deck)

	holding := make(chan struct{})
	api.Reading(t.Context(), func(context.Context, domain.Vault) error {
		<-holding
		return nil
	})

	api.countVault(t.Context(), held[0])
	testsupport.WaitFor(t, func() bool { return len(api.Window.Tasking.List()) == 1 })

	if at := api.Window.Tasking.List()[0]; at.IsAsked {
		t.Errorf("reading a vault the index carries came back as work asked for: %+v", at)
	}
	close(holding)
}

// A reading that failed says why in the vault's own row, and is not begun over
// and over by the counts that follow.
func TestAVaultThatCouldNotBeReadSaysWhyAndIsLetAlone(t *testing.T) {
	api, _ := newAPI(t)
	unread := testsupport.NewVault(t, deck)
	api.Registry = registry{held: []domain.Vault{unread}}

	var tried atomic.Int64
	api.Reading(t.Context(), func(context.Context, domain.Vault) error {
		tried.Add(1)
		return errors.New("the folder is not there")
	})

	testsupport.WaitFor(t, func() bool {
		return api.countVault(t.Context(), unread).GetUnread() == "the folder is not there"
	})
	api.countVault(t.Context(), unread)
	if got := tried.Load(); got != 1 {
		t.Errorf("the vault was read %d times", got)
	}

	// The vault moved underneath the window, which is what lets it be tried
	// again.
	api.ClearIndexFailure(unread.ID)
	api.countVault(t.Context(), unread)
	testsupport.WaitFor(t, func() bool { return tried.Load() == 2 })
}

// A vault forgotten after a clean reading is claimable again.
func TestAVaultThatWasReadIsClaimedAgainOnceForgotten(t *testing.T) {
	r := &readings{}
	v := testsupport.NewVault(t, deck)
	r.setIndexer(t.Context(), func(context.Context, domain.Vault) error { return nil })

	r.record(v, nil)
	if read, _, _, _ := r.tryStart(v); read != nil {
		t.Fatalf("a vault read clean was claimed before it was forgotten")
	}

	r.clear(v.ID)

	read, _, _, _ := r.tryStart(v)
	if read == nil {
		t.Fatalf("a vault forgotten after a clean reading was not claimable again")
	}
}

// The list of work is answered whether or not the window keeps one, so the page
// has one thing to listen to.
func TestAWindowDoingNothingBehindItselfStillAnswers(t *testing.T) {
	api, _ := newAPI(t)
	api.Window.Tasking = nil

	stream, err := newWindowClient(t, api).WatchTasks(
		t.Context(), connect.NewRequest(&v1.WatchTasksRequest{Window: wire.Review}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	if !stream.Receive() {
		t.Fatalf("nothing was said: %v", stream.Err())
	}
	if said := stream.Msg().GetTasks(); len(said) != 0 {
		t.Errorf("a window doing nothing said %+v", said)
	}
}

// A vault is claimed by one reader at a time: the claim that begins a reading
// says it is underway, a claim while it runs says so and begins nothing, a
// reading that failed hands back why and begins nothing, and a vault read clean
// begins nothing and is not underway.
func TestAVaultIsClaimedForOneReadingAtATime(t *testing.T) {
	r := &readings{}
	r.start(t.Context(), func(context.Context, domain.Vault) error { return nil })
	v := domain.Vault{ID: "one"}

	read, under, underway, reason := r.claim(v)
	if read == nil || under == nil || !underway || reason != "" {
		t.Fatalf("the first claim came back %v %v %v %q", read != nil, under != nil, underway, reason)
	}

	read, _, underway, reason = r.claim(v)
	if read != nil || !underway || reason != "" {
		t.Errorf("a claim while it ran came back %v %v %q", read != nil, underway, reason)
	}

	r.finish(v, errors.New("the folder is not there"))
	read, _, underway, reason = r.claim(v)
	if read != nil || underway || reason != "the folder is not there" {
		t.Errorf("a claim after a failure came back %v %v %q", read != nil, underway, reason)
	}

	r.forget(v.ID)
	r.finish(v, nil)
	read, _, underway, reason = r.claim(v)
	if read != nil || underway || reason != "" {
		t.Errorf("a claim after a clean reading came back %v %v %q", read != nil, underway, reason)
	}
}

// A window that reads nothing claims nothing.
func TestAWindowThatReadsNothingClaimsNothing(t *testing.T) {
	r := &readings{}
	read, _, underway, reason := r.claim(domain.Vault{ID: "one"})
	if read != nil || underway || reason != "" {
		t.Errorf("the claim came back %v %v %q", read != nil, underway, reason)
	}
}

// A window with no index holds no vault.
func TestAWindowWithNoIndexHoldsNoVault(t *testing.T) {
	if (&API{}).hasVault(t.Context(), domain.Vault{ID: "one"}) {
		t.Error("a window with no index holds a vault")
	}
}

// A reading that failed stays in the list of work with why it failed.
func TestAReadingThatFailedStaysAsWorkWithWhy(t *testing.T) {
	isolate(t)
	api, _ := newAPI(t)
	unread := testsupport.NewVault(t, deck)
	api.Registry = registry{held: []domain.Vault{unread}}
	api.Reading(t.Context(), func(context.Context, domain.Vault) error {
		return errors.New("the folder is not there")
	})

	api.countVault(t.Context(), unread)
	testsupport.WaitFor(t, func() bool {
		for _, at := range api.Window.Tasking.List() {
			if at.Doing == "Reading the vault" && at.Error == "the folder is not there" {
				return true
			}
		}
		return false
	})
}

// A vault read wakes everyone waiting to hear that something moved.
func TestAVaultReadWakesTheListeners(t *testing.T) {
	isolate(t)
	api, _ := newAPI(t)
	unread := testsupport.NewVault(t, deck)
	api.Registry = registry{held: []domain.Vault{unread}}
	api.Reading(t.Context(), func(context.Context, domain.Vault) error { return nil })

	heard, stop := api.listeners.listen()
	t.Cleanup(stop)
	api.countVault(t.Context(), unread)

	select {
	case <-heard:
	case <-time.After(30 * time.Second):
		t.Fatal("nobody was told the vault was read")
	}
}

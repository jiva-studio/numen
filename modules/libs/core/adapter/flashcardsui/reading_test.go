package flashcardsui

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// A vault being read is work the window reports, named by the vault it is on
// and by how far it has got. A window sitting on a blank frame reads as broken.
func TestReadingAVaultIsReportedAsWork(t *testing.T) {
	api, _ := windowed(t)
	unread := testsupport.NewVault(t, deck)
	unread.Name = "Sanskrit"
	api.Registry = registry{held: []domain.Vault{unread}}

	holding := make(chan struct{})
	api.Reading(t.Context(), func(context.Context, domain.Vault, func(int64)) error {
		<-holding
		return nil
	})

	if one := api.counted(t.Context(), unread); !one.GetReading() || one.GetUnread() != "" {
		t.Fatalf("the vault came back %+v", one)
	}
	waitFor(t, func() bool {
		for _, at := range api.Tasking.List() {
			if at.Doing == "Reading the vault" && at.About == "Sanskrit" && at.Asked {
				return true
			}
		}
		return false
	})

	// The reading ends, and the work goes with it.
	close(holding)
	waitFor(t, func() bool { return len(api.Tasking.List()) == 0 })
}

// A vault the index already carries is drawn from what it holds, so reading it
// again is work drawn only once it has lasted. An instant walk says nothing.
func TestReadingAVaultTheIndexCarriesIsNotWorkAPersonAskedFor(t *testing.T) {
	api, held := windowed(t, deck)

	holding := make(chan struct{})
	api.Reading(t.Context(), func(context.Context, domain.Vault, func(int64)) error {
		<-holding
		return nil
	})

	api.counted(t.Context(), held[0])
	waitFor(t, func() bool { return len(api.Tasking.List()) == 1 })

	if at := api.Tasking.List()[0]; at.Asked {
		t.Errorf("reading a vault the index carries came back as work asked for: %+v", at)
	}
	close(holding)
}

// A reading that failed says why in the vault's own row, and is not begun over
// and over by the counts that follow.
func TestAVaultThatCouldNotBeReadSaysWhyAndIsLetAlone(t *testing.T) {
	api, _ := windowed(t)
	unread := testsupport.NewVault(t, deck)
	api.Registry = registry{held: []domain.Vault{unread}}

	var tried atomic.Int64
	api.Reading(t.Context(), func(context.Context, domain.Vault, func(int64)) error {
		tried.Add(1)
		return errors.New("the folder is not there")
	})

	waitFor(t, func() bool {
		return api.counted(t.Context(), unread).GetUnread() == "the folder is not there"
	})
	api.counted(t.Context(), unread)
	if got := tried.Load(); got != 1 {
		t.Errorf("the vault was read %d times", got)
	}

	// The vault moved underneath the window, which is what lets it be tried
	// again.
	api.Forget(unread.ID)
	api.counted(t.Context(), unread)
	waitFor(t, func() bool { return tried.Load() == 2 })
}

// The list of work is answered whether or not the window keeps one, so the page
// has one thing to listen to.
func TestAWindowDoingNothingBehindItselfStillAnswers(t *testing.T) {
	api, _ := windowed(t)
	api.Tasking = nil

	stream, err := serving(t, api).Tasks(
		t.Context(), connect.NewRequest(&v1.FlashcardsServiceTasksRequest{}),
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

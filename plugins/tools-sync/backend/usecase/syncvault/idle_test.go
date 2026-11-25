package syncvault_test

import (
	"context"
	"fmt"
	"slices"
	gosync "sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

type writeLog struct {
	mu    gosync.Mutex
	names []string
}

func (l *writeLog) add(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.names = append(l.names, name)
}

func (l *writeLog) take() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	taken := slices.Clone(l.names)
	l.names = nil
	return taken
}

type writeSpyHost struct {
	ports.Host
	log *writeLog
}

func (h writeSpyHost) Store(id domain.VaultID) ports.Store {
	return writeSpyStore{Store: h.Host.Store(id), log: h.log}
}

type writeSpyStore struct {
	ports.Store
	log *writeLog
}

func (s writeSpyStore) Write(ctx context.Context, name string, content []byte) error {
	s.log.add(name)
	return s.Store.Write(ctx, name, content)
}

func TestSyncVault_APassOverAnUnchangedVaultWritesNothingToTheStore(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Notes/One.md", "one")
	p.a.Clock.Advance(time.Minute)
	writes := &writeLog{}
	p.a.State = hoststore.New(writeSpyHost{Host: p.a.Host, log: writes})
	sync(t, p.a, source)
	sync(t, p.a, source)
	writes.take()

	sync(t, p.a, source)

	if got := writes.take(); len(got) != 0 {
		t.Errorf("an unchanged pass wrote %v, want nothing", got)
	}
}

func TestSyncVault_TwoConcurrentPassesOverOneVaultLoseNoUpdate(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	for i := range 5 {
		p.a.WriteNote(source, fmt.Sprintf("Notes/N%d.md", i), fmt.Sprintf("text %d", i))
	}
	p.a.Clock.Advance(time.Minute)

	var group gosync.WaitGroup
	for range 2 {
		group.Go(func() {
			if err := p.a.Sync(t.Context(), source); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()

	records, err := p.a.State.ReadRecords(t.Context(), source)
	if err != nil || len(records) != 5 {
		t.Fatalf("records = %d, %v, want 5", len(records), err)
	}
}

func TestSyncVault_ThePushOfManyNotesCommitsThemInBoundedBatches(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.writeMany(t, 450)
	p.a.Clock.Advance(time.Minute)

	sync(t, p.a, source)

	if got := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got < 3 {
		t.Errorf("450 notes went in %d commits, want batches of at most 200", got)
	}
	if got := len(p.getLiveHeads(t)); got != 450 {
		t.Errorf("the server holds %d notes, want 450", got)
	}
}

type commitProbe struct {
	ports.Revisions
	onCommit func()
}

func (p *commitProbe) CommitRevisions(ctx context.Context, vaultID []byte, revisions []*syncv1.Revision) (*syncv1.CommitRevisionsResponse, error) {
	p.onCommit()
	return p.Revisions.CommitRevisions(ctx, vaultID, revisions)
}

func TestSyncVault_ThePushReadsOnlyTheNotesOfOneBatchBeforeItCommitsThem(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	spy := p.a.Spy()
	p.writeMany(t, 450)
	p.a.Clock.Advance(time.Minute)
	deps := p.a.WireDeps()
	readAtFirstCommit := -1
	deps.Revisions = &commitProbe{Revisions: deps.Revisions, onCommit: func() {
		if readAtFirstCommit < 0 {
			readAtFirstCommit = spy.GetReadCalls()
		}
	}}

	if err := syncwire.NewSyncVault(deps).Execute(t.Context(), source); err != nil {
		t.Fatal(err)
	}

	if readAtFirstCommit < 0 || readAtFirstCommit > 200 {
		t.Errorf("%d notes were read before the first commit, want at most one batch of 200", readAtFirstCommit)
	}
}

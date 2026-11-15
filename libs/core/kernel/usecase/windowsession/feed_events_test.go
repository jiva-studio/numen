package windowsession

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// feedStub is a feed whose head the test moves, and the writes and plugin changes it reports.
type feedStub struct {
	mu      sync.Mutex
	head    filechange.Head
	report  func()
	plugins chan struct{}
	quiet   chan time.Time
	emitted chan domain.VaultID
}

func newFeedStub() *feedStub {
	return &feedStub{plugins: make(chan struct{}), quiet: make(chan time.Time), emitted: make(chan domain.VaultID, 8)}
}

func (s *feedStub) events() FeedEvents {
	return FeedEvents{
		WatchWrites: func(_ domain.VaultID, report func()) func() {
			s.mu.Lock()
			s.report = report
			s.mu.Unlock()
			return func() {}
		},
		WatchPlugins: func() (<-chan struct{}, func()) { return s.plugins, func() {} },
		ReadHead: func(context.Context, domain.VaultID) (filechange.Head, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			return s.head, nil
		},
		Emit:  func(v domain.Vault) { s.emitted <- v.ID },
		After: func(time.Duration) <-chan time.Time { return s.quiet },
	}
}

func (s *feedStub) write(seq int64) {
	s.mu.Lock()
	if seq > 0 {
		s.head = filechange.Head{Cursor: filechange.Cursor{FeedID: "feed-1", Seq: seq}}
	}
	report := s.report
	s.mu.Unlock()
	report()
}

func (s *feedStub) passQuiet(t *testing.T) {
	t.Helper()
	select {
	case s.quiet <- time.Time{}:
	case <-time.After(5 * time.Second):
		t.Fatal("the loop did not wait for quiet")
	}
}

func (s *feedStub) requireEmitted(t *testing.T, want int) {
	t.Helper()
	for range want {
		select {
		case id := <-s.emitted:
			if id != garden().ID {
				t.Fatalf("emitted for %s", id)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("no event was emitted")
		}
	}
	select {
	case id := <-s.emitted:
		t.Fatalf("an extra event for %s", id)
	default:
	}
}

func startFeedEvents(t *testing.T, stub *feedStub) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	run := stub.events().start(ctx, garden())
	done := make(chan struct{})
	go func() {
		defer close(done)
		run()
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func TestAVaultWithAnEmptyFeedSendsNoEventWhenItOpens(t *testing.T) {
	stub := newFeedStub()
	startFeedEvents(t, stub)
	stub.passQuiet(t)
	stub.requireEmitted(t, 0)
}

func TestAWriteThatStoredChangesSendsOneEventAfterTheQuietPeriod(t *testing.T) {
	stub := newFeedStub()
	startFeedEvents(t, stub)
	stub.passQuiet(t)

	stub.write(1)
	stub.write(2)
	stub.write(3)
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)

	stub.passQuiet(t)
	stub.requireEmitted(t, 0)
}

func TestAWriteThatStoredNoChangeSendsNoEvent(t *testing.T) {
	stub := newFeedStub()
	startFeedEvents(t, stub)
	stub.passQuiet(t)
	stub.write(4)
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)

	stub.write(0)
	stub.passQuiet(t)
	stub.requireEmitted(t, 0)
}

func TestAFeedThatHoldsChangesWhenTheVaultOpensSendsOneEvent(t *testing.T) {
	stub := newFeedStub()
	stub.head = filechange.Head{Cursor: filechange.Cursor{FeedID: "feed-1", Seq: 9}}
	startFeedEvents(t, stub)
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)
}

func TestAChangeOfThePluginsSendsTheEventAgain(t *testing.T) {
	stub := newFeedStub()
	stub.head = filechange.Head{Cursor: filechange.Cursor{FeedID: "feed-1", Seq: 9}}
	startFeedEvents(t, stub)
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)

	stub.plugins <- struct{}{}
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)
}

func TestAWindowWithoutPluginsWatchesNoFeed(t *testing.T) {
	if run := (FeedEvents{}).start(t.Context(), garden()); run != nil {
		t.Error("a window without plugins runs a feed loop")
	}
}

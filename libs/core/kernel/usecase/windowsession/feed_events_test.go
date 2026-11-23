package windowsession

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// feedStub is a feed whose head the test moves, and the writes and plugin changes it reports.
type feedStub struct {
	mu      sync.Mutex
	head    filechange.Head
	report  func()
	plugins chan struct{}
	quiet   chan time.Time
	paths   map[int64]string
	emitted chan domain.VaultID
	types   chan []domain.FileType
	failing int
	headErr int
}

func newFeedStub() *feedStub {
	return &feedStub{
		plugins: make(chan struct{}), quiet: make(chan time.Time), paths: map[int64]string{},
		emitted: make(chan domain.VaultID, 8), types: make(chan []domain.FileType, 8),
	}
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
			if s.headErr > 0 {
				s.headErr--
				return filechange.Head{}, errors.New("the index is busy")
			}
			return s.head, nil
		},
		ListFiles: func(_ context.Context, _ domain.VaultID, after, upTo int64) ([]filechange.File, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.failing > 0 {
				s.failing--
				return nil, errors.New("the index is busy")
			}
			var out []filechange.File
			for seq, path := range s.paths {
				if seq > after && seq <= upTo {
					out = append(out, filechange.File{Path: path})
				}
			}
			return out, nil
		},
		GetContentType: func(path string) domain.ContentType { return domain.ContentType("test/" + filepath.Ext(path)[1:]) },
		Emit: func(v domain.Vault, types []domain.FileType) {
			s.emitted <- v.ID
			s.types <- types
		},
		CreateTimer: func(time.Duration) <-chan time.Time { return s.quiet },
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

func (s *feedStub) record(seq int64, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.head = filechange.Head{Cursor: filechange.Cursor{FeedID: "feed-1", Seq: seq}}
	s.paths[seq] = path
}

func (s *feedStub) passQuiet(t *testing.T) {
	t.Helper()
	select {
	case s.quiet <- time.Time{}:
	case <-time.After(5 * time.Second):
		t.Fatal("the loop did not wait for quiet")
	}
}

func (s *feedStub) settle(t *testing.T) {
	t.Helper()
	for {
		select {
		case s.quiet <- time.Time{}:
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func (s *feedStub) requireEmitted(t *testing.T, want int) {
	t.Helper()
	for range want {
		select {
		case id := <-s.emitted:
			<-s.types
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
	run := stub.events().Start(ctx, garden())
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
	stub.settle(t)
	stub.requireEmitted(t, 0)
}

func TestAWriteThatStoredChangesSendsOneEventAfterTheQuietPeriod(t *testing.T) {
	stub := newFeedStub()
	startFeedEvents(t, stub)

	stub.write(1)
	stub.write(2)
	stub.write(3)
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)

	stub.settle(t)
	stub.requireEmitted(t, 0)
}

func TestAWriteThatStoredNoChangeSendsNoEvent(t *testing.T) {
	stub := newFeedStub()
	startFeedEvents(t, stub)
	stub.write(4)
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)

	stub.write(0)
	stub.passQuiet(t)
	stub.requireEmitted(t, 0)
}

func TestOpenVaultWithUnchangedFeedSendsNoEvent(t *testing.T) {
	stub := newFeedStub()
	stub.record(1, "books/river.pdf")
	startFeedEvents(t, stub)
	stub.settle(t)
	stub.requireEmitted(t, 0)

	stub.writeFile(2, "books/lake.epub")
	stub.passQuiet(t)
	stub.requireTypes(t, "test/epub")
}

func TestAFailedReadOfTheFeedIsRetriedOnceAfterTheQuietPeriod(t *testing.T) {
	stub := newFeedStub()
	stub.failing = 1
	startFeedEvents(t, stub)
	stub.write(9)
	stub.passQuiet(t)
	stub.requireEmitted(t, 0)
	stub.passQuiet(t)
	stub.requireEmitted(t, 1)
}

func TestAFailedInitialHeadReadRecoversOnNextWrite(t *testing.T) {
	stub := newFeedStub()
	stub.headErr = 1
	startFeedEvents(t, stub)

	stub.writeFile(1, "books/river.pdf")
	stub.passQuiet(t)
	stub.requireTypes(t, "test/pdf")
}

func TestAChangeOfThePluginsSendsTheEventOverTheFeed(t *testing.T) {
	stub := newFeedStub()
	stub.record(1, "books/river.pdf")
	stub.record(2, "books/lake.epub")
	startFeedEvents(t, stub)
	stub.settle(t)
	stub.requireEmitted(t, 0)

	stub.plugins <- struct{}{}
	stub.passQuiet(t)
	stub.requireTypes(t, "test/epub", "test/pdf")
}

func TestAWindowWithoutPluginsWatchesNoFeed(t *testing.T) {
	if run := (FeedEvents{}).Start(t.Context(), garden()); run != nil {
		t.Error("a window without plugins runs a feed loop")
	}
}

func (s *feedStub) writeFile(seq int64, path string) {
	s.mu.Lock()
	s.paths[seq] = path
	s.mu.Unlock()
	s.write(seq)
}

func (s *feedStub) requireTypes(t *testing.T, want ...domain.ContentType) {
	t.Helper()
	select {
	case <-s.emitted:
	case <-time.After(5 * time.Second):
		t.Fatal("no event was emitted")
	}
	content := make([]domain.ContentType, 0, len(want))
	for _, file := range <-s.types {
		content = append(content, file.Content)
	}
	if got := content; !slices.Equal(got, want) {
		t.Errorf("the event covers the types %v, want %v", got, want)
	}
}

func TestAnEventCoversTheTypesOfTheFilesItsWritesChanged(t *testing.T) {
	stub := newFeedStub()
	startFeedEvents(t, stub)

	stub.writeFile(1, "books/river.pdf")
	stub.writeFile(2, "books/lake.epub")
	stub.passQuiet(t)
	stub.requireTypes(t, "test/epub", "test/pdf")

	stub.writeFile(3, "pictures/cover.png")
	stub.passQuiet(t)
	stub.requireTypes(t, "test/png")
	stub.settle(t)
	stub.requireEmitted(t, 0)

	stub.plugins <- struct{}{}
	stub.passQuiet(t)
	stub.requireTypes(t, "test/epub", "test/pdf", "test/png")
}

func TestFeedEventIncludesTheContentAndNoteTypesOfEachChange(t *testing.T) {
	events := FeedEvents{
		ListFiles: func(context.Context, domain.VaultID, int64, int64) ([]filechange.File, error) {
			return []filechange.File{
				{Path: "article.md", NoteType: "article"},
				{Path: "old-name.md", NoteType: "article"},
				{Path: "plain.md", NoteType: domain.TypeNote},
			}, nil
		},
		GetContentType: func(string) domain.ContentType { return "text/markdown" },
	}
	loop := feedLoop{events: events, vault: garden()}
	got, err := loop.listTypes(t.Context(), 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.FileType{
		{Content: "text/markdown", Note: "article"},
		{Content: "text/markdown", Note: domain.TypeNote},
	}
	if !slices.Equal(got, want) {
		t.Errorf("changed file types = %+v, want %+v", got, want)
	}
}

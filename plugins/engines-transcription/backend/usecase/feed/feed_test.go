package feed_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/feed"
)

// recordedRuns is the transcriptions of a process: the files it started, those running, and those
// whose text the vault holds. A start of a file in failing fails.
type recordedRuns struct {
	mu      sync.Mutex
	started []string
	running map[string]bool
	done    map[string]bool
	failing map[string]bool
}

func newRuns(done ...string) *recordedRuns {
	r := &recordedRuns{running: map[string]bool{}, done: map[string]bool{}, failing: map[string]bool{}}
	for _, path := range done {
		r.done[path] = true
	}
	return r
}

func (r *recordedRuns) Start(_ context.Context, _ domain.VaultID, path string) (feed.State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case r.failing[path]:
		return 0, errors.New("the host is busy")
	case r.done[path]:
		return feed.StateDone, nil
	case r.running[path]:
		return feed.StateRunning, nil
	}
	r.running[path] = true
	r.started = append(r.started, path)
	return feed.StateStarted, nil
}

func (r *recordedRuns) IsRunning(_ context.Context, _ domain.VaultID, path string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running[path], nil
}

func (r *recordedRuns) finish(path string, isTranscribed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, path)
	r.done[path] = isTranscribed
}

func (r *recordedRuns) getStartedPaths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.started)
}

// readFeed is the feed of a vault, recording the changes each read returned.
type readFeed struct {
	*hosttest.Vault
	mu   sync.Mutex
	read []host.FileChange
}

func (r *readFeed) ListFileChanges(ctx context.Context, q host.FileChangeQuery) (host.FileChangePage, error) {
	page, err := r.Vault.ListFileChanges(ctx, q)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.read = append(r.read, page.Changes...)
	return page, err
}

func (r *readFeed) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.read))
	for _, one := range r.read {
		out = append(out, one.Path+" "+one.Event)
	}
	r.read = nil
	return out
}

type scene struct {
	vault    *hosttest.Vault
	feed     *readFeed
	store    *hosttest.Store
	runs     *recordedRuns
	follower *feed.Follower
	reported []error
}

func newScene(t *testing.T, files ...string) *scene {
	t.Helper()
	v := hosttest.New("Lectures")
	for _, path := range files {
		v.WriteFile(path, "the bytes of "+path)
	}
	s := &scene{vault: v, feed: &readFeed{Vault: v}, store: v.Store()}
	s.restart(newRuns())
	return s
}

func (s *scene) restart(runs *recordedRuns) {
	s.runs = runs
	s.follower = feed.New(func(domain.VaultID) feed.Vault { return feed.Vault{Feed: s.feed, Store: s.store} }, runs,
		func(_ string, err error) { s.reported = append(s.reported, err) })
}

func (s *scene) update(t *testing.T) {
	t.Helper()
	if err := s.follower.Execute(t.Context(), "Lectures"); err != nil {
		t.Fatal(err)
	}
}

func (s *scene) checkStartedPaths(t *testing.T, want ...string) {
	t.Helper()
	if got := s.runs.getStartedPaths(); !slices.Equal(got, want) {
		t.Errorf("started %v, want %v", got, want)
	}
}

func TestANewRecordingIsStartedOnce(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3", "notes/ohm.md")
	s.update(t)
	s.checkStartedPaths(t, "lectures/ohm.mp3")

	s.update(t)
	s.runs.finish("lectures/ohm.mp3", true)
	s.update(t)
	s.vault.WriteFile("interviews/board.wav", "an interview at the board")
	s.update(t)

	s.checkStartedPaths(t, "lectures/ohm.mp3", "interviews/board.wav")
}

func TestAChangedRecordingIsStartedAgain(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.update(t)
	s.runs.finish("lectures/ohm.mp3", false)
	s.update(t)

	s.vault.WriteFile("lectures/ohm.mp3", "the second edition")
	s.update(t)

	s.checkStartedPaths(t, "lectures/ohm.mp3", "lectures/ohm.mp3")
}

func TestAMovedRecordingIsNotTranscribedAgain(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.update(t)
	s.runs.finish("lectures/ohm.mp3", true)
	s.update(t)

	if _, err := s.vault.MoveNotes(t.Context(), []string{"lectures/ohm.mp3"}, "physics"); err != nil {
		t.Fatal(err)
	}
	s.runs.finish("physics/ohm.mp3", true)
	s.feed.take()
	s.update(t)

	s.checkStartedPaths(t, "lectures/ohm.mp3")
	if read := s.feed.take(); !slices.Equal(read, []string{"physics/ohm.mp3 file.moved"}) {
		t.Errorf("read %v, want the move alone", read)
	}
}

func TestARemovedRecordingIsNotStarted(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.runs.failing["lectures/ohm.mp3"] = true
	s.update(t)
	if len(s.reported) != 1 {
		t.Fatalf("reported %v, want the failed start", s.reported)
	}

	s.vault.Remove("lectures/ohm.mp3")
	s.runs.failing["lectures/ohm.mp3"] = false
	s.update(t)
	s.restart(newRuns())
	s.update(t)

	s.checkStartedPaths(t)
}

func TestABatchAppliedTwiceStartsNothingMore(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.update(t)
	cursor, err := s.store.Read(t.Context(), feed.CursorFile)
	if err != nil {
		t.Fatal(err)
	}
	s.vault.WriteFile("lectures/faraday.mp3", "the bytes of Faraday")
	s.update(t)

	s.store.WriteFile(feed.CursorFile, cursor)
	s.update(t)
	s.runs.finish("lectures/faraday.mp3", true)
	s.store.WriteFile(feed.CursorFile, cursor)
	s.update(t)

	s.checkStartedPaths(t, "lectures/ohm.mp3", "lectures/faraday.mp3")
}

func TestAnUpdateThatFailedIsRetriedOnTheNextUpdate(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.update(t)
	s.vault.WriteFile("lectures/faraday.mp3", "the bytes of Faraday")
	s.store.Refuse(feed.CursorFile, errors.New("disk full"))

	if err := s.follower.Execute(t.Context(), "Lectures"); err == nil {
		t.Fatal("an update whose cursor was not written succeeded")
	}
	s.store.Refuse(feed.CursorFile, nil)
	s.update(t)
	s.update(t)

	s.checkStartedPaths(t, "lectures/ohm.mp3", "lectures/faraday.mp3")
}

func TestAResetStartsOnlyTheRecordingsWithoutText(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3", "lectures/faraday.mp3")
	s.restart(newRuns("lectures/ohm.mp3"))
	s.update(t)
	s.checkStartedPaths(t, "lectures/faraday.mp3")
	s.runs.finish("lectures/faraday.mp3", true)

	s.vault.ResetFeed()
	s.update(t)

	s.checkStartedPaths(t, "lectures/faraday.mp3")
}

func TestARestartReadsOnlyTheChangesAfterTheCursor(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.update(t)
	s.runs.finish("lectures/ohm.mp3", true)
	s.update(t)

	s.restart(newRuns("lectures/ohm.mp3"))
	s.vault.WriteFile("lectures/faraday.mp3", "the bytes of Faraday")
	s.feed.take()
	s.update(t)

	s.checkStartedPaths(t, "lectures/faraday.mp3")
	if read := s.feed.take(); !slices.Equal(read, []string{"lectures/faraday.mp3 file.added"}) {
		t.Errorf("read %v, want the new recording alone", read)
	}
}

func TestARestartStartsAgainATranscriptionThatDidNotEnd(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.update(t)

	s.restart(newRuns())
	s.update(t)

	s.checkStartedPaths(t, "lectures/ohm.mp3")
}

func TestATranscriptionThatEndedWithoutTextIsNotStartedAgain(t *testing.T) {
	s := newScene(t, "lectures/ohm.mp3")
	s.update(t)
	s.runs.finish("lectures/ohm.mp3", false)

	s.update(t)
	s.vault.WriteFile("interviews/board.wav", "an interview at the board")
	s.update(t)

	s.checkStartedPaths(t, "lectures/ohm.mp3", "interviews/board.wav")
}

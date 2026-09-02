package container

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// deaf is a transcriber that hears whatever it was told to hear: a file it
// cannot open, or a recording carrying nothing.
type deaf struct {
	why error

	mu     sync.Mutex
	opened int
}

func (d *deaf) Transcription() port.Transcription {
	return port.Transcription{Model: "deaf", Segmenter: "none"}
}

func (d *deaf) Open(context.Context, []byte) (port.Recording, error) {
	d.mu.Lock()
	d.opened++
	d.mu.Unlock()
	if d.why != nil {
		return nil, d.why
	}
	return quiet{}, nil
}

func (d *deaf) Hear(context.Context, port.Audio) (string, error) { return "", nil }

func (d *deaf) Close() error { return nil }

// times is how many recordings this transcriber was handed.
func (d *deaf) times() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opened
}

// quiet is a recording that carries no speech.
type quiet struct{}

func (quiet) Length() int { return 60_000 }

func (quiet) Speech(context.Context, int, int) ([]port.Audio, error) { return nil, nil }

func (quiet) Close() error { return nil }

// listens is a Transcribing hearing through one transcriber, in a vault holding
// the recordings named.
func listens(t *testing.T, by *deaf, recordings ...string) (*Transcribing, domain.Vault) {
	t.Helper()
	root := t.TempDir()
	for _, path := range recordings {
		at := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte("not a recording"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	held := &Transcribing{
		cfg:      Config{ServiceDir: ".numen"},
		tasks:    task.New(),
		ready:    func() bool { return true },
		answered: map[string]bool{},
		Cut:      func(context.Context, domain.Vault, string) error { return nil },
		open: func(context.Context, func(string, int64, int64)) (port.Transcriber, func() error, error) {
			return by, by.Close, nil
		},
	}
	return held, domain.Vault{ID: "v", Path: root}
}

// held is an index holding recordings, none of which stands on a text.
type held struct {
	port.SourceQueries
	recordings []string
}

func (h held) Fingerprints(
	_ context.Context, _ string, _ domain.SourceKind,
) (map[string]domain.FileRef, error) {
	out := map[string]domain.FileRef{}
	for _, path := range h.recordings {
		out[path] = domain.FileRef{Path: path}
	}
	return out, nil
}

func (h held) Recognised(
	_ context.Context, _ string, _ domain.SourceKind,
) ([]port.Recognised, error) {
	return nil, nil
}

// A recording nothing here can open is a recording that has answered, and a
// queue that came back to it would spend the machine on the same answer for as
// long as the application ran.
func TestARecordingThatWillNotOpenIsHandedOverOnce(t *testing.T) {
	by := &deaf{why: errors.New("not a container anything here decodes")}
	listening, v := listens(t, by, "talks/one.mp3")
	known := held{recordings: []string{"talks/one.mp3"}}

	for range 3 {
		listening.round(t.Context(), known, v)
	}
	if got := by.times(); got != 1 {
		t.Errorf("the recording was handed over %d times", got)
	}
}

// Silence is an answer too.
func TestARecordingCarryingNoSpeechIsHandedOverOnce(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3")
	known := held{recordings: []string{"talks/one.mp3"}}

	for range 3 {
		listening.round(t.Context(), known, v)
	}
	if got := by.times(); got != 1 {
		t.Errorf("the recording was handed over %d times", got)
	}
}

// The queue is a question asked of the index: what it holds, less what stands
// on a text a producer made.
func TestARecordingStandingOnATextIsNotOwed(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by)
	owed := listening.owing(t.Context(), heard{"talks/one.mp3"}, v)
	if len(owed) != 0 {
		t.Errorf("the queue owes %v", owed)
	}
}

// heard is an index holding one recording that stands on what a model wrote.
type heard struct{ path string }

func (h heard) Fingerprints(
	_ context.Context, _ string, _ domain.SourceKind,
) (map[string]domain.FileRef, error) {
	return map[string]domain.FileRef{h.path: {Path: h.path}}, nil
}

func (h heard) Recognised(
	_ context.Context, _ string, _ domain.SourceKind,
) ([]port.Recognised, error) {
	return []port.Recognised{{Path: h.path, From: "asr", Hash: "x"}}, nil
}

func (h heard) Under(context.Context, string, string) ([]domain.FileRef, error) { return nil, nil }

func (h heard) Unchunked(context.Context, string, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (h heard) ByOtherRecipe(
	context.Context, string, domain.SourceKind, []string, int,
) ([]string, error) {
	return nil, nil
}

func (h heard) Reading(context.Context, string, string) (port.Recognised, bool, error) {
	return port.Recognised{}, false, nil
}

// One at a time: the models hold a worker each.
func TestASecondRecordingIsNotTakenWhileOneIsBeingHeard(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3", "talks/two.mp3")

	going := make(chan struct{})
	listening.open = func(
		context.Context, func(string, int64, int64),
	) (port.Transcriber, func() error, error) {
		<-going
		return nil, nil, errors.New("nothing to listen with")
	}

	if !listening.Start(v, "talks/one.mp3") {
		t.Fatal("the first recording was not taken")
	}
	if listening.Start(v, "talks/two.mp3") {
		t.Error("a second recording was taken while one was being heard")
	}
	close(going)
	listening.Wait()

	if !listening.Start(v, "talks/two.mp3") {
		t.Error("nothing was taken once the first transcription was over")
	}
	listening.Wait()
}

// One heavy run on a machine: a scan being read holds the turn, and a
// transcription waits for it and says so.
func TestARecordingWaitsForTheTurnAScanHolds(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3")

	held, err := heavy.take(t.Context(), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	listening.Start(v, "talks/one.mp3")
	waited := false
	for range 200 {
		for _, at := range listening.tasks.List() {
			if at.Doing == "Waiting for a turn at the models" {
				waited = true
			}
		}
		if waited {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	held()
	listening.Wait()

	if !waited {
		t.Error("the transcription was not shown waiting for its turn")
	}
	if got := by.times(); got != 1 {
		t.Errorf("the recording was heard %d times once the turn was free", got)
	}
}

// sized is an index holding recordings of a given size.
type sized struct {
	port.SourceQueries
	recordings map[string]int64
}

func (s sized) Fingerprints(
	_ context.Context, _ string, _ domain.SourceKind,
) (map[string]domain.FileRef, error) {
	out := map[string]domain.FileRef{}
	for path, size := range s.recordings {
		out[path] = domain.FileRef{Path: path, Size: size}
	}
	return out, nil
}

func (sized) Recognised(
	_ context.Context, _ string, _ domain.SourceKind,
) ([]port.Recognised, error) {
	return nil, nil
}

// A folder of albums is days of a machine, and nobody put them there to be read.
// What the queue takes on its own stops at a size; the hand still asks for
// anything.
func TestALargeRecordingIsLeftForTheHand(t *testing.T) {
	held, v := listens(t, &deaf{}, "talk.mp3", "album.flac")
	held.cfg.TranscribesUnder = 10 << 20

	known := sized{recordings: map[string]int64{
		"talk.mp3":   5 << 20,
		"album.flac": 400 << 20,
	}}
	owed := held.owing(t.Context(), known, v)
	if len(owed) != 1 || owed[0] != "talk.mp3" {
		t.Errorf("the queue took %v", owed)
	}

	// Naming no size takes whatever the vault holds.
	held.cfg.TranscribesUnder = 0
	if owed := held.owing(t.Context(), known, v); len(owed) != 2 {
		t.Errorf("with no limit the queue took %v", owed)
	}
}

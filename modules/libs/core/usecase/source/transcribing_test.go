package source

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// The vault a queue reads and writes through in these tests, on the disk a test
// was given.
var (
	vaultReaders  = filesystem.VaultReaders{Options: filesystem.Options{ServiceDir: ".numen"}}
	derivedStores = filesystem.DerivedStores{
		Options: filesystem.Options{ServiceDir: ".numen"},
		Area:    filesystem.OCRDir,
		Areas:   []string{filesystem.SpeechDir},
	}
)

// deaf is a transcriber that hears whatever it was told to hear: a file it
// cannot open, or a recording carrying nothing.
type deaf struct {
	why error

	mu     sync.Mutex
	opened int
}

func (d *deaf) Transcription() port.TranscriptionModel {
	return port.TranscriptionModel{Model: "deaf", Segmenter: "none"}
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

func (d *deaf) Transcribe(context.Context, port.Audio) (string, error) { return "", nil }

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
		// Bytes of its own: a recording is kept under the hash of what it
		// holds, and two files holding the same thing are one recording.
		if err := os.WriteFile(at, []byte("not a recording: "+path), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	held := NewTranscribing(context.Background(), Transcriptions{
		Readers: vaultReaders,
		Derived: derivedStores,
		Tasks:   task.New(),
		Ready:   func() bool { return true },
		Open: func(context.Context, func(string, int64, int64)) (port.Transcriber, func() error, error) {
			return by, by.Close, nil
		},
	})
	held.Cut = func(context.Context, domain.Vault, string) error { return nil }
	return held, domain.Vault{ID: "v", Path: root}
}

// unheard is an index holding recordings, none of which stands on a text.
type unheard struct {
	port.SourceQueries
	recordings []string
}

func (h unheard) Fingerprints(
	_ context.Context, _ string, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	out := map[string]domain.Fingerprint{}
	for _, path := range h.recordings {
		out[path] = domain.Fingerprint{Path: path}
	}
	return out, nil
}

func (h unheard) Recognised(
	_ context.Context, _ string, _ domain.SourceKind,
) ([]port.SourceText, error) {
	return nil, nil
}

// A recording nothing here can open is a recording that has answered, and a
// queue that came back to it would spend the machine on the same answer for as
// long as the application ran.
func TestARecordingThatWillNotOpenIsHandedOverOnce(t *testing.T) {
	by := &deaf{why: errors.New("not a container anything here decodes")}
	listening, v := listens(t, by, "talks/one.mp3")
	known := unheard{recordings: []string{"talks/one.mp3"}}

	for range 3 {
		listening.round(t.Context(), known, v)
	}
	if got := by.times(); got != 1 {
		t.Errorf("the recording was handed over %d times", got)
	}
}

// Getting the models is a step of its own, and it is named for what it is. A
// row that calls it by the name of the work that follows leaves a person
// watching a recording nothing has listened to yet.
func TestTheModelsAreGotUnderTheirOwnNameBeforeARecordingIsHeard(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3")
	known := unheard{recordings: []string{"talks/one.mp3"}}

	var while []task.Task
	open := listening.with.Open
	listening.with.Open = func(
		ctx context.Context,
		tell func(string, int64, int64),
	) (port.Transcriber, func() error, error) {
		while = listening.with.Tasks.List()
		return open(ctx, tell)
	}

	listening.round(t.Context(), known, v)

	if len(while) != 1 {
		t.Fatalf("the list holds %d pieces of work while the models arrive: %+v", len(while), while)
	}
	if while[0].Doing != "Fetching models" {
		t.Errorf("getting the models is shown as %q", while[0].Doing)
	}
	// Nothing has come down, so there is no share of it to draw.
	if while[0].Total != 0 {
		t.Errorf("a step that has counted nothing is drawn against %d", while[0].Total)
	}
}

// Silence is an answer too.
func TestARecordingCarryingNoSpeechIsHandedOverOnce(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3")
	known := unheard{recordings: []string{"talks/one.mp3"}}

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
	owed := listening.owing(t.Context(), recognised{"talks/one.mp3"}, v)
	if len(owed) != 0 {
		t.Errorf("the queue owes %v", owed)
	}
}

// recognised is an index holding one recording that stands on what a model wrote.
type recognised struct{ path string }

func (h recognised) Fingerprints(
	_ context.Context, _ string, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	return map[string]domain.Fingerprint{h.path: {Path: h.path}}, nil
}

func (h recognised) Recognised(
	_ context.Context, _ string, _ domain.SourceKind,
) ([]port.SourceText, error) {
	return []port.SourceText{{Path: h.path, Producer: "asr", Hash: "x"}}, nil
}

func (h recognised) Under(context.Context, string, string) ([]domain.Fingerprint, error) {
	return nil, nil
}

func (h recognised) Unchunked(context.Context, string, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (h recognised) ByOtherRecipe(
	context.Context, string, domain.SourceKind, []string, int,
) ([]string, error) {
	return nil, nil
}

func (h recognised) Reading(context.Context, string, string) (port.SourceText, bool, error) {
	return port.SourceText{}, false, nil
}

// One at a time: the models hold a worker each. A recording named while one is
// being heard waits its turn and is heard when the turn is free, and one named
// twice waits once.
func TestARecordingNamedWhileOneIsBeingHeardWaitsItsTurn(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3", "talks/two.mp3")

	hearing, going := make(chan struct{}, 1), make(chan struct{})
	first := true
	listening.with.Open = func(
		context.Context, func(string, int64, int64),
	) (port.Transcriber, func() error, error) {
		if first {
			first = false
			hearing <- struct{}{}
			<-going
		}
		return by, by.Close, nil
	}

	if got := listening.Start(v, "talks/one.mp3"); got != port.Began {
		t.Fatalf("the first recording was not heard: %v", got)
	}
	// The first recording is out of the line and being heard.
	<-hearing

	if got := listening.Start(v, "talks/two.mp3"); got != port.Queued {
		t.Errorf("a second recording was heard while one was being heard: %v", got)
	}
	if got := listening.Start(v, "talks/two.mp3"); got != port.Queued {
		t.Errorf("the same recording named again: %v", got)
	}
	if listening.Waiting() != 1 {
		t.Errorf("%d recordings are in line", listening.Waiting())
	}

	close(going)
	listening.Wait()

	if got := by.times(); got != 2 {
		t.Errorf("%d recordings were heard", got)
	}
	if listening.Waiting() != 0 {
		t.Errorf("%d recordings were left in line", listening.Waiting())
	}
}

// A recording named by hand is heard whatever the queue would leave alone. A
// person naming a file has said that this file is worth the machine's time.
func TestALargeRecordingIsHeardWhenItIsAskedForByHand(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "album.flac")
	listening.with.Unasked = 10 << 20

	known := sized{recordings: map[string]int64{"album.flac": 400 << 20}}
	if owed := listening.owing(t.Context(), known, v); len(owed) != 0 {
		t.Fatalf("the queue took %v on its own", owed)
	}

	if got := listening.Start(v, "album.flac"); got != port.Began {
		t.Fatalf("the recording was not heard: %v", got)
	}
	listening.Wait()

	if got := by.times(); got != 1 {
		t.Errorf("the recording was heard %d times", got)
	}
}

// An installation that listens to nothing on its own still listens to what is
// asked for: nothing here calls Queue.
func TestAnInstallationListeningToNothingStillHearsWhatIsAsked(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3")

	if got := listening.Start(v, "talks/one.mp3"); got != port.Began {
		t.Fatalf("the recording was not heard: %v", got)
	}
	listening.Wait()

	if got := by.times(); got != 1 {
		t.Errorf("the recording was heard %d times", got)
	}
}

// A round hands over what a person named before what the vault owes on its own.
func TestARecordingNamedIsHeardBeforeTheOnesNobodyAskedFor(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/owed.mp3", "talks/named.mp3")

	var mu sync.Mutex
	var order []string
	listening.Cut = func(_ context.Context, _ domain.Vault, path string) error {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, path)
		return nil
	}
	listening.queue.add(v, "talks/named.mp3")

	listening.round(t.Context(), unheard{recordings: []string{"talks/owed.mp3"}}, v)

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "talks/named.mp3" {
		t.Errorf("the recordings were heard in the order %v", order)
	}
}

// One run holds the models on a machine: a scan being read holds them, and a
// transcription waits for them and says so.
func TestARecordingWaitsForTheModelsAScanHolds(t *testing.T) {
	by := &deaf{}
	listening, v := listens(t, by, "talks/one.mp3")

	held, err := models.acquire(t.Context(), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	listening.Start(v, "talks/one.mp3")
	waited := false
	for range 200 {
		for _, at := range listening.with.Tasks.List() {
			if at.Doing == "Waiting for the models" {
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
		t.Error("the transcription was not shown waiting for the models")
	}
	if got := by.times(); got != 1 {
		t.Errorf("the recording was heard %d times once the models were free", got)
	}
}

// sized is an index holding recordings of a given size.
type sized struct {
	port.SourceQueries
	recordings map[string]int64
}

func (s sized) Fingerprints(
	_ context.Context, _ string, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	out := map[string]domain.Fingerprint{}
	for path, size := range s.recordings {
		out[path] = domain.Fingerprint{Path: path, Size: size}
	}
	return out, nil
}

func (sized) Recognised(
	_ context.Context, _ string, _ domain.SourceKind,
) ([]port.SourceText, error) {
	return nil, nil
}

// A folder of albums is days of a machine, and nobody put them there to be read.
// What the queue takes on its own stops at a size; the hand still asks for
// anything.
func TestALargeRecordingIsLeftForTheHand(t *testing.T) {
	held, v := listens(t, &deaf{}, "talk.mp3", "album.flac")
	held.with.Unasked = 10 << 20

	known := sized{recordings: map[string]int64{
		"talk.mp3":   5 << 20,
		"album.flac": 400 << 20,
	}}
	owed := held.owing(t.Context(), known, v)
	if len(owed) != 1 || owed[0] != "talk.mp3" {
		t.Errorf("the queue took %v", owed)
	}

	// Naming no size takes whatever the vault holds.
	held.with.Unasked = 0
	if owed := held.owing(t.Context(), known, v); len(owed) != 2 {
		t.Errorf("with no limit the queue took %v", owed)
	}
}

// A recording is transcribed through whatever runtime this process has, opened
// when there is a recording to transcribe.
func TestARecordingIsTranscribedThroughARuntimeOpenedNow(t *testing.T) {
	by := &deaf{}
	held, v := listens(t, by, "talk.mp3")

	var opened, given int
	held.with.Open = func(context.Context, func(string, int64, int64)) (port.Transcriber, func() error, error) {
		opened++
		return by, func() error { given++; return by.Close() }, nil
	}

	if err := held.one(t.Context(), v, "talk.mp3", true); err != nil {
		t.Fatalf("the recording was not transcribed: %v", err)
	}
	if opened != 1 {
		t.Errorf("what was missing was fetched %d times", opened)
	}
	if given != 1 {
		t.Error("what was opened was not given back")
	}
	if by.times() != 1 {
		t.Errorf("the recording was handed over %d times", by.times())
	}
}

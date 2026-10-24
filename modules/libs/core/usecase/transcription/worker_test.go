package transcription

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/transcriptionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/windowport"
)

// deaf is a transcriber that hears whatever it was told to hear: a file it
// cannot open, or a recording carrying nothing.
type deaf struct {
	why error

	mu     sync.Mutex
	opened int
}

func (d *deaf) GetTranscriptionModel() transcriptionport.TranscriptionModel {
	return transcriptionport.TranscriptionModel{Model: "deaf", Segmenter: "none"}
}

func (d *deaf) Open(context.Context, []byte) (transcriptionport.Recording, error) {
	d.mu.Lock()
	d.opened++
	d.mu.Unlock()
	if d.why != nil {
		return nil, d.why
	}
	return quiet{}, nil
}

func (d *deaf) Transcribe(context.Context, transcriptionport.Audio) (string, error) { return "", nil }

func (d *deaf) Close() error { return nil }

// times is how many recordings this transcriber was handed.
func (d *deaf) times() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opened
}

// quiet is a recording that carries no speech.
type quiet struct{}

func (quiet) GetDuration() int { return 60_000 }

func (quiet) GetSegments(context.Context, int, int) ([]transcriptionport.Audio, error) {
	return nil, nil
}

func (quiet) Close() error { return nil }

// newTranscriptionWorker creates a Worker hearing through one transcriber, in a vault holding
// the recordings named.
func newTranscriptionWorker(t *testing.T, delta *deaf, recordings ...string) (*Worker, domain.Vault) {
	t.Helper()
	shelved := newLibrary()
	for _, path := range recordings {
		// Bytes of its own: a recording is kept under the hash of what it
		// holds, and two files holding the same thing are one recording.
		shelved.hold(path, domain.KindRecording, []byte("not a recording: "+path), 1)
	}
	v := domain.Vault{ID: "v", Path: "/vault"}
	target := NewWorker(t.Context(), Config{
		Readers: vaults{v.ID: shelved},
		Derived: fakeStores{newShelf()},
		Tasks:   run.NewTasks(),
		Wait:    waitForContext,
		Delay:   neverDelay,
		Runtime: Runtime{
			Ready: func() bool { return true },
			Open: func(context.Context, func(string, int64, int64)) (transcriptionport.Transcriber, func() error, error) {
				return delta, delta.Close, nil
			},
		},
	})
	target.Cut = func(context.Context, domain.Vault, string) error { return nil }
	return target, v
}

// unheard is an index holding recordings, none of which relies on a text.
type unheard struct {
	sourceport.SourceQueries
	recordings []string
}

func (h unheard) GetFingerprints(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	out := map[string]domain.Fingerprint{}
	for _, path := range h.recordings {
		out[path] = domain.Fingerprint{Path: path}
	}
	return out, nil
}

func (h unheard) GetRecognisedSources(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) ([]sourceport.SourceText, error) {
	return nil, nil
}

// A recording nothing here can open is a recording that has answered, and a
// queue that came back to it would spend the machine on the same answer for as
// long as the application ran.
func TestARecordingThatWillNotOpenIsHandedOverOnce(t *testing.T) {
	delta := &deaf{why: errors.New("not a container anything here decodes")}
	listening, v := newTranscriptionWorker(t, delta, "talks/one.mp3")
	recognized := unheard{recordings: []string{"talks/one.mp3"}}

	for range 3 {
		listening.round(t.Context(), recognized, v)
	}
	if got := delta.times(); got != 1 {
		t.Errorf("the recording was handed over %d times", got)
	}
}

// Getting the models is a step of its own, and it is named for what it is. A
// row that calls it by the name of the work that follows leaves a person
// watching a recording nothing has listened to yet.
func TestTheModelsAreGotBelowTheirOwnNameBeforeARecordingIsHeard(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, "talks/one.mp3")
	recognized := unheard{recordings: []string{"talks/one.mp3"}}

	var while []run.Task
	open := listening.with.Runtime.Open
	listening.with.Runtime.Open = func(
		ctx context.Context,
		tell func(string, int64, int64),
	) (transcriptionport.Transcriber, func() error, error) {
		while = listening.with.Tasks.List()
		return open(ctx, tell)
	}

	listening.round(t.Context(), recognized, v)

	if len(while) != 1 {
		t.Fatalf("the list holds %d pieces of work while the models arrive: %+v", len(while), while)
	}
	if while[0].Title != "Fetching models" {
		t.Errorf("getting the models is shown as %q", while[0].Title)
	}
	// Nothing has come down, so there is no share of it to draw.
	if while[0].Total != 0 {
		t.Errorf("a step that has counted nothing is drawn against %d", while[0].Total)
	}
}

// Silence is an answer too.
func TestARecordingCarryingNoSpeechIsHandedOverOnce(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, "talks/one.mp3")
	recognized := unheard{recordings: []string{"talks/one.mp3"}}

	for range 3 {
		listening.round(t.Context(), recognized, v)
	}
	if got := delta.times(); got != 1 {
		t.Errorf("the recording was handed over %d times", got)
	}
}

// The queue is a question asked of the index: what it holds, less what stands
// on a text a producer made.
func TestARecordingActiveOnATextIsNotOwed(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta)
	owed := listening.getUntranscribed(t.Context(), recognised{"talks/one.mp3"}, v)
	if len(owed) != 0 {
		t.Errorf("the queue owes %v", owed)
	}
}

// recognised is an index holding one recording that relies on what a model wrote.
type recognised struct{ path string }

func (h recognised) GetFingerprints(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	return map[string]domain.Fingerprint{h.path: {Path: h.path}}, nil
}

func (h recognised) GetRecognisedSources(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) ([]sourceport.SourceText, error) {
	return []sourceport.SourceText{{
		Fingerprint: domain.Fingerprint{Path: h.path}, Producer: "asr", Hash: text.Fingerprint([]byte("not a recording: " + h.path)),
	}}, nil
}

type heard struct {
	path string
	hash string
}

func (h heard) GetFingerprints(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	return map[string]domain.Fingerprint{h.path: {Path: h.path}}, nil
}

func (h heard) GetRecognisedSources(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) ([]sourceport.SourceText, error) {
	return []sourceport.SourceText{{
		Fingerprint: domain.Fingerprint{Path: h.path}, Producer: "asr", Hash: h.hash,
	}}, nil
}

func (h heard) GetSourcesUnder(context.Context, domain.VaultID, string) ([]domain.Fingerprint, error) {
	return nil, nil
}

func (h heard) GetUnchunkedSources(context.Context, domain.VaultID, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (h heard) GetSourcesWithOtherRecipe(
	context.Context, domain.VaultID, domain.SourceKind, []string, int,
) ([]string, error) {
	return nil, nil
}

func (h heard) GetSourceText(context.Context, domain.VaultID, string) (sourceport.SourceText, bool, error) {
	return sourceport.SourceText{}, false, nil
}

func (h recognised) GetSourcesUnder(context.Context, domain.VaultID, string) ([]domain.Fingerprint, error) {
	return nil, nil
}

func (h recognised) GetUnchunkedSources(context.Context, domain.VaultID, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (h recognised) GetSourcesWithOtherRecipe(
	context.Context, domain.VaultID, domain.SourceKind, []string, int,
) ([]string, error) {
	return nil, nil
}

func (h recognised) GetSourceText(context.Context, domain.VaultID, string) (sourceport.SourceText, bool, error) {
	return sourceport.SourceText{}, false, nil
}

// One at a time: the models hold a worker each. A recording named while one is
// being heard waits its turn and is heard when the turn is free, and one named
// twice waits once.
func TestARecordingNamedWhileOneIsBeingHeardWaitsItsTurn(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, "talks/one.mp3", "talks/two.mp3")

	hearing, going := make(chan struct{}, 1), make(chan struct{})
	first := true
	listening.with.Runtime.Open = func(
		context.Context, func(string, int64, int64),
	) (transcriptionport.Transcriber, func() error, error) {
		if first {
			first = false
			hearing <- struct{}{}
			<-going
		}
		return delta, delta.Close, nil
	}

	if got := listening.Start(v, "talks/one.mp3"); got != windowport.Began {
		t.Fatalf("the first recording was not heard: %v", got)
	}
	// The first recording is out of the line and being heard.
	<-hearing

	if got := listening.Start(v, "talks/two.mp3"); got != windowport.Queued {
		t.Errorf("a second recording was heard while one was being heard: %v", got)
	}
	if got := listening.Start(v, "talks/two.mp3"); got != windowport.Queued {
		t.Errorf("the same recording named again: %v", got)
	}
	if listening.CountWaiting() != 1 {
		t.Errorf("%d recordings are in line", listening.CountWaiting())
	}

	close(going)
	listening.Wait()

	if got := delta.times(); got != 2 {
		t.Errorf("%d recordings were heard", got)
	}
	if listening.CountWaiting() != 0 {
		t.Errorf("%d recordings were left in line", listening.CountWaiting())
	}
}

// A recording named by hand is heard whatever the queue would leave alone. A
// person naming a file has said that this file is worth the machine's time.
func TestALargeRecordingIsHeardWhenItIsAskedForByHand(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, "album.flac")
	listening.with.Unasked = 10 << 20

	recognized := sized{recordings: map[string]int64{"album.flac": 400 << 20}}
	if owed := listening.getUntranscribed(t.Context(), recognized, v); len(owed) != 0 {
		t.Fatalf("the queue took %v on its own", owed)
	}

	if got := listening.Start(v, "album.flac"); got != windowport.Began {
		t.Fatalf("the recording was not heard: %v", got)
	}
	listening.Wait()

	if got := delta.times(); got != 1 {
		t.Errorf("the recording was heard %d times", got)
	}
}

// An installation that listens to nothing on its own still listens to what is
// asked for: nothing here calls Queue.
func TestAnInstallationListeningToNothingStillHearsWhatIsAsked(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, "talks/one.mp3")

	if got := listening.Start(v, "talks/one.mp3"); got != windowport.Began {
		t.Fatalf("the recording was not heard: %v", got)
	}
	listening.Wait()

	if got := delta.times(); got != 1 {
		t.Errorf("the recording was heard %d times", got)
	}
}

// A round hands over what a person named before what the vault owes on its own.
func TestARecordingNamedIsHeardBeforeTheOnesNobodyAskedFor(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, "talks/owed.mp3", "talks/named.mp3")

	var mu sync.Mutex
	var order []string
	listening.Cut = func(_ context.Context, _ domain.Vault, path string) error {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, path)
		return nil
	}
	listening.line.Add(v, "talks/named.mp3")

	listening.round(t.Context(), unheard{recordings: []string{"talks/owed.mp3"}}, v)

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "talks/named.mp3" {
		t.Errorf("the recordings were heard in the order %v", order)
	}
}

// One run holds the models on a machine: a scan being read holds them, and a
// transcription waits for them and says so.
func TestARecordingWaitsForTheModelsAScanContains(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, "talks/one.mp3")

	target, err := listening.with.Models.Acquire(t.Context(), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	listening.Start(v, "talks/one.mp3")
	waited := false
	for range 200 {
		for _, item := range listening.with.Tasks.List() {
			if item.Title == "Waiting for the models" {
				waited = true
			}
		}
		if waited {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	target()
	listening.Wait()

	if !waited {
		t.Error("the transcription was not shown waiting for the models")
	}
	if got := delta.times(); got != 1 {
		t.Errorf("the recording was heard %d times once the models were free", got)
	}
}

// sized is an index holding recordings of a given size.
type sized struct {
	sourceport.SourceQueries
	recordings map[string]int64
}

func (s sized) GetFingerprints(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	out := map[string]domain.Fingerprint{}
	for path, size := range s.recordings {
		out[path] = domain.Fingerprint{Path: path, Size: size}
	}
	return out, nil
}

func (sized) GetRecognisedSources(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) ([]sourceport.SourceText, error) {
	return nil, nil
}

// A folder of albums is days of a machine, and nobody put them there to be read.
// What the queue takes on its own stops at a size; the hand still asks for
// anything.
func TestALargeRecordingIsLeftForTheHand(t *testing.T) {
	target, v := newTranscriptionWorker(t, &deaf{}, "talk.mp3", "album.flac")
	target.with.Unasked = 10 << 20

	recognized := sized{recordings: map[string]int64{
		"talk.mp3":   5 << 20,
		"album.flac": 400 << 20,
	}}
	owed := target.getUntranscribed(t.Context(), recognized, v)
	if len(owed) != 1 || owed[0] != "talk.mp3" {
		t.Errorf("the queue took %v", owed)
	}

	// Naming no size takes whatever the vault holds.
	target.with.Unasked = 0
	if owed := target.getUntranscribed(t.Context(), recognized, v); len(owed) != 2 {
		t.Errorf("with no limit the queue took %v", owed)
	}
}

// A recording is transcribed through whatever runtime this process has, opened
// when there is a recording to transcribe.
func TestARecordingIsTranscribedThroughARuntimeOpenedNow(t *testing.T) {
	delta := &deaf{}
	target, v := newTranscriptionWorker(t, delta, "talk.mp3")

	var opened, provided int
	target.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (transcriptionport.Transcriber, func() error, error) {
		opened++
		return delta, func() error { provided++; return delta.Close() }, nil
	}

	if err := target.transcribeOne(t.Context(), v, "talk.mp3", true); err != nil {
		t.Fatalf("the recording was not transcribed: %v", err)
	}
	if opened != 1 {
		t.Errorf("what was missing was fetched %d times", opened)
	}
	if provided != 1 {
		t.Error("what was opened was not given back")
	}
	if delta.times() != 1 {
		t.Errorf("the recording was handed over %d times", delta.times())
	}
}

// Taking up a transcript proofreading uses the known hash from the index and
// avoids reading the source file from disk.
func TestTakingUpTranscriptProofreadingDoesNotReadTheSourceFileWhenHashIsKnown(t *testing.T) {
	delta := &deaf{}
	target, v := newTranscriptionWorker(t, delta, "talks/one.mp3")
	hash := text.Fingerprint([]byte("not a recording: talks/one.mp3"))

	store, err := target.with.Derived.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	raw := transcript.Marshal([]domain.TranscriptCue{
		{Text: "first thing", From: 0, To: 1000},
	})
	if err := store.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash), raw); err != nil {
		t.Fatal(err)
	}

	corr := &corrector{says: correctionsMap(fixes(0, "FIRST THING"))}
	target.with.Proofreading = run.ProofreadingConfig{
		IsNamed: true, IsAutomatic: true,
		By: func() (recognitionport.Proofreader, error) { return corr, nil },
	}
	target.with.Readers = brokenReaders{}

	recognized := heard{path: "talks/one.mp3", hash: hash}
	target.TakeUp(t.Context(), recognized, v)
	target.Wait()

	if len(corr.froms) != 1 {
		t.Errorf("expected proofreader to be asked, got %v", corr.froms)
	}
}

func TestAMissingRecordingIsNotReportedAsFailedAndIsPruned(t *testing.T) {
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta)
	recognized := unheard{recordings: []string{"missing.mp3"}}

	listening.round(t.Context(), recognized, v)

	if delta.times() != 0 {
		t.Errorf("expected transcriber not to be opened, got %d", delta.times())
	}
	for _, task := range listening.with.Tasks.List() {
		if task.Error != "" {
			t.Errorf("expected no failed task for missing file, got: %+v", task)
		}
	}
}

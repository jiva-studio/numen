package transcription

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// newDrop is one vault holding one recording, with the run that
// writes its words down, the cut that follows it, and the drop that takes both
// away.
func newDrop(t *testing.T, words ...string) (Transcribe, Drop, domain.Vault, *store, *shelf, *voice, string) {
	t.Helper()
	raw := newRecording(words)
	shelved := newLibrary()
	shelved.hold(recordingPath, domain.KindRecording, raw, 1)
	index := newStore()
	kept := newShelf()
	model := &voice{words: words}

	cutting := source.NewExtract(vaults{first.ID: shelved}, index, index)
	cutting.Derived = kept
	listen := NewTranscribe(vaults{first.ID: shelved}, index, shelves{kept}, model)
	listen.Batch = 1
	listen.Cut = func(ctx context.Context, v domain.Vault, path string) error {
		_, err := cutting.ExtractOne(ctx, v, path)
		return err
	}
	drop := Drop{
		Readers: vaults{first.ID: shelved},
		Sources: index,
		Queries: index,
		Derived: shelves{kept},
	}
	return listen, drop, first, index, kept, model, text.Fingerprint(raw)
}

// cutFrom is the text of the chunks one source was cut into.
func cutFrom(index *store, v domain.Vault, path string) []string {
	var out []string
	for _, c := range index.getOrderedChunks() {
		if c.vault == v.ID && c.path == path {
			out = append(out, c.text)
		}
	}
	return out
}

func TestDroppingATranscriptLeavesTheRecordingAsItWas(t *testing.T) {
	listen, drop, v, index, kept, _, hash := newDrop(t, "first thing", "second thing")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	// A person put the words right, so the file beside the artifact stands too.
	right := transcript.Marshal([]domain.TranscriptCue{{Text: "first thing said", From: 0, To: 800}})
	if err := kept.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash), right); err != nil {
		t.Fatal(err)
	}
	if len(cutFrom(index, v, recordingPath)) == 0 {
		t.Fatal("the recording was not cut from what was heard in it")
	}

	res, err := drop.Execute(t.Context(), v, recordingPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsNone || res.IsBusy {
		t.Errorf("a transcript that was there came back as %+v", res)
	}
	if left := kept.artifactNames(); len(left) != 0 {
		t.Errorf("the store still holds %v", left)
	}
	if chunks := cutFrom(index, v, recordingPath); len(chunks) != 0 {
		t.Errorf("the index still holds %d chunks of the words", len(chunks))
	}

	src := index.sources[v.ID][recordingPath]
	if src.Producer != "" || src.Hash != "" || src.Recipe != "" {
		t.Errorf("the source still stands on a recognition: %+v", src)
	}
	if src.Fingerprint.Path != recordingPath || src.Fingerprint.Kind != domain.KindRecording {
		t.Errorf("the recording is no longer a source of the vault: %+v", src.Fingerprint)
	}
}

func TestARecordingIsHeardAgainAfterItsTranscriptIsDropped(t *testing.T) {
	// The two runs share no word, so a chunk says which of them cut it.
	listen, drop, v, index, kept, model, hash := newDrop(t, "udyana", "vrksa", "bija")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if _, err := drop.Execute(t.Context(), v, recordingPath, nil); err != nil {
		t.Fatal(err)
	}

	model.words = []string{"aqua", "terra", "ventus"}
	res, err := listen.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 0 {
		t.Errorf("the run took up %d ms a run before it had written down", res.Resumed)
	}

	raw, err := kept.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash))
	if err != nil {
		t.Fatalf("the recording was not written down again: %v", err)
	}
	if said := getWords(t, raw); !slices.Equal(said, model.words) {
		t.Errorf("the artifact says %q and the recording says %q", said, model.words)
	}
	if index.sources[v.ID][recordingPath].Producer != domain.ASR {
		t.Error("the source does not stand on the transcript the second run wrote")
	}
	chunks := cutFrom(index, v, recordingPath)
	if len(chunks) == 0 {
		t.Fatal("the recording was not cut from what the second run heard")
	}
	for _, chunk := range chunks {
		for _, word := range []string{"udyana", "vrksa", "bija"} {
			if strings.Contains(chunk, word) {
				t.Errorf("a chunk still says %q, which is what the first run heard", word)
			}
		}
	}
}

func TestTheAnswerOfARecordingWithNoSpeechIsDropped(t *testing.T) {
	// A recording that gave no words stands on nothing, so what the run wrote
	// is found by the fingerprint of the bytes.
	listen, drop, v, index, kept, _, hash := newDrop(t, "", "")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if _, err := kept.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactAnswer, domain.ASR, hash)); err != nil {
		t.Fatalf("nothing says what the recording answered: %v", err)
	}

	res, err := drop.Execute(t.Context(), v, recordingPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsNone {
		t.Error("an answer that was there came back as nothing to drop")
	}
	if left := kept.artifactNames(); len(left) != 0 {
		t.Errorf("the store still holds %v", left)
	}
	if _, held := index.sources[v.ID][recordingPath]; !held {
		t.Error("the recording is no longer a source of the vault")
	}
}

func TestARecordingNobodyHasListenedToHasNoTranscriptToDrop(t *testing.T) {
	_, drop, v, index, kept, _, _ := newDrop(t, "one")

	res, err := drop.Execute(t.Context(), v, recordingPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNone {
		t.Errorf("a recording nothing has listened to came back as %+v", res)
	}
	if _, held := index.sources[v.ID][recordingPath]; held {
		t.Error("dropping nothing recorded a source")
	}
	if left := kept.artifactNames(); len(left) != 0 {
		t.Errorf("the store holds %v", left)
	}
}

// watching is the index, with a look at the store taken as the source is
// written.
type watching struct {
	*store
	saw func()
}

func (w watching) SaveExtraction(ctx context.Context, vaultID domain.VaultID, e domain.SourceChunks) error {
	w.saw()
	return w.store.SaveExtraction(ctx, vaultID, e)
}

// The recording is held for as long as the drop takes, so nothing listens to it
// and writes words the drop is about to say it has none of.
func TestTheRecordingIsHeldUntilTheIndexIsWritten(t *testing.T) {
	listen, drop, v, index, kept, _, hash := newDrop(t, "one", "two")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	free := false
	partial := domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, hash)
	drop.Sources = watching{store: index, saw: func() { free = !kept.claims(partial) }}

	if _, err := drop.Execute(t.Context(), v, recordingPath, nil); err != nil {
		t.Fatal(err)
	}
	if free {
		t.Error("the recording was free to be listened to before the index was written")
	}
	if kept.claims(partial) {
		t.Error("the drop left the recording held")
	}
}

// The store is a folder on the person's disk and they may empty it. What the
// index says about words nothing holds is the drop's to take away.
func TestARecordingWhoseStoreWasEmptiedIsDroppedFromTheIndex(t *testing.T) {
	listen, drop, v, index, kept, _, hash := newDrop(t, "one", "two")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	for _, name := range domain.GetArtifactNames(domain.ASR, hash) {
		if err := kept.RemoveArtifact(t.Context(), name); err != nil {
			t.Fatal(err)
		}
	}

	res, err := drop.Execute(t.Context(), v, recordingPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsNone {
		t.Error("a recording the index stands on came back as nothing to drop")
	}
	if src := index.sources[v.ID][recordingPath]; src.Producer != "" || src.Hash != "" {
		t.Errorf("the source still stands on a recognition: %+v", src)
	}
	if chunks := cutFrom(index, v, recordingPath); len(chunks) != 0 {
		t.Errorf("the index still holds %d chunks of words nothing holds", len(chunks))
	}
}

// A recording that gave no words is out of the queue's reach for the life of a
// run, and dropping its answer puts it back.
func TestDroppingAnAnswerPutsTheRecordingBackInReach(t *testing.T) {
	listen, drop, v, _, _, _, _ := newDrop(t, "", "")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	var forgotten []string
	forget := func(_ domain.Vault, path string) { forgotten = append(forgotten, path) }
	if _, err := drop.Execute(t.Context(), v, recordingPath, forget); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(forgotten, []string{recordingPath}) {
		t.Errorf("the queue was told about %v", forgotten)
	}
}

func TestATranscriptBeingWrittenIsNotDropped(t *testing.T) {
	listen, drop, v, index, kept, _, hash := newDrop(t, "one", "two")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	kept.hold(domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, hash))

	res, err := drop.Execute(t.Context(), v, recordingPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsBusy {
		t.Errorf("a recording a run holds came back as %+v", res)
	}
	if _, err := kept.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash)); err != nil {
		t.Errorf("the transcript went out from under the run: %v", err)
	}
	if index.sources[v.ID][recordingPath].Producer != domain.ASR {
		t.Error("the source was taken off its transcript")
	}
}

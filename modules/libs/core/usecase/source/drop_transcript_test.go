package source

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// dropping is one vault holding one recording, with the run that writes its
// words down, the cut that follows it, and the drop that takes both away.
func dropping(t *testing.T, words ...string) (Transcribe, DropTranscript, domain.Vault, *store, *shelf, *voice, string) {
	t.Helper()
	raw := recorded(words)
	shelved := newLibrary()
	shelved.hold(recordingPath, domain.KindRecording, raw, 1)
	index := newStore()
	kept := newShelf()
	model := &voice{words: words}

	cutting := Extract{Readers: vaults{string(first.ID): shelved}, Sources: index, Owing: index, Derived: kept}
	listen := Transcribe{
		Readers: vaults{string(first.ID): shelved},
		Sources: index,
		Derived: kept,
		By:      model,
		Batch:   1,
		Cut: func(ctx context.Context, v domain.Vault, path string) error {
			_, err := cutting.One(ctx, v, path)
			return err
		},
	}
	drop := DropTranscript{
		Readers: vaults{string(first.ID): shelved},
		Sources: index,
		Owing:   index,
		Derived: kept,
	}
	return listen, drop, first, index, kept, model, text.Fingerprint(raw)
}

// cutFrom is the text of the chunks one source was cut into.
func cutFrom(index *store, v domain.Vault, path string) []string {
	var out []string
	for _, c := range index.ordered() {
		if c.vault == string(v.ID) && c.path == path {
			out = append(out, c.text)
		}
	}
	return out
}

func TestDroppingATranscriptLeavesTheRecordingAsItWas(t *testing.T) {
	listen, drop, v, index, kept, _, hash := dropping(t, "first thing", "second thing")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	// A person put the words right, so the file beside the artifact stands too.
	right := transcript.Marshal([]transcript.Cue{{Text: "first thing said", From: 0, To: 800}})
	if err := kept.Write(t.Context(), text.Corrected(text.ASR, hash), right); err != nil {
		t.Fatal(err)
	}
	if len(cutFrom(index, v, recordingPath)) == 0 {
		t.Fatal("the recording was not cut from what was heard in it")
	}

	res, err := drop.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.None || res.Busy {
		t.Errorf("a transcript that was there came back as %+v", res)
	}
	if left := kept.names(); len(left) != 0 {
		t.Errorf("the store still holds %v", left)
	}
	if chunks := cutFrom(index, v, recordingPath); len(chunks) != 0 {
		t.Errorf("the index still holds %d chunks of the words", len(chunks))
	}

	src := index.sources[string(v.ID)][recordingPath]
	if src.TextFrom != "" || src.Hash != "" || src.Recipe != "" {
		t.Errorf("the source still stands on a reading: %+v", src)
	}
	if src.Ref.Path != recordingPath || src.Ref.Kind != domain.KindRecording {
		t.Errorf("the recording is no longer a source of the vault: %+v", src.Ref)
	}
}

func TestARecordingIsHeardAgainAfterItsTranscriptIsDropped(t *testing.T) {
	// The two runs share no word, so a chunk says which of them cut it.
	listen, drop, v, index, kept, model, hash := dropping(t, "udyana", "vrksa", "bija")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if _, err := drop.Execute(t.Context(), v, recordingPath); err != nil {
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

	raw, err := kept.Read(t.Context(), text.Artifact(text.ASR, hash))
	if err != nil {
		t.Fatalf("the recording was not written down again: %v", err)
	}
	if said := spoken(t, raw); !slices.Equal(said, model.words) {
		t.Errorf("the artifact says %q and the recording says %q", said, model.words)
	}
	if index.sources[string(v.ID)][recordingPath].TextFrom != text.ASR {
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
	listen, drop, v, index, kept, _, hash := dropping(t, "", "")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if _, err := kept.Read(t.Context(), text.Answer(text.ASR, hash)); err != nil {
		t.Fatalf("nothing says what the recording answered: %v", err)
	}

	res, err := drop.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.None {
		t.Error("an answer that was there came back as nothing to drop")
	}
	if left := kept.names(); len(left) != 0 {
		t.Errorf("the store still holds %v", left)
	}
	if _, held := index.sources[string(v.ID)][recordingPath]; !held {
		t.Error("the recording is no longer a source of the vault")
	}
}

func TestARecordingNobodyHasListenedToHasNoTranscriptToDrop(t *testing.T) {
	_, drop, v, index, kept, _, _ := dropping(t, "one")

	res, err := drop.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.None {
		t.Errorf("a recording nothing has listened to came back as %+v", res)
	}
	if _, held := index.sources[string(v.ID)][recordingPath]; held {
		t.Error("dropping nothing recorded a source")
	}
	if left := kept.names(); len(left) != 0 {
		t.Errorf("the store holds %v", left)
	}
}

// watching is the index, with a look at the store taken as the source is
// written.
type watching struct {
	*store
	saw func()
}

func (w watching) SaveExtraction(ctx context.Context, vaultID string, e port.SourceChunks) error {
	w.saw()
	return w.store.SaveExtraction(ctx, vaultID, e)
}

// The recording is held for as long as the drop takes, so nothing listens to it
// and writes words the drop is about to say it has none of.
func TestTheRecordingIsHeldUntilTheIndexIsWritten(t *testing.T) {
	listen, drop, v, index, kept, _, hash := dropping(t, "one", "two")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	free := false
	partial := text.Partial(text.ASR, hash)
	drop.Sources = watching{store: index, saw: func() { free = !kept.claims(partial) }}

	if _, err := drop.Execute(t.Context(), v, recordingPath); err != nil {
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
	listen, drop, v, index, kept, _, hash := dropping(t, "one", "two")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	for _, name := range text.Names(text.ASR, hash) {
		if err := kept.Remove(t.Context(), name); err != nil {
			t.Fatal(err)
		}
	}

	res, err := drop.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.None {
		t.Error("a recording the index stands on came back as nothing to drop")
	}
	if src := index.sources[string(v.ID)][recordingPath]; src.TextFrom != "" || src.Hash != "" {
		t.Errorf("the source still stands on a reading: %+v", src)
	}
	if chunks := cutFrom(index, v, recordingPath); len(chunks) != 0 {
		t.Errorf("the index still holds %d chunks of words nothing holds", len(chunks))
	}
}

// A recording that gave no words is out of the queue's reach for the life of a
// run, and dropping its answer puts it back.
func TestDroppingAnAnswerPutsTheRecordingBackInReach(t *testing.T) {
	listen, drop, v, _, _, _, _ := dropping(t, "", "")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	var forgotten []string
	drop.Forgets = func(_ domain.Vault, path string) { forgotten = append(forgotten, path) }

	if _, err := drop.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(forgotten, []string{recordingPath}) {
		t.Errorf("the queue was told about %v", forgotten)
	}
}

func TestATranscriptBeingWrittenIsNotDropped(t *testing.T) {
	listen, drop, v, index, kept, _, hash := dropping(t, "one", "two")

	if _, err := listen.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	kept.hold(text.Partial(text.ASR, hash))

	res, err := drop.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Busy {
		t.Errorf("a recording a run holds came back as %+v", res)
	}
	if _, err := kept.Read(t.Context(), text.Artifact(text.ASR, hash)); err != nil {
		t.Errorf("the transcript went out from under the run: %v", err)
	}
	if index.sources[string(v.ID)][recordingPath].TextFrom != text.ASR {
		t.Error("the source was taken off its transcript")
	}
}

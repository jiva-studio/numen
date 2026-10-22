package source

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const artifactHash = "abc123"

// newArtifactStore is one vault whose recording is indexed under a hash, with
// the transcript a model heard kept in the store.
func newArtifactStore(t *testing.T, cues ...domain.TranscriptCue) (*store, *fakeStorage) {
	t.Helper()
	index := newStore()
	index.sources[first.ID] = map[string]domain.Source{
		recordingPath: {
			Fingerprint: domain.Fingerprint{Path: recordingPath, Kind: domain.KindRecording},
			Producer:    domain.ASR,
			Hash:        artifactHash,
		},
	}
	shelf := newShelf()
	if err := shelf.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, artifactHash), transcript.Marshal(cues)); err != nil {
		t.Fatal(err)
	}
	return index, shelf
}

func getCueTexts(cues []domain.TranscriptCue) []string {
	out := make([]string, 0, len(cues))
	for _, one := range cues {
		out = append(out, one.Text)
	}
	return out
}

func TestListingArtifactsNamesWhatWasMadeFromAFile(t *testing.T) {
	index, shelf := newArtifactStore(t, domain.TranscriptCue{Text: "one", From: 0, To: 1000})

	target, err := NewGetArtifactList(index, fakeStores{shelf}).Execute(t.Context(), first, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 1 {
		t.Fatalf("%d artifacts listed, want the one transcript", len(target))
	}
	got := target[0]
	if got.Kind != KindTranscript || got.Producer != domain.ASR || got.Format != "text/vtt" || !got.IsReadable() {
		t.Errorf("the transcript was listed as %+v", got)
	}
}

func TestListingArtifactsOfAFileTheIndexDoesNotHoldIsEmpty(t *testing.T) {
	index, shelf := newArtifactStore(t, domain.TranscriptCue{Text: "one", From: 0, To: 1000})

	target, err := NewGetArtifactList(index, fakeStores{shelf}).Execute(t.Context(), first, "library/other.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 0 {
		t.Errorf("%d artifacts listed for a file nothing was made from", len(target))
	}
}

func TestACopyIsListedAndIsNotReadable(t *testing.T) {
	index, shelf := newArtifactStore(t, domain.TranscriptCue{Text: "one", From: 0, To: 1000})
	if err := shelf.WriteArtifact(t.Context(), domain.NewCopyName(artifactHash), []byte("video")); err != nil {
		t.Fatal(err)
	}

	target, err := NewGetArtifactList(index, fakeStores{shelf}).Execute(t.Context(), first, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 2 || target[1].Kind != KindCopy || target[1].IsReadable() || target[1].Bytes != 5 {
		t.Fatalf("the copy was listed as %+v", target)
	}
	if _, err := NewGetArtifactText(index, fakeStores{shelf}).Execute(t.Context(), first, recordingPath, KindCopy, 0, 0); err == nil {
		t.Error("a copy was read as words")
	}
}

func TestReadingAnArtifactProducesTheRangeAsItExistsOnDisk(t *testing.T) {
	index, shelf := newArtifactStore(t, domain.TranscriptCue{Text: "one", From: 0, To: 1000})
	read := NewGetArtifactText(index, fakeStores{shelf})

	whole, err := read.Execute(t.Context(), first, recordingPath, KindTranscript, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(whole.Words, "one") || whole.Start != 0 || whole.Length != len(whole.Words) {
		t.Errorf("the transcript read as %+v", whole)
	}

	part, err := read.Execute(t.Context(), first, recordingPath, KindTranscript, 5, 4)
	if err != nil {
		t.Fatal(err)
	}
	if part.Words != whole.Words[5:9] || part.Start != 5 || part.Length != 4 {
		t.Errorf("a range of four from five read as %+v", part)
	}
}

func TestReadingAKindNothingWasMadeAsIsRefused(t *testing.T) {
	index, shelf := newArtifactStore(t, domain.TranscriptCue{Text: "one", From: 0, To: 1000})

	_, err := NewGetArtifactText(index, fakeStores{shelf}).Execute(t.Context(), first, recordingPath, KindArticle, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "nothing of the kind") {
		t.Errorf("an article that was never made read as %v", err)
	}
}

func TestACorrectionReplacesTheCuesBeginningInsideItsSpan(t *testing.T) {
	index, shelf := newArtifactStore(t,
		domain.TranscriptCue{Text: "a", From: 0, To: 1000},
		domain.TranscriptCue{Text: "b", From: 1000, To: 2000},
		domain.TranscriptCue{Text: "c", From: 2000, To: 3000},
	)
	var changed []string
	write := NewWriteCorrections(index, fakeStores{shelf}, func(path string) { changed = append(changed, path) })

	words := string(transcript.Marshal([]domain.TranscriptCue{{Text: "B", From: 1000, To: 2000}}))
	n, err := write.Execute(t.Context(), first, Correction{Path: recordingPath, Kind: KindTranscript, Words: words})
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("the transcript holds %d cues, want 3", n)
	}
	raw, err := shelf.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, artifactHash))
	if err != nil {
		t.Fatal(err)
	}
	_, got := transcript.Parse(raw)
	if texts := getCueTexts(got); strings.Join(texts, "") != "aBc" {
		t.Errorf("the corrections read %v", texts)
	}
	if !transcript.IsWrittenByHand(raw) {
		t.Error("the corrections are not marked as written by hand")
	}
	if len(changed) != 1 || changed[0] != recordingPath {
		t.Errorf("the change was told as %v", changed)
	}
}

func TestACorrectionNamingASpanReplacesEveryCueInIt(t *testing.T) {
	index, shelf := newArtifactStore(t,
		domain.TranscriptCue{Text: "a", From: 0, To: 1000},
		domain.TranscriptCue{Text: "b", From: 1000, To: 2000},
		domain.TranscriptCue{Text: "c", From: 2000, To: 3000},
	)
	words := string(transcript.Marshal([]domain.TranscriptCue{{Text: "X", From: 0, To: 1000}}))

	n, err := NewWriteCorrections(index, fakeStores{shelf}, nil).Execute(t.Context(), first, Correction{
		Path: recordingPath, Kind: KindTranscript, Words: words, Start: 0, Length: 2000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("the transcript holds %d cues, want the one sent and the one after the span", n)
	}
}

func TestACorrectionIsRefusedWhereItCannotBeSaved(t *testing.T) {
	index, shelf := newArtifactStore(t, domain.TranscriptCue{Text: "a", From: 0, To: 1000})
	write := NewWriteCorrections(index, fakeStores{shelf}, nil)
	words := string(transcript.Marshal([]domain.TranscriptCue{{Text: "A", From: 0, To: 1000}}))

	if _, err := write.Execute(t.Context(), first, Correction{
		Path: recordingPath, Kind: KindRecognition, Words: words,
	}); !errors.Is(err, ErrNotCorrectable) {
		t.Errorf("a recognition was written: %v", err)
	}
	if _, err := write.Execute(t.Context(), first, Correction{
		Path: recordingPath, Kind: KindTranscript, Words: "nothing timed",
	}); !errors.Is(err, ErrNoCue) {
		t.Errorf("words with no cue were written: %v", err)
	}
	if _, err := write.Execute(t.Context(), first, Correction{
		Path: "library/other.mp3", Kind: KindTranscript, Words: words,
	}); err == nil {
		t.Error("a file nothing was made from was written")
	}

	shelf.hold(domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, artifactHash))
	if _, err := write.Execute(t.Context(), first, Correction{
		Path: recordingPath, Kind: KindTranscript, Words: words,
	}); !errors.Is(err, ErrTranscriptBusy) {
		t.Errorf("a transcript a run is writing was corrected: %v", err)
	}
}

package source

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/notes/domain"
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
	if err := shelf.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, artifactHash), text.MarshalWebVTT(cues)); err != nil {
		t.Fatal(err)
	}
	return index, shelf
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

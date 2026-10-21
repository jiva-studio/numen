package editor

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// noting is an index that holds on to what a drop wrote to it.
type noting struct {
	unrecorded
	written []domain.SourceChunks
}

func (n *noting) SaveExtraction(_ context.Context, _ domain.VaultID, e domain.SourceChunks) error {
	n.written = append(n.written, e)
	return nil
}

// dropper is a window holding one recording a model has listened to, with the
// use case that takes what it heard away. `known` is what the index says the
// recording relies on.
func dropper(t *testing.T, held port.DerivedStore, known indexed) (*Services, *noting, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{talk: sound})
	index := &noting{}
	hl := source.NewHighlight(filesystem.VaultReaders{}, known, storing{held})
	api := newBareServices()
	api.readFiles(filesystem.VaultReaders{})
	api.Ocr.highlight = &hl
	api.readTexts(source.NewTexts(filesystem.VaultReaders{}, nil, known, storing{held}))
	api.Artifact.drops = &transcription.Drop{
		Readers: filesystem.VaultReaders{},
		Sources: index,
		Queries: known,
		Derived: storing{held},
	}
	api.state.setVault(vault)
	return api, index, api.NewHandler(http.NotFoundHandler())
}

// newTranscriptIndex is what the index says a recording a model listened to relies on.
func newTranscriptIndex() indexed {
	return indexed{talk: {Fingerprint: domain.Fingerprint{Path: talk}, Producer: asr, Hash: hashed}}
}

// deleteTranscript asks for what a recording was heard as to be taken away.
func deleteTranscript(api *Services) error {
	_, err := api.Artifact.DeleteArtifact(context.Background(), connect.NewRequest(&v1.DeleteArtifactRequest{
		Path: talk, Kind: v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT,
	}))
	return err
}

// Everything one run of listening produced goes, and the recording is left
// owing its text.
func TestATranscriptDroppedTakesEverythingListeningProduced(t *testing.T) {
	target := whole(newCues())
	target[domain.NewArtifactName(domain.ArtifactCorrections, asr, hashed)] = target[domain.NewArtifactName(domain.ArtifactText, asr, hashed)]
	target[domain.NewArtifactName(domain.ArtifactProducer, asr, hashed)] = []byte(`{"model":"parakeet"}`)
	api, index, _ := dropper(t, target, newTranscriptIndex())

	if err := deleteTranscript(api); err != nil {
		t.Fatalf("dropped the transcript and failed: %v", err)
	}
	if len(target) != 0 {
		t.Errorf("the store still holds %v", target)
	}
	if len(index.written) != 1 {
		t.Fatalf("the index was written %d times", len(index.written))
	}
	wrote := index.written[0]
	if wrote.Source.Fingerprint.Path != talk {
		t.Errorf("the index was told about %q", wrote.Source.Fingerprint.Path)
	}
	if wrote.Source.Producer != "" || wrote.Source.Hash != "" || wrote.Source.Recipe != "" {
		t.Errorf("the source still stands on a reading: %+v", wrote.Source)
	}
	if len(wrote.Chunks) != 0 {
		t.Errorf("the source was left with %d chunks of the words", len(wrote.Chunks))
	}

	if notified := readTranscript(t, api); len(notified.GetCues()) != 0 {
		t.Errorf("the recording still says %+v", notified.GetCues())
	}
}

// A run appends to the transcript, and what is being appended to is not taken
// out from under it.
func TestATranscriptIsNotDroppedWhileTheRecordingIsBeingListenedTo(t *testing.T) {
	target := heldBy{stored: whole(newCues()), name: domain.NewArtifactName(domain.ArtifactPartial, asr, hashed)}
	api, index, _ := dropper(t, target, newTranscriptIndex())

	if err := deleteTranscript(api); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("dropped a transcript being written and failed %v", err)
	}
	if _, retained := target.stored[domain.NewArtifactName(domain.ArtifactText, asr, hashed)]; !retained {
		t.Error("the transcript went out from under the run")
	}
	if len(index.written) != 0 {
		t.Error("the index was written for a drop that did nothing")
	}
}

// A recording nobody has listened to has no transcript to drop.
func TestARecordingNobodyHasListenedToHasNoTranscriptToDrop(t *testing.T) {
	api, index, _ := dropper(t, stored{}, indexed{})

	if err := deleteTranscript(api); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("dropped the transcript of a recording nothing heard and failed %v", err)
	}
	if len(index.written) != 0 {
		t.Error("the index was written for a drop that did nothing")
	}
}

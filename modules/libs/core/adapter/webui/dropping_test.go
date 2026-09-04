package webui

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// noting is an index that holds on to what a drop wrote to it.
type noting struct {
	unrecorded
	written []port.SourceChunks
}

func (n *noting) SaveExtraction(_ context.Context, _ string, e port.SourceChunks) error {
	n.written = append(n.written, e)
	return nil
}

// dropper is a window holding one recording a model has listened to, with the
// use case that takes what it heard away. `known` is what the index says the
// recording stands on.
func dropper(t *testing.T, held port.DerivedStores, known indexed) (*API, *noting, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{talk: sound})
	index := &noting{}
	api := &API{
		Readers:   filesystem.VaultReaders{},
		Highlight: &source.Highlight{Sources: known, Derived: held},
		Drops: &source.DropTranscript{
			Readers: filesystem.VaultReaders{},
			Sources: index,
			Owing:   known,
			Derived: held,
		},
	}
	api.show(vault)
	return api, index, api.Serving(http.NotFoundHandler())
}

// heardBy is what the index says a recording a model listened to stands on.
func heardBy() indexed {
	return indexed{talk: {Path: talk, Producer: listener, Hash: hashed}}
}

// dropping asks for what a recording was heard as to be taken away.
func dropping(api *API) (*v1.Artifact, error) {
	out, err := api.DeleteArtifact(context.Background(), connect.NewRequest(&v1.DeleteArtifactRequest{
		Path: talk, ArtifactId: heardID,
	}))
	if err != nil {
		return nil, err
	}
	return out.Msg.GetArtifact(), nil
}

// Everything one run of listening produced goes, and the recording is left
// owing its text.
func TestATranscriptDroppedTakesEverythingListeningProduced(t *testing.T) {
	held := whole(spoke())
	held[derived.Corrected(listener, hashed)] = held[derived.Artifact(listener, hashed)]
	held[derived.Beside(listener, hashed)] = []byte(`{"model":"parakeet"}`)
	api, index, handler := dropper(t, held, heardBy())

	gone, err := dropping(api)
	if err != nil {
		t.Fatalf("dropped the transcript and was refused: %v", err)
	}
	if gone.GetState() != v1.State_STATE_NONE {
		t.Errorf("what was taken away is %s", gone.GetState())
	}
	if len(held) != 0 {
		t.Errorf("the store still holds %v", held)
	}
	if len(index.written) != 1 {
		t.Fatalf("the index was written %d times", len(index.written))
	}
	wrote := index.written[0]
	if wrote.Source.Fingerprint.Path != talk {
		t.Errorf("the index was told about %q", wrote.Source.Fingerprint.Path)
	}
	if wrote.Source.TextFrom != "" || wrote.Source.Hash != "" || wrote.Source.Recipe != "" {
		t.Errorf("the source still stands on a reading: %+v", wrote.Source)
	}
	if len(wrote.Chunks) != 0 {
		t.Errorf("the source was left with %d chunks of the words", len(wrote.Chunks))
	}

	if told := heard(t, handler); len(told.Cues) != 0 {
		t.Errorf("the recording still says %+v", told.Cues)
	}
}

// A run appends to the transcript, and what is being appended to is not taken
// out from under it.
func TestATranscriptIsNotDroppedWhileTheRecordingIsBeingListenedTo(t *testing.T) {
	held := heldBy{stored: whole(spoke()), name: derived.Partial(listener, hashed)}
	api, index, _ := dropper(t, held, heardBy())

	if _, err := dropping(api); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("dropped a transcript being written and was refused %v", err)
	}
	if _, kept := held.stored[derived.Artifact(listener, hashed)]; !kept {
		t.Error("the transcript went out from under the run")
	}
	if len(index.written) != 0 {
		t.Error("the index was written for a drop that did nothing")
	}
}

// A recording nobody has listened to has no transcript to drop.
func TestARecordingNobodyHasListenedToHasNoTranscriptToDrop(t *testing.T) {
	api, index, _ := dropper(t, stored{}, indexed{})

	if _, err := dropping(api); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("dropped the transcript of a recording nothing heard and was refused %v", err)
	}
	if len(index.written) != 0 {
		t.Error("the index was written for a drop that did nothing")
	}
}

// A build that cannot read a transcript cannot drop one either, and says so.
func TestABuildThatCannotHearDropsNoTranscript(t *testing.T) {
	api, _, _ := dropper(t, whole(spoke()), heardBy())
	api.Drops = nil

	if _, err := dropping(api); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("dropped a transcript in a build that cannot hear and was refused %v", err)
	}
}

// What a recording was heard as is the only artifact taken away here: the words
// a person put right go with it, and are not taken away on their own.
func TestOnlyWhatARecordingWasHeardAsIsTakenAway(t *testing.T) {
	api, index, _ := dropper(t, whole(spoke()), heardBy())

	_, err := api.DeleteArtifact(t.Context(), connect.NewRequest(&v1.DeleteArtifactRequest{
		Path: talk, ArtifactId: correctedID,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("took the corrections away on their own and was refused %v", err)
	}
	if len(index.written) != 0 {
		t.Error("the index was written for a drop that did nothing")
	}
}

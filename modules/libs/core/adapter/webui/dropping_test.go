package webui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// noting is an index that holds on to what a drop wrote to it.
type noting struct {
	unrecorded
	written []port.Extraction
}

func (n *noting) SaveExtraction(_ context.Context, _ string, e port.Extraction) error {
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
		Readers:   filesystem.Readers{},
		Highlight: &source.Highlight{Sources: known, Derived: held},
		Drops: &source.DropTranscript{
			Readers: filesystem.Readers{},
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
	return indexed{talk: {Path: talk, From: listener, Hash: hashed}}
}

// dropping asks the window's own facet for a transcript to take it away.
func dropping(handler http.Handler) *httptest.ResponseRecorder {
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, httptest.NewRequest(http.MethodDelete, cuesOf(talk), nil))
	return out
}

// Everything one run of listening produced goes, and the recording is left
// owing its text.
func TestATranscriptDroppedTakesEverythingListeningProduced(t *testing.T) {
	held := whole(spoke())
	held[derived.Corrected(listener, hashed)] = held[derived.Artifact(listener, hashed)]
	held[derived.Beside(listener, hashed)] = []byte(`{"model":"parakeet"}`)
	_, index, handler := dropper(t, held, heardBy())

	out := dropping(handler)
	if out.Code != http.StatusOK {
		t.Fatalf("dropped the transcript and got %d: %s", out.Code, out.Body)
	}
	if len(held) != 0 {
		t.Errorf("the store still holds %v", held)
	}
	if len(index.written) != 1 {
		t.Fatalf("the index was written %d times", len(index.written))
	}
	wrote := index.written[0]
	if wrote.Source.Ref.Path != talk {
		t.Errorf("the index was told about %q", wrote.Source.Ref.Path)
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
	_, index, handler := dropper(t, held, heardBy())

	out := dropping(handler)
	if out.Code != http.StatusConflict {
		t.Fatalf("dropped a transcript being written and got %d: %s", out.Code, out.Body)
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
	_, index, handler := dropper(t, stored{}, indexed{})

	out := dropping(handler)
	if out.Code != http.StatusNotFound {
		t.Fatalf("dropped the transcript of a recording nothing heard and got %d: %s", out.Code, out.Body)
	}
	if len(index.written) != 0 {
		t.Error("the index was written for a drop that did nothing")
	}
}

// A build that cannot read a transcript cannot drop one either, and says so.
func TestABuildThatCannotHearDropsNoTranscript(t *testing.T) {
	api, _, handler := dropper(t, whole(spoke()), heardBy())
	api.Drops = nil

	if out := dropping(handler); out.Code != http.StatusNotImplemented {
		t.Fatalf("dropped a transcript in a build that cannot hear and got %d: %s", out.Code, out.Body)
	}
}

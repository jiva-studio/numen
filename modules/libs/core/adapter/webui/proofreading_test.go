package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
)

// proofreads is the proofreading a test hands the window: whether this
// installation names anything to put a transcript right with, and what it was
// asked to put right.
type proofreads struct {
	ready bool
	vault string
	path  string
	times int
}

func (p *proofreads) ProofreaderReady() bool { return p.ready }

func (p *proofreads) Proofread(v domain.Vault, path string) {
	p.times++
	p.vault, p.path = v.ID, path
}

// proofreading is a window over the same vault the runs are asked for over,
// with a proofreading a test watches.
func proofreading(t *testing.T, held port.DerivedStores, read indexed) (*API, http.Handler, *proofreads) {
	t.Helper()
	api, handler := running(t, held, read, willRun(), willRun())
	by := &proofreads{ready: true}
	api.Proofreads = by
	return api, handler, by
}

// Where a proofreading is asked for.
func proofreadAt(path string) string { return assetOf(path) + "/" + proofreadFacet }

// onTheShelf is a store holding the transcript a model wrote of the recording.
func onTheShelf() stored {
	return stored{derived.Artifact(listener, hashed): []byte("what the model heard")}
}

// A transcript the window asks for is put right, and the run is told which file
// of which vault.
func TestATranscriptIsProofreadWhenTheWindowAsksForIt(t *testing.T) {
	api, handler, by := proofreading(t, onTheShelf(), heardBy())

	back := answered(t, post(handler, proofreadAt(talk)))
	if back.Answer != outcomeStarted {
		t.Fatalf("the transcript was answered %q: %s", back.Answer, back.Why)
	}
	if back.Path != talk {
		t.Errorf("the answer is about %q", back.Path)
	}
	if back.Why != "" {
		t.Errorf("a run begun was told %q, which the list of what is being done says", back.Why)
	}
	if by.times != 1 || by.path != talk || by.vault != api.Showing().ID {
		t.Errorf("the run was asked for %q of %q, %d times", by.path, by.vault, by.times)
	}
}

// One run to a recording. A person asking for a transcript another run holds is
// told that it is happening, and not that theirs did not start.
func TestATranscriptARunHoldsIsSaidToBeUnderWay(t *testing.T) {
	_, handler, by := proofreading(t, claimed{
		stored: onTheShelf(),
		name:   derived.Partial(listener, hashed),
	}, heardBy())

	back := answered(t, post(handler, proofreadAt(talk)))
	if back.Answer != outcomeRunning {
		t.Fatalf("a transcript a run holds was answered %q", back.Answer)
	}
	if back.Why != proofreadingNow {
		t.Errorf("it was told %q", back.Why)
	}
	if by.times != 0 {
		t.Error("a run was given a recording already being worked on")
	}
}

// A recording nothing has listened to holds no words to put right.
func TestARecordingNothingHasListenedToHasNothingToProofread(t *testing.T) {
	_, handler, by := proofreading(t, stored{}, nothingRead())

	back := answered(t, post(handler, proofreadAt(talk)))
	if back.Answer != outcomeUnheard {
		t.Fatalf("a recording nothing has heard was answered %q", back.Answer)
	}
	if back.Why != nothingHeard {
		t.Errorf("it was told %q", back.Why)
	}
	if by.times != 0 {
		t.Error("a run was given a recording holding no words")
	}
}

// A file that is not a recording carries no transcript, and nothing is given to
// the run.
func TestOnlyARecordingsTranscriptIsProofread(t *testing.T) {
	for _, path := range []string{book, idea} {
		t.Run(path, func(t *testing.T) {
			_, handler, by := proofreading(t, onTheShelf(), heardBy())

			back := answered(t, post(handler, proofreadAt(path)))
			if back.Answer != outcomeUnfit {
				t.Fatalf("a file of the wrong kind was answered %q", back.Answer)
			}
			if back.Why != notATranscript {
				t.Errorf("it was told %q", back.Why)
			}
			if by.times != 0 {
				t.Error("a run was given a file carrying no transcript")
			}
		})
	}
}

// An installation naming nothing to put a transcript right with says so, and
// the window offers the run nowhere after that.
func TestAnInstallationNamingNoProofreaderSaysSo(t *testing.T) {
	for _, one := range []struct {
		name  string
		holds func(*API)
	}{
		{"a build with no proofreading at all", func(a *API) { a.Proofreads = nil }},
		{"an installation naming no profile", func(a *API) { a.Proofreads = &proofreads{} }},
	} {
		t.Run(one.name, func(t *testing.T) {
			api, handler, _ := proofreading(t, onTheShelf(), heardBy())
			one.holds(api)

			out := post(handler, proofreadAt(talk))
			if out.Code != http.StatusNotImplemented {
				t.Fatalf("it answered %d: %s", out.Code, out.Body)
			}
		})
	}
}

// A run changes the vault, and is asked for with POST.
func TestAProofreadingIsAskedForWithPost(t *testing.T) {
	_, handler, by := proofreading(t, onTheShelf(), heardBy())

	out := httptest.NewRecorder()
	handler.ServeHTTP(out, httptest.NewRequest(http.MethodGet, proofreadAt(talk), nil))

	if out.Code != http.StatusMethodNotAllowed {
		t.Fatalf("a proofreading asked for with GET answered %d: %s", out.Code, out.Body)
	}
	if by.times != 0 {
		t.Error("a run was begun by a question that only asked")
	}
}

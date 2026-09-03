package webui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// proofreads is the proofreading a test hands the window: whether this
// installation names anything to put a transcript right with, what it was asked
// to put right, and what it says came of the ask.
type proofreads struct {
	ready bool
	came  source.PutRightResult
	why   error

	vault string
	path  string
	times int
}

func (p *proofreads) ProofreaderReady() bool { return p.ready }

func (p *proofreads) Proofread(
	_ context.Context,
	v domain.Vault,
	path string,
) (source.PutRightResult, error) {
	p.times++
	p.vault, p.path = string(v.ID), path
	return p.came, p.why
}

// proofreading is a window over the same vault the runs are asked for over,
// with a proofreading a test watches.
func proofreading(t *testing.T, held port.DerivedStores, read indexed) (*API, http.Handler, *proofreads) {
	t.Helper()
	api, handler := running(t, held, read, willRun(), willRun())
	by := &proofreads{ready: true}
	runningBehind(api, func(on *showing) { on.proofreads = by })
	return api, handler, by
}

// Where a proofreading is asked for.
func proofreadAt(path string) string { return assetOf(path) + "/" + proofreadFacet }

// onTheShelf is a store holding the transcript a model wrote of the recording.
func onTheShelf() stored {
	return stored{derived.Artifact(listener, hashed): []byte("what the model heard")}
}

// A transcript the window asks for is put right, the run is told which file of
// which vault, and the person is told that it began.
func TestATranscriptIsProofreadWhenTheWindowAsksForIt(t *testing.T) {
	api, handler, by := proofreading(t, onTheShelf(), heardBy())

	back := answered(t, post(handler, proofreadAt(talk)))
	if back.Answer != outcomeStarted {
		t.Fatalf("the transcript was answered %q: %s", back.Answer, back.Why)
	}
	if back.Path != talk {
		t.Errorf("the answer is about %q", back.Path)
	}
	if back.Why != proofreadingBegun {
		t.Errorf("a run begun was told %q", back.Why)
	}
	if by.times != 1 || by.path != talk || by.vault != string(api.Showing().ID) {
		t.Errorf("the run was asked for %q of %q, %d times", by.path, by.vault, by.times)
	}
}

// Every way a run can come to nothing is a sentence the person reads, so that
// pressing the item is never silence.
func TestWhatAProofreadingCameToIsSaid(t *testing.T) {
	for _, one := range []struct {
		name   string
		came   source.PutRightResult
		answer string
		why    string
	}{
		{"a recording another run holds", source.PutRightResult{Busy: true}, outcomeRunning, proofreadingNow},
		{"a transcript already put right", source.PutRightResult{Already: true}, outcomeDone, proofreadAlready},
		{"words a person wrote themselves", source.PutRightResult{Edited: true}, outcomeByHand, proofreadByHand},
		{"a transcript holding no words", source.PutRightResult{None: true}, outcomeUnheard, nothingHeard},
	} {
		t.Run(one.name, func(t *testing.T) {
			_, handler, by := proofreading(t, onTheShelf(), heardBy())
			by.came = one.came

			back := answered(t, post(handler, proofreadAt(talk)))
			if back.Answer != one.answer {
				t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
			}
			if back.Why == "" {
				t.Fatal("it was told nothing at all")
			}
			if back.Why != one.why {
				t.Errorf("it was told %q", back.Why)
			}
		})
	}
}

// A recording the index says nothing has listened to holds no words to put
// right, and the run is never asked for.
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
		{"a build with no proofreading at all", func(a *API) {
			runningBehind(a, func(on *showing) { on.proofreads = nil })
		}},
		{"an installation naming no profile", func(a *API) {
			runningBehind(a, func(on *showing) { on.proofreads = &proofreads{} })
		}},
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

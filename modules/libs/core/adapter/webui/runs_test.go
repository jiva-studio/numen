package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// What read the scan a test asks about, and the bytes it read.
const (
	reader  = "ocr"
	scanned = "7c3a91f4"
)

// A note stands in the vault beside the scan and the recording, as a file
// neither run is for.
const idea = "notes/idea.md"

// asking is a run a test hands the window: what it makes of a source it is
// given, and what it was given.
type asking struct {
	takes port.Taking
	vault string
	path  string
	times int
}

func (a *asking) Start(v domain.Vault, path string) port.Taking {
	a.times++
	a.vault, a.path = v.ID, path
	return a.takes
}

// claimed is a store a run holds one name in. A run holds the name it writes
// under for as long as it takes, and a claim on it is refused.
type claimed struct {
	stored
	name string
}

func (c claimed) Open(domain.Vault) (port.DerivedStore, error) { return c, nil }

func (c claimed) Claim(_ context.Context, name string) (func() error, error) {
	if name == c.name {
		return nil, port.ErrClaimed
	}
	return func() error { return nil }, nil
}

// running is a window over a vault holding a scan, a recording and a note, with
// the runs a test hands it and whatever a run before this one wrote down.
//
// read is what the index says each source's text came from; a source it does
// not name is one nothing has produced a text of.
func running(
	t *testing.T,
	held port.DerivedStores,
	read indexed,
	scans, hears Run,
) (*API, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{
		book: "the bytes of a scan",
		talk: sound,
		idea: "# an idea\n",
	})
	api := &API{
		Readers:   filesystem.VaultReaders{},
		Highlight: &source.Highlight{Sources: read, Derived: held},
	}
	api.show(vault)
	runningBehind(api, func(on *showing) { on.recognises, on.transcribes = scans, hears })
	return api, api.Serving(http.NotFoundHandler())
}

// willRun is a run that begins what it is given now, and willQueue one that
// puts it in line behind the work already going.
func willRun() *asking   { return &asking{takes: port.Began} }
func willQueue() *asking { return &asking{takes: port.Queued} }

// nothingRead is an index holding no produced text at all.
func nothingRead() indexed { return indexed{} }

// post asks for a run and answers with what came back.
func post(handler http.Handler, url string) *httptest.ResponseRecorder {
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, httptest.NewRequest(http.MethodPost, url, nil))
	return out
}

// answered is what the window was told, and it fails the test where the answer
// was not one.
func answered(t *testing.T, out *httptest.ResponseRecorder) began {
	t.Helper()
	if out.Code != http.StatusOK {
		t.Fatalf("asked for a run and got %d: %s", out.Code, out.Body)
	}
	var back began
	if err := json.NewDecoder(out.Body).Decode(&back); err != nil {
		t.Fatalf("the answer is not one the window reads: %v", err)
	}
	return back
}

// Where a run is asked for.
func recogniseAt(path string) string  { return assetOf(path) + "/" + recogniseFacet }
func transcribeAt(path string) string { return assetOf(path) + "/" + transcribeFacet }

// A scan the window asks for is read, and the run is told which file of which
// vault.
func TestAScanIsReadWhenTheWindowAsksForIt(t *testing.T) {
	scans := willRun()
	api, handler := running(t, stored{}, nothingRead(), scans, willRun())

	back := answered(t, post(handler, recogniseAt(book)))
	if back.Answer != outcomeStarted {
		t.Fatalf("the scan was answered %q: %s", back.Answer, back.Why)
	}
	if back.Path != book {
		t.Errorf("the answer is about %q", back.Path)
	}
	if back.Why != "" {
		t.Errorf("a run begun was told %q, which the list of what is being done says", back.Why)
	}
	if scans.times != 1 || scans.path != book || scans.vault != api.Showing().ID {
		t.Errorf("the run was asked for %q of %q, %d times", scans.path, scans.vault, scans.times)
	}
}

// A recording the window asks for is heard.
func TestARecordingIsHeardWhenTheWindowAsksForIt(t *testing.T) {
	hears := willRun()
	api, handler := running(t, stored{}, nothingRead(), willRun(), hears)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeStarted {
		t.Fatalf("the recording was answered %q: %s", back.Answer, back.Why)
	}
	if back.Path != talk {
		t.Errorf("the answer is about %q", back.Path)
	}
	if back.Why != "" {
		t.Errorf("a run begun was told %q, which the list of what is being done says", back.Why)
	}
	if hears.times != 1 || hears.path != talk || hears.vault != api.Showing().ID {
		t.Errorf("the run was asked for %q of %q, %d times", hears.path, hears.vault, hears.times)
	}
}

// A source named while a run of its kind is going waits its turn, and is told
// what it is waiting behind. Nothing is turned away.
func TestASourceNamedWhileARunIsGoingWaitsItsTurn(t *testing.T) {
	for _, one := range []struct {
		name string
		at   func(string) string
		path string
		why  string
	}{
		{"a scan", recogniseAt, book, readingQueued},
		{"a recording", transcribeAt, talk, hearingQueued},
	} {
		t.Run(one.name, func(t *testing.T) {
			scans, hears := willQueue(), willQueue()
			_, handler := running(t, stored{}, nothingRead(), scans, hears)

			back := answered(t, post(handler, one.at(one.path)))
			if back.Answer != outcomeQueued {
				t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
			}
			if back.Why != one.why {
				t.Errorf("it was told %q", back.Why)
			}
			if scans.times+hears.times != 1 {
				t.Error("the run was not given the source a person named")
			}
		})
	}
}

// A file of the wrong kind is answered with a sentence, and nothing is given to
// a run.
func TestAFileIsOnlyPutThroughTheRunItsKindIsFor(t *testing.T) {
	for _, one := range []struct {
		name string
		at   func(string) string
		path string
		why  string
	}{
		{"a scan asked to be heard", transcribeAt, book, notARecording},
		{"a recording asked to be read", recogniseAt, talk, notAScan},
		{"a note asked to be read", recogniseAt, idea, notAScan},
		{"a note asked to be heard", transcribeAt, idea, notARecording},
	} {
		t.Run(one.name, func(t *testing.T) {
			scans, hears := willRun(), willRun()
			_, handler := running(t, stored{}, nothingRead(), scans, hears)

			back := answered(t, post(handler, one.at(one.path)))
			if back.Answer != outcomeUnfit {
				t.Fatalf("a file of the wrong kind was answered %q", back.Answer)
			}
			if back.Why != one.why {
				t.Errorf("it was told %q", back.Why)
			}
			if scans.times+hears.times != 0 {
				t.Error("a run was given a file of the wrong kind")
			}
		})
	}
}

// A source already standing on the whole of the text a model produced is not
// put through the run again, and is told so plainly.
func TestASourceAlreadyDoneIsNotRunAgain(t *testing.T) {
	for _, one := range []struct {
		name string
		at   func(string) string
		path string
		from string
		hash string
		why  string
	}{
		{"a scan already read", recogniseAt, book, reader, scanned, readAlready},
		{"a recording already heard", transcribeAt, talk, listener, hashed, heardAlready},
	} {
		t.Run(one.name, func(t *testing.T) {
			scans, hears := willRun(), willRun()
			_, handler := running(t,
				stored{derived.Artifact(one.from, one.hash): []byte("what the model wrote")},
				indexed{one.path: {Path: one.path, Producer: one.from, Hash: one.hash}},
				scans, hears,
			)

			back := answered(t, post(handler, one.at(one.path)))
			if back.Answer != outcomeDone {
				t.Fatalf("a source already done was answered %q", back.Answer)
			}
			if back.Why != one.why {
				t.Errorf("it was told %q", back.Why)
			}
			if scans.times+hears.times != 0 {
				t.Error("a run was given a source already done")
			}
		})
	}
}

// A source a run holds is said to be under way. A recording is heard without
// anybody asking, so a person asking for the one being heard is told that it is
// happening, and not that theirs did not start.
func TestASourceARunHoldsIsSaidToBeUnderWay(t *testing.T) {
	for _, one := range []struct {
		name string
		at   func(string) string
		path string
		from string
		hash string
		why  string
	}{
		{"a scan being read", recogniseAt, book, reader, scanned, readingNow},
		{"a recording being heard", transcribeAt, talk, listener, hashed, hearingNow},
	} {
		t.Run(one.name, func(t *testing.T) {
			scans, hears := willRun(), willRun()
			_, handler := running(t,
				claimed{
					stored: stored{derived.Partial(one.from, one.hash): []byte("as far as it has got")},
					name:   derived.Partial(one.from, one.hash),
				},
				indexed{one.path: {Path: one.path, Producer: one.from, Hash: one.hash}},
				scans, hears,
			)

			back := answered(t, post(handler, one.at(one.path)))
			if back.Answer != outcomeRunning {
				t.Fatalf("a source a run holds was answered %q", back.Answer)
			}
			if back.Why != one.why {
				t.Errorf("it was told %q", back.Why)
			}
			if scans.times+hears.times != 0 {
				t.Error("a run was given a source already being worked on")
			}
		})
	}
}

// A source nothing holds is not under way, whatever a run wrote before it
// stopped. It is named afresh and waits its turn.
func TestASourceNothingHoldsIsNotUnderWay(t *testing.T) {
	for _, one := range []struct {
		name string
		at   func(string) string
		path string
		from string
		hash string
		why  string
	}{
		{"a scan", recogniseAt, book, reader, scanned, readingQueued},
		{"a recording", transcribeAt, talk, listener, hashed, hearingQueued},
	} {
		t.Run(one.name, func(t *testing.T) {
			_, handler := running(t,
				stored{derived.Partial(one.from, one.hash): []byte("as far as it got")},
				indexed{one.path: {Path: one.path, Producer: one.from, Hash: one.hash}},
				willQueue(), willQueue(),
			)

			back := answered(t, post(handler, one.at(one.path)))
			if back.Answer != outcomeQueued {
				t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
			}
			if back.Why != one.why {
				t.Errorf("it was told %q", back.Why)
			}
		})
	}
}

// A source a run holds and has finished is one already done. A finished run
// gives the name it was writing under back.
func TestASourceDoneIsDoneEvenWhereAPartialStands(t *testing.T) {
	scans := willRun()
	_, handler := running(t,
		stored{
			derived.Artifact(reader, scanned): []byte("what the model wrote"),
			derived.Partial(reader, scanned):  []byte("what it wrote on the way"),
		},
		indexed{book: {Path: book, Producer: reader, Hash: scanned}},
		scans, willRun(),
	)

	back := answered(t, post(handler, recogniseAt(book)))
	if back.Answer != outcomeDone {
		t.Errorf("it was answered %q: %s", back.Answer, back.Why)
	}
	if scans.times != 0 {
		t.Error("a run was given a scan already read")
	}
}

// A source whose own bytes are the text has had no run over it, and asking for
// one begins it.
func TestASourceCarryingItsOwnTextHasNotBeenRun(t *testing.T) {
	scans := willRun()
	_, handler := running(t, stored{},
		indexed{book: {Path: book, Hash: scanned}},
		scans, willRun(),
	)

	back := answered(t, post(handler, recogniseAt(book)))
	if back.Answer != outcomeStarted {
		t.Fatalf("the scan was answered %q: %s", back.Answer, back.Why)
	}
	if scans.times != 1 {
		t.Errorf("the run was asked for %d times", scans.times)
	}
}

// A path the vault does not hold is not something a person can act on, and is
// refused as an error.
func TestARunOverAPathTheVaultDoesNotHoldIsNotFound(t *testing.T) {
	_, handler := running(t, stored{}, nothingRead(), willRun(), willRun())

	for _, at := range []string{recogniseAt("library/nothing.pdf"), transcribeAt("talks/nothing.mp3")} {
		out := post(handler, at)
		if out.Code != http.StatusNotFound {
			t.Errorf("asked for a run over nothing and got %d: %s", out.Code, out.Body)
		}
	}
}

// A build with nothing to run with offers no facet, and the window then offers
// nothing.
func TestABuildWithNoRunsOffersNoFacet(t *testing.T) {
	_, handler := running(t, stored{}, nothingRead(), nil, nil)

	for _, at := range []string{recogniseAt(book), transcribeAt(talk)} {
		out := post(handler, at)
		if out.Code != http.StatusNotImplemented {
			t.Errorf("asked a build that cannot run and got %d: %s", out.Code, out.Body)
		}
	}
}

// A run is asked for with POST, and nothing is given to a run on any other
// method.
func TestARunIsAskedForWithPost(t *testing.T) {
	scans := willRun()
	_, handler := running(t, stored{}, nothingRead(), scans, willRun())

	out := ask(handler, recogniseAt(book))
	if out.Code != http.StatusMethodNotAllowed {
		t.Fatalf("asked for a run with GET and got %d: %s", out.Code, out.Body)
	}
	if scans.times != 0 {
		t.Error("a run was given a source without being asked for with POST")
	}
}

// Every outcome but the run begun carries a sentence, and no two outcomes say
// the same thing.
func TestEveryOutcomeSaysSomethingOfItsOwn(t *testing.T) {
	said := map[string]bool{}
	for _, why := range []string{
		notAScan, notARecording,
		readAlready, heardAlready,
		readingNow, hearingNow,
		readingQueued, hearingQueued,
		readingSilent, hearingSilent,
		readingUnopened, hearingUnopened,
	} {
		if why == "" {
			t.Fatal("an outcome carries no sentence")
		}
		if said[why] {
			t.Errorf("two outcomes say %q", why)
		}
		said[why] = true
	}
}

package webui

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// Where the recording a test asks about sits in its vault, and the bytes that
// stand in for the sound of it.
const (
	talk  = "talks/lecture.mp3"
	sound = "the bytes of a recording, long enough to ask for a piece of"
)

// stored is what a run of transcription left in the vault's store, by the name
// it left it under.
type stored map[string][]byte

func (s stored) Open(domain.Vault) (port.DerivedStore, error) { return s, nil }

func (s stored) Read(_ context.Context, name string) ([]byte, error) {
	raw, held := s[name]
	if !held {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (s stored) Write(context.Context, string, []byte) error  { return nil }
func (s stored) Append(context.Context, string, []byte) error { return nil }
func (s stored) Remove(context.Context, string) error         { return nil }

func (s stored) List(context.Context, string) ([]port.Stored, error) { return nil, nil }

func (s stored) Claim(context.Context, string) (func() error, error) {
	return func() error { return nil }, nil
}

// The model a test's words were heard by, and the bytes it heard them in.
const (
	listener = "parakeet-1"
	hashed   = "2fd4e1c6"
)

// listeningTo is a window holding one recording, with what a model wrote of it
// in the store. Nothing written down is a recording nobody has listened to.
func listeningTo(t *testing.T, held stored) (*API, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{talk: sound, book: "the bytes of a scan"})
	api := &API{
		Readers: filesystem.Readers{},
		Marking: &source.Marks{
			Sources: indexed{talk: {Path: talk, From: listener, Hash: hashed}},
			Derived: held,
		},
	}
	api.show(vault)
	return api, api.Serving(http.NotFoundHandler())
}

// whole is the store holding a finished transcript of the recording, and partly
// is one a run is still writing.
func whole(cues []transcript.Cue) stored {
	return stored{derived.Artifact(listener, hashed): transcript.Write(cues)}
}

func partly(cues []transcript.Cue, reached int) stored {
	return stored{
		derived.Partial(listener, hashed): append(transcript.Write(cues), transcript.Heard(reached)...),
	}
}

// spoke is what a test writes down of the recording.
func spoke() []transcript.Cue {
	return []transcript.Cue{
		{Text: "what was said", From: 1500, To: 4200},
		{Text: "what was said next", From: 4200, To: 9100},
	}
}

// A recording is played from its own bytes.
func TestARecordingIsPlayedFromItsOwnBytes(t *testing.T) {
	_, handler := listeningTo(t, nil)

	out := ask(handler, mediaOf(talk))
	if out.Code != http.StatusOK {
		t.Fatalf("asked for the recording and got %d: %s", out.Code, out.Body)
	}
	if got := out.Header().Get("Content-Type"); got != "audio/mpeg" {
		t.Errorf("the recording came back as %q", got)
	}
	if out.Body.String() != sound {
		t.Errorf("the recording came back as %q", out.Body)
	}
	if out.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("the recording is not answered a piece at a time")
	}
}

// A player seeking asks for the piece it lands in, and is answered with that
// piece and no more.
func TestAPlayerAsksForOnePieceOfARecording(t *testing.T) {
	_, handler := listeningTo(t, nil)

	out := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, mediaOf(talk), nil)
	r.Header.Set("Range", "bytes=4-12")
	handler.ServeHTTP(out, r)

	if out.Code != http.StatusPartialContent {
		t.Fatalf("asked for a piece and got %d: %s", out.Code, out.Body)
	}
	if got := out.Body.String(); got != sound[4:13] {
		t.Errorf("the piece came back as %q, want %q", got, sound[4:13])
	}
	if got := out.Header().Get("Content-Range"); !strings.HasPrefix(got, "bytes 4-12/") {
		t.Errorf("the piece is said to be %q", got)
	}
}

// The words come back against the milliseconds they were spoken in, so the
// window draws them along the recording.
func TestTheWordsHeardComeBackAgainstTheRecording(t *testing.T) {
	for _, one := range []struct {
		what string
		held stored
	}{
		{"a run that finished", whole(spoke())},
		{"a run still going", partly(spoke(), 9100)},
	} {
		t.Run(one.what, func(t *testing.T) {
			_, handler := listeningTo(t, one.held)

			out := ask(handler, cuesOf(talk))
			if out.Code != http.StatusOK {
				t.Fatalf("asked what was heard and got %d: %s", out.Code, out.Body)
			}
			var told spoken
			if err := json.NewDecoder(out.Body).Decode(&told); err != nil {
				t.Fatal(err)
			}
			if told.Path != talk || len(told.Cues) != 2 {
				t.Fatalf("what was heard came back as %+v", told)
			}
			if first := told.Cues[0]; first.Text != "what was said" || first.From != 1500 || first.To != 4200 {
				t.Errorf("the first thing said came back as %+v", first)
			}
		})
	}
}

// A recording nobody has listened to holds no words, and is played all the
// same.
func TestARecordingNobodyHasListenedToHoldsNoWords(t *testing.T) {
	_, handler := listeningTo(t, nil)

	out := ask(handler, cuesOf(talk))
	if out.Code != http.StatusOK {
		t.Fatalf("asked what was heard and got %d: %s", out.Code, out.Body)
	}
	var told spoken
	if err := json.NewDecoder(out.Body).Decode(&told); err != nil {
		t.Fatal(err)
	}
	if len(told.Cues) != 0 {
		t.Errorf("a recording nobody heard says %+v", told.Cues)
	}
	if played := ask(handler, mediaOf(talk)); played.Code != http.StatusOK {
		t.Errorf("the recording itself was answered %d", played.Code)
	}
}

// What a recording is: how far the words reach, and how much of it a run has
// written down.
func TestWhatARecordingIsIsHowLongItRuns(t *testing.T) {
	for _, one := range []struct {
		what   string
		held   stored
		length int
		heard  int
	}{
		{"nothing heard", nil, 0, 0},
		{"a run that finished", whole(spoke()), 9100, 0},
		{"a run still going", partly(spoke()[:1], 4200), 4200, 4200},
	} {
		t.Run(one.what, func(t *testing.T) {
			_, handler := listeningTo(t, one.held)

			out := ask(handler, assetOf(talk))
			if out.Code != http.StatusOK {
				t.Fatalf("asked what the recording is and got %d: %s", out.Code, out.Body)
			}
			var told listened
			if err := json.NewDecoder(out.Body).Decode(&told); err != nil {
				t.Fatal(err)
			}
			if told.Path != talk || told.Length != one.length || told.Heard != one.heard {
				t.Errorf("the recording came back as %+v, want %d long and %d heard",
					told, one.length, one.heard)
			}
		})
	}
}

// The facets of a recording are a recording's. A document is played from
// nowhere and says nothing.
func TestOnlyARecordingIsPlayedAndHeard(t *testing.T) {
	_, handler := listeningTo(t, whole(spoke()))

	for _, url := range []string{mediaOf(book), cuesOf(book), mediaOf("talks/none.mp3")} {
		if out := ask(handler, url); out.Code != http.StatusNotFound {
			t.Errorf("%s was answered %d", url, out.Code)
		}
	}
}

// A build with nothing to read a transcript with says so, and the recording is
// played all the same.
func TestABuildThatReadsNoTranscriptSaysSo(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{talk: sound})
	api := &API{Readers: filesystem.Readers{}}
	api.show(vault)
	handler := api.Serving(http.NotFoundHandler())

	if out := ask(handler, cuesOf(talk)); out.Code != http.StatusNotImplemented {
		t.Errorf("asked what was heard and got %d: %s", out.Code, out.Body)
	}
	if out := ask(handler, mediaOf(talk)); out.Code != http.StatusOK {
		t.Errorf("the recording itself was answered %d", out.Code)
	}
}

// A file of the vault is one part of its address, and what is asked of it is
// the next.
func TestARecordingsFacetsAreAddressed(t *testing.T) {
	for _, one := range []struct {
		url   string
		facet string
	}{
		{mediaOf(talk), mediaFacet},
		{cuesOf(talk), cuesFacet},
	} {
		got, ok := addressed(httptest.NewRequest(http.MethodGet, one.url, nil))
		if !ok {
			t.Fatalf("%s is not an address", one.url)
		}
		if got.path != talk || got.facet != one.facet {
			t.Errorf("%s reads as %q/%q", one.url, got.path, got.facet)
		}
	}
}

package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func (s stored) Write(_ context.Context, name string, content []byte) error {
	s[name] = content
	return nil
}

func (s stored) Append(context.Context, string, []byte) error { return nil }

func (s stored) Remove(_ context.Context, name string) error {
	delete(s, name)
	return nil
}

func (s stored) List(context.Context, string) ([]port.Stored, error) { return nil, nil }

func (s stored) Claim(context.Context, string) (func() error, error) {
	return func() error { return nil }, nil
}

// The model a test's words were heard by, and the bytes it heard them in.
const (
	listener = derived.ASR
	hashed   = "2fd4e1c6"
)

// listeningTo is a window holding one recording, with what a model wrote of it
// in the store. Nothing written down is a recording nobody has listened to.
func listeningTo(t *testing.T, held stored) (*API, http.Handler) {
	t.Helper()
	return windowOn(t, held)
}

// windowOn is the same window, with whatever store the test hands it.
func windowOn(t *testing.T, held port.DerivedStores) (*API, http.Handler) {
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
	return stored{derived.Artifact(listener, hashed): transcript.Marshal(cues)}
}

func partly(cues []transcript.Cue, reached int) stored {
	return stored{
		derived.Partial(listener, hashed): append(transcript.Marshal(cues), transcript.Heard(reached)...),
	}
}

// spoke is what a test writes down of the recording.
func spoke() []transcript.Cue {
	return []transcript.Cue{
		{Text: "what was said", From: 1500, To: 4200},
		{Text: "what was said next", From: 4200, To: 9100},
	}
}

// A recording is played from its own bytes, over the socket a player reaches.
func TestARecordingIsPlayedFromItsOwnBytes(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	out := ask(handler, back.Address(api.Showing(), talk))
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
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	out := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, back.Address(api.Showing(), talk), nil)
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
	api, handler := listeningTo(t, nil)

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
	back, playing := played(t, api)
	if out := ask(playing, back.Address(api.Showing(), talk)); out.Code != http.StatusOK {
		t.Errorf("the recording itself was answered %d", out.Code)
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

// What was heard is a recording's. A document holds no words and is asked for
// none.
func TestOnlyARecordingIsHeard(t *testing.T) {
	_, handler := listeningTo(t, whole(spoke()))

	for _, url := range []string{cuesOf(book), cuesOf("talks/none.mp3")} {
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
	back, playing := played(t, api)
	if out := ask(playing, back.Address(api.Showing(), talk)); out.Code != http.StatusOK {
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

// A question about a run of the words is answered with the speech those bytes
// were said in, so a search hit is played from where it was said.
func TestCuesNarrowToARunOfTheWords(t *testing.T) {
	said := []transcript.Cue{
		{Text: "first", From: 0, To: 1000},
		{Text: "second", From: 1000, To: 2000},
		{Text: "third", From: 2000, To: 3000},
	}
	_, cues := transcript.Parse(transcript.Marshal(said))

	// "second" begins after "first\n".
	got, err := narrowed(url.Values{"start": {"6"}, "length": {"6"}}, cues)
	if err != nil {
		t.Fatalf("a run of the words was refused: %v", err)
	}
	if len(got) != 1 || got[0].From != 1000 {
		t.Errorf("the run was placed at %v", got)
	}

	whole, err := narrowed(url.Values{}, cues)
	if err != nil || len(whole) != 3 {
		t.Errorf("a question naming no run answered with %d cues (%v)", len(whole), err)
	}

	if _, err := narrowed(url.Values{"start": {"-1"}, "length": {"6"}}, cues); err == nil {
		t.Errorf("a place before the words was taken")
	}
}

// heldBy is a store with a run holding one of its names, as a transcription
// holds the recording it is writing down.
type heldBy struct {
	stored
	name string
}

func (h heldBy) Open(domain.Vault) (port.DerivedStore, error) { return h, nil }

func (h heldBy) Claim(ctx context.Context, name string) (func() error, error) {
	if name == h.name {
		return nil, fmt.Errorf("%s: %w", name, port.ErrClaimed)
	}
	return h.stored.Claim(ctx, name)
}

// putting sends a transcript to the window's own facet for one.
func putting(handler http.Handler, body string) *httptest.ResponseRecorder {
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, httptest.NewRequest(http.MethodPut, cuesOf(talk), strings.NewReader(body)))
	return out
}

// edited is a transcript as the window sends one.
func edited(cues ...cue) string {
	body, err := json.Marshal(putRight{Path: talk, Cues: cues})
	if err != nil {
		panic(err)
	}
	return string(body)
}

// What a transcript was put right to is kept beside what was heard, and what
// was heard stays where it is.
func TestATranscriptPutRightIsKeptBesideWhatWasHeard(t *testing.T) {
	held := whole(spoke())
	_, handler := listeningTo(t, held)

	out := putting(handler, edited(
		cue{Text: "what was said", From: 1500, To: 4200},
		cue{Text: "what Rupa said next", From: 4200, To: 9100},
	))
	if out.Code != http.StatusOK {
		t.Fatalf("put the transcript right and got %d: %s", out.Code, out.Body)
	}

	put, kept := held[derived.Corrected(listener, hashed)]
	if !kept {
		t.Fatalf("nothing was kept beside the transcript: %v", held)
	}
	if _, cues := transcript.Parse(put); len(cues) != 2 || cues[1].Text != "what Rupa said next" {
		t.Errorf("the transcript was put right to %q", put)
	}
	if _, cues := transcript.Parse(held[derived.Artifact(listener, hashed)]); cues[1].Text != "what was said next" {
		t.Error("what was heard was written over")
	}
}

// What a person wrote says so, so a proofreader leaves it alone.
func TestATranscriptAPersonWroteSaysSo(t *testing.T) {
	held := whole(spoke())
	_, handler := listeningTo(t, held)

	out := putting(handler, edited(cue{Text: "what Rupa said", From: 1500, To: 4200}))
	if out.Code != http.StatusOK {
		t.Fatalf("put the transcript right and got %d: %s", out.Code, out.Body)
	}
	if !transcript.Written(held[derived.Corrected(listener, hashed)]) {
		t.Errorf("the transcript does not say a person wrote it:\n%s", held[derived.Corrected(listener, hashed)])
	}
}

// The chunks in the index hold the words as they were heard, so a transcript
// put right is a source asked for again.
func TestATranscriptPutRightIsCutAgain(t *testing.T) {
	api, handler := listeningTo(t, whole(spoke()))

	var asked []string
	runningBehind(api, func(on *showing) {
		on.cut = func(_ context.Context, _ domain.Vault, path string) error {
			asked = append(asked, path)
			return nil
		}
	})

	out := putting(handler, edited(cue{Text: "what was said", From: 1500, To: 4200}))
	if out.Code != http.StatusOK {
		t.Fatalf("put the transcript right and got %d: %s", out.Code, out.Body)
	}
	if len(asked) != 1 || asked[0] != talk {
		t.Errorf("the sources cut again are %v", asked)
	}
}

// A cut that could not be asked for is not a write that failed: the correction
// is on disk either way.
func TestATranscriptStandsWhenItCannotBeCutAgain(t *testing.T) {
	held := whole(spoke())
	api, handler := listeningTo(t, held)
	runningBehind(api, func(on *showing) {
		on.cut = func(context.Context, domain.Vault, string) error {
			return errors.New("nothing is cutting")
		}
	})

	out := putting(handler, edited(cue{Text: "what was said", From: 1500, To: 4200}))
	if out.Code != http.StatusOK {
		t.Fatalf("put the transcript right and got %d: %s", out.Code, out.Body)
	}
	if _, kept := held[derived.Corrected(listener, hashed)]; !kept {
		t.Error("the correction was not kept")
	}
}

// The window is told the transcript as it now stands, and taking away what it
// was put right to gives back what was heard.
func TestATranscriptPutRightIsWhatTheWindowIsToldNext(t *testing.T) {
	held := whole(spoke())
	_, handler := listeningTo(t, held)

	if out := putting(handler, edited(
		cue{Text: "what was said", From: 1500, To: 4200},
		cue{Text: "what Rupa said next", From: 4200, To: 9100},
	)); out.Code != http.StatusOK {
		t.Fatalf("put the transcript right and got %d: %s", out.Code, out.Body)
	}
	if told := heard(t, handler); told.Cues[1].Text != "what Rupa said next" {
		t.Errorf("the window is told %+v", told.Cues)
	}

	delete(held, derived.Corrected(listener, hashed))
	if told := heard(t, handler); told.Cues[1].Text != "what was said next" {
		t.Errorf("what was heard did not come back: %+v", told.Cues)
	}
}

// heard is the transcript the window is told about.
func heard(t *testing.T, handler http.Handler) spoken {
	t.Helper()
	out := ask(handler, cuesOf(talk))
	if out.Code != http.StatusOK {
		t.Fatalf("asked what was heard and got %d: %s", out.Code, out.Body)
	}
	var told spoken
	if err := json.NewDecoder(out.Body).Decode(&told); err != nil {
		t.Fatal(err)
	}
	return told
}

// Speech runs forward, and a transcript that says otherwise was not cut from a
// recording. Nothing of it is written.
func TestATranscriptThatRunsBackwardsIsRefused(t *testing.T) {
	for _, one := range []struct {
		what string
		body string
	}{
		{"a cue ending before it began", edited(cue{Text: "said", From: 4200, To: 1500})},
		{"a cue before the one above it", edited(
			cue{Text: "said", From: 4200, To: 9100},
			cue{Text: "said next", From: 1500, To: 2000},
		)},
		{"a cue overlapping the one above it", edited(
			cue{Text: "said", From: 1500, To: 4200},
			cue{Text: "said next", From: 3000, To: 9100},
		)},
		{"a cue beginning before the recording", edited(cue{Text: "said", From: -1, To: 4200})},
		{"bytes that are not a transcript", "{"},
		{"a transcript of another recording", `{"path":"talks/other.mp3","cues":[]}`},
		// One cue is one line, and the window edits it as one.
		{"a cue broken over two lines", edited(
			cue{Text: "said\nand said next", From: 1500, To: 4200},
		)},
		{"a cue carrying a carriage return", edited(
			cue{Text: "said\rand said next", From: 1500, To: 4200},
		)},
	} {
		t.Run(one.what, func(t *testing.T) {
			held := whole(spoke())
			_, handler := listeningTo(t, held)

			out := putting(handler, one.body)
			if out.Code != http.StatusBadRequest {
				t.Fatalf("the transcript was answered with %d: %s", out.Code, out.Body)
			}
			if strings.TrimSpace(out.Body.String()) == "" {
				t.Error("the transcript was refused without saying why")
			}
			if _, kept := held[derived.Corrected(listener, hashed)]; kept {
				t.Error("a transcript that was refused was written")
			}
		})
	}
}

// A cue whose words trim away is silence, and Marshal writes none.
func TestACueWithNoWordsIsDropped(t *testing.T) {
	held := whole(spoke())
	_, handler := listeningTo(t, held)

	if out := putting(handler, edited(
		cue{Text: "what was said", From: 1500, To: 4200},
		cue{Text: "   ", From: 4200, To: 6000},
		cue{Text: "what was said last", From: 6000, To: 9100},
	)); out.Code != http.StatusOK {
		t.Fatalf("put the transcript right and got %d: %s", out.Code, out.Body)
	}
	if _, cues := transcript.Parse(held[derived.Corrected(listener, hashed)]); len(cues) != 2 {
		t.Errorf("the transcript was written down as %v", cues)
	}
}

// A run appends to the transcript, and what is being appended to is not edited
// underneath.
func TestATranscriptIsNotEditedWhileTheRecordingIsBeingListenedTo(t *testing.T) {
	held := heldBy{stored: partly(spoke(), 9100), name: derived.Partial(listener, hashed)}
	_, handler := windowOn(t, held)

	out := putting(handler, edited(cue{Text: "what was said", From: 1500, To: 4200}))
	if out.Code != http.StatusConflict {
		t.Fatalf("edited a recording being listened to and got %d: %s", out.Code, out.Body)
	}
	if _, kept := held.stored[derived.Corrected(listener, hashed)]; kept {
		t.Error("the transcript was written while a run held the recording")
	}

	if told := heard(t, handler); told.Editable {
		t.Error("the window was told it may edit a transcript a run is writing")
	}
}

// A transcript nothing is writing may be put right, and the window is told so.
func TestTheWindowIsToldATranscriptMayBePutRight(t *testing.T) {
	_, handler := listeningTo(t, whole(spoke()))

	if told := heard(t, handler); !told.Editable {
		t.Error("the window was told it may not edit a transcript nothing is writing")
	}
}

// A recording nobody has listened to has no transcript to put right.
func TestARecordingNobodyHasListenedToHasNoTranscriptToPutRight(t *testing.T) {
	held := stored{}
	api, handler := listeningTo(t, held)
	api.Marking.Sources = indexed{}

	out := putting(handler, edited(cue{Text: "what was said", From: 1500, To: 4200}))
	if out.Code != http.StatusNotFound {
		t.Fatalf("put right a recording nothing heard and got %d: %s", out.Code, out.Body)
	}
	if len(held) != 0 {
		t.Errorf("a transcript was written for a recording nothing heard: %v", held)
	}
}

// A transcript of no words is not one a recording was put right to. Taking away
// the file beside the artifact is what gives back what was heard.
func TestATranscriptOfNoWordsIsRefused(t *testing.T) {
	for _, one := range []struct {
		what string
		body string
	}{
		{"a transcript naming no cues", `{"path":"` + talk + `"}`},
		{"a transcript of empty cues", edited(cue{Text: "  ", From: 1500, To: 4200})},
	} {
		t.Run(one.what, func(t *testing.T) {
			held := whole(spoke())
			_, handler := listeningTo(t, held)

			out := putting(handler, one.body)
			if out.Code != http.StatusBadRequest {
				t.Fatalf("the transcript was answered with %d: %s", out.Code, out.Body)
			}
			if _, kept := held[derived.Corrected(listener, hashed)]; kept {
				t.Error("a transcript of no words was written over the recording")
			}
			if told := heard(t, handler); len(told.Cues) != 2 {
				t.Errorf("the recording now says %+v", told.Cues)
			}
		})
	}
}

// swapping is a window whose vault is put away while a request is being
// answered, as it is when a person opens another one.
type swapping struct {
	port.VaultReaders
	then func()
}

func (s swapping) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := s.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return stating{VaultReader: reader, then: s.then}, nil
}

type stating struct {
	port.VaultReader
	then func()
}

func (s stating) Stat(ctx context.Context, path string) (domain.FileRef, error) {
	ref, err := s.VaultReader.Stat(ctx, path)
	s.then()
	return ref, err
}

// The vault a path belongs to is the vault it is cut again in. A window moved
// to another vault while the write was being made does not cut this recording
// there.
func TestATranscriptIsCutAgainInTheVaultItBelongsTo(t *testing.T) {
	held := whole(spoke())
	api, handler := listeningTo(t, held)
	standing := api.Showing()
	elsewhere := testsupport.NewVault(t, map[string]string{talk: sound})

	api.Readers = swapping{VaultReaders: filesystem.Readers{}, then: func() { api.show(elsewhere) }}

	var cutIn []domain.Vault
	runningBehind(api, func(on *showing) {
		on.cut = func(_ context.Context, v domain.Vault, _ string) error {
			cutIn = append(cutIn, v)
			return nil
		}
	})

	out := putting(handler, edited(cue{Text: "what Rupa said", From: 1500, To: 4200}))
	if out.Code != http.StatusOK {
		t.Fatalf("put the transcript right and got %d: %s", out.Code, out.Body)
	}
	if len(cutIn) != 1 {
		t.Fatalf("the recording was cut again in %v", cutIn)
	}
	if cutIn[0].ID != standing.ID {
		t.Errorf("the recording was cut again in %q, and it belongs to %q", cutIn[0].ID, standing.ID)
	}
}

// refusing is a store that will not take what a transcript was put right to.
type refusing struct {
	stored
	why error
}

func (r refusing) Open(domain.Vault) (port.DerivedStore, error) { return r, nil }

func (r refusing) Write(ctx context.Context, name string, content []byte) error {
	if name == derived.Corrected(listener, hashed) {
		return r.why
	}
	return r.stored.Write(ctx, name, content)
}

// A correction that could not be written is said so, and the source is not cut
// again from words that are not on disk.
func TestATranscriptThatCouldNotBeWrittenIsRefused(t *testing.T) {
	held := refusing{stored: whole(spoke()), why: errors.New("the disk is full")}
	api, handler := windowOn(t, held)

	var asked []string
	runningBehind(api, func(on *showing) {
		on.cut = func(_ context.Context, _ domain.Vault, path string) error {
			asked = append(asked, path)
			return nil
		}
	})

	out := putting(handler, edited(cue{Text: "what Rupa said", From: 1500, To: 4200}))
	if out.Code == http.StatusOK {
		t.Fatalf("a correction that was not written was answered %d: %s", out.Code, out.Body)
	}
	if len(asked) != 0 {
		t.Errorf("the source was cut again from words nothing holds: %v", asked)
	}
	if told := heard(t, handler); told.Cues[1].Text != "what was said next" {
		t.Errorf("the recording now says %+v", told.Cues)
	}
}

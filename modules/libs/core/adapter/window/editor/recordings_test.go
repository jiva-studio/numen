package editor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
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

// storing hands the one store to whatever opens a vault's.
type storing struct{ port.DerivedStore }

func (s storing) Open(domain.Vault) (port.DerivedStore, error) { return s.DerivedStore, nil }

// Open is one file of the store, which a copy of a video is played from.
func (s stored) Open(_ context.Context, name string) (io.ReadSeekCloser, int64, error) {
	raw, held := s[name]
	if !held {
		return nil, 0, fs.ErrNotExist
	}
	return readingBytes{bytes.NewReader(raw)}, int64(len(raw)), nil
}

// Take puts what a reader gives under a name.
func (s stored) Take(_ context.Context, name string, from io.Reader) (int64, error) {
	raw, err := io.ReadAll(from)
	if err != nil {
		return 0, err
	}
	s[name] = raw
	return int64(len(raw)), nil
}

// readingBytes is a reader of bytes already in memory, closed by nobody.
type readingBytes struct{ *bytes.Reader }

func (readingBytes) Close() error { return nil }

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

func (s stored) List(context.Context, string) ([]port.Entry, error) { return nil, nil }

func (s stored) Claim(context.Context, string) (func() error, error) {
	return func() error { return nil }, nil
}

// The model a test's words were heard by, and the bytes it heard them in.
const (
	asr    = derived.ASR
	hashed = "2fd4e1c6"
)

// listeningTo is a window holding one recording, with what a model wrote of it
// in the store. Nothing written down is a recording nobody has listened to.
func listeningTo(t *testing.T, held stored) (*API, http.Handler) {
	t.Helper()
	return windowOn(t, held)
}

// windowOn is the same window, with whatever store the test hands it.
func windowOn(t *testing.T, held port.DerivedStore) (*API, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{talk: sound, book: "the bytes of a scan"})
	api := &API{
		Readers: filesystem.VaultReaders{},
		Highlight: &source.Highlight{
			Sources: indexed{talk: {Fingerprint: domain.Fingerprint{Path: talk}, Producer: asr, Hash: hashed}},
			Derived: storing{held},
		},
	}
	api.show(vault)
	return api, api.Serving(http.NotFoundHandler())
}

// whole is the store holding a finished transcript of the recording, and partly
// is one a run is still writing.
func whole(cues []transcript.Cue) stored {
	return stored{derived.Artifact(asr, hashed): transcript.Marshal(cues)}
}

func partly(cues []transcript.Cue, reached int) stored {
	return stored{
		derived.Partial(asr, hashed): append(transcript.Marshal(cues), transcript.Heard(reached)...),
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

	out := ask(handler, back.Address(api.Showing(), statOf(t, api, api.Showing(), talk)))
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
	r := httptest.NewRequest(http.MethodGet, back.Address(api.Showing(), statOf(t, api, api.Showing(), talk)), nil)
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
			api, _ := listeningTo(t, one.held)

			told := heard(t, api)
			if len(told.GetCues()) != 2 {
				t.Fatalf("what was heard came back as %+v", told)
			}
			if first := told.GetCues()[0]; first.GetText() != "what was said" ||
				first.GetFrom() != 1500 || first.GetTo() != 4200 {
				t.Errorf("the first thing said came back as %+v", first)
			}
		})
	}
}

// A recording nobody has listened to holds no words, and is played all the
// same.
func TestARecordingNobodyHasListenedToHoldsNoWords(t *testing.T) {
	api, _ := listeningTo(t, nil)

	if told := heard(t, api); len(told.GetCues()) != 0 {
		t.Errorf("a recording nobody heard says %+v", told.GetCues())
	}
	back, playing := played(t, api)
	if out := ask(playing, back.Address(api.Showing(), statOf(t, api, api.Showing(), talk))); out.Code != http.StatusOK {
		t.Errorf("the recording itself was answered %d", out.Code)
	}
}

// What a recording is: how far the words reach, and how much of it a run has
// written down.
func TestWhatARecordingIsIsHowLongItRuns(t *testing.T) {
	for _, one := range []struct {
		what   string
		held   stored
		length int32
		heard  int32
	}{
		{"nothing heard", nil, 0, 0},
		{"a run that finished", whole(spoke()), 9100, 0},
		{"a run still going", partly(spoke()[:1], 4200), 4200, 4200},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, _ := listeningTo(t, one.held)

			out, err := api.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{
				Path: talk,
			}))
			if err != nil {
				t.Fatalf("asked what the recording is and was refused: %v", err)
			}
			told := out.Msg
			if told.GetLength() != one.length || told.GetHeard() != one.heard {
				t.Errorf("the recording came back as %+v, want %d long and %d heard",
					told, one.length, one.heard)
			}
		})
	}
}

// What was heard is a recording's. A document holds no words and is asked for
// none, and neither is a path the vault does not hold.
func TestOnlyARecordingIsHeard(t *testing.T) {
	api, _ := listeningTo(t, whole(spoke()))

	for path, want := range map[string]connect.Code{
		book:             connect.CodeInvalidArgument,
		"talks/none.mp3": connect.CodeNotFound,
	} {
		_, err := api.ReadTranscript(t.Context(), connect.NewRequest(&v1.ReadTranscriptRequest{
			Path: path,
		}))
		if connect.CodeOf(err) != want {
			t.Errorf("%s was refused %v", path, err)
		}
	}
}

// A build that mounts nothing about what is made from a file plays the
// recording all the same: the bytes are served over a socket of their own and
// not over the service that says what was heard in them.
func TestABuildThatServesNoArtifactPlaysTheRecording(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{talk: sound})
	api := &API{Readers: filesystem.VaultReaders{}}
	api.show(vault)
	_ = api.Serving(http.NotFoundHandler(), numenv1connect.VaultServiceName)

	back, playing := played(t, api)
	if out := ask(playing, back.Address(api.Showing(), statOf(t, api, api.Showing(), talk))); out.Code != http.StatusOK {
		t.Errorf("the recording itself was answered %d", out.Code)
	}
}

// A question about a run of the words is answered with the speech those bytes
// were said in, so a search hit is played from where it was said.
func TestCuesNarrowToARunOfTheWords(t *testing.T) {
	spoke := []transcript.Cue{
		{Text: "first", From: 0, To: 1000},
		{Text: "second", From: 1000, To: 2000},
		{Text: "third", From: 2000, To: 3000},
	}
	_, cues := transcript.Parse(transcript.Marshal(spoke))

	// "second" begins after "first\n".
	got, err := within(&v1.Stretch{Start: 6, Length: 6})(cues)
	if err != nil {
		t.Fatalf("a run of the words was refused: %v", err)
	}
	if len(got) != 1 || got[0].From != 1000 {
		t.Errorf("the run was placed at %v", got)
	}

	if _, err := within(&v1.Stretch{Start: -1, Length: 6})(cues); err == nil {
		t.Errorf("a place before the words was taken")
	}
}

// A question naming no run at all is about the whole transcript.
func TestAQuestionNamingNoRunIsAboutTheWholeTranscript(t *testing.T) {
	api, _ := listeningTo(t, whole(spoke()))

	if told := heard(t, api); len(told.GetCues()) != 2 {
		t.Errorf("the whole transcript came back as %+v", told.GetCues())
	}
}

// heldBy is a store with a run holding one of its names, as a transcription
// holds the recording it is writing down.
type heldBy struct {
	stored
	name string
}

func (h heldBy) Claim(ctx context.Context, name string) (func() error, error) {
	if name == h.name {
		return nil, fmt.Errorf("%s: %w", name, port.ErrClaimed)
	}
	return h.stored.Claim(ctx, name)
}

// putting sends a transcript as the window sends one.
func putting(api *API, cues ...*v1.Cue) error {
	_, err := api.WriteTranscript(context.Background(), connect.NewRequest(&v1.WriteTranscriptRequest{
		Path: talk, Cues: cues,
	}))
	return err
}

// cueOf is one cue as the window sends one.
func cueOf(text string, from, to int32) *v1.Cue {
	return &v1.Cue{Text: text, From: from, To: to}
}

// What a transcript was put right to is kept beside what was heard, and what
// was heard stays where it is.
func TestATranscriptPutRightIsKeptBesideWhatWasHeard(t *testing.T) {
	held := whole(spoke())
	api, _ := listeningTo(t, held)

	if err := putting(api,
		cueOf("what was said", 1500, 4200),
		cueOf("what Rupa said next", 4200, 9100),
	); err != nil {
		t.Fatalf("put the transcript right and was refused: %v", err)
	}

	put, kept := held[derived.Corrections(asr, hashed)]
	if !kept {
		t.Fatalf("nothing was kept beside the transcript: %v", held)
	}
	if _, cues := transcript.Parse(put); len(cues) != 2 || cues[1].Text != "what Rupa said next" {
		t.Errorf("the transcript was put right to %q", put)
	}
	if _, cues := transcript.Parse(held[derived.Artifact(asr, hashed)]); cues[1].Text != "what was said next" {
		t.Error("what was heard was written over")
	}
}

// What a person wrote says so, so a proofreader leaves it alone.
func TestATranscriptAPersonWroteSaysSo(t *testing.T) {
	held := whole(spoke())
	api, _ := listeningTo(t, held)

	if err := putting(api, cueOf("what Rupa said", 1500, 4200)); err != nil {
		t.Fatalf("put the transcript right and was refused: %v", err)
	}
	if !transcript.Written(held[derived.Corrections(asr, hashed)]) {
		t.Errorf("the transcript does not say a person wrote it:\n%s", held[derived.Corrections(asr, hashed)])
	}
}

// The chunks in the index hold the words as they were heard, so a transcript
// put right is a source asked for again.
func TestATranscriptPutRightIsCutAgain(t *testing.T) {
	api, _ := listeningTo(t, whole(spoke()))

	var asked []string
	runningBehind(api, func(on *passes) {
		on.cut = func(_ context.Context, _ domain.Vault, path string) error {
			asked = append(asked, path)
			return nil
		}
	})

	if err := putting(api, cueOf("what was said", 1500, 4200)); err != nil {
		t.Fatalf("put the transcript right and was refused: %v", err)
	}
	if len(asked) != 1 || asked[0] != talk {
		t.Errorf("the sources cut again are %v", asked)
	}
}

// A cut that could not be asked for is not a write that failed: the correction
// is on disk either way.
func TestATranscriptStandsWhenItCannotBeCutAgain(t *testing.T) {
	held := whole(spoke())
	api, _ := listeningTo(t, held)
	runningBehind(api, func(on *passes) {
		on.cut = func(context.Context, domain.Vault, string) error {
			return errors.New("nothing is cutting")
		}
	})

	if err := putting(api, cueOf("what was said", 1500, 4200)); err != nil {
		t.Fatalf("put the transcript right and was refused: %v", err)
	}
	if _, kept := held[derived.Corrections(asr, hashed)]; !kept {
		t.Error("the correction was not kept")
	}
}

// The window is told the transcript as it now stands, and taking away what it
// was put right to gives back what was heard.
func TestATranscriptPutRightIsWhatTheWindowIsToldNext(t *testing.T) {
	held := whole(spoke())
	api, _ := listeningTo(t, held)

	if err := putting(api,
		cueOf("what was said", 1500, 4200),
		cueOf("what Rupa said next", 4200, 9100),
	); err != nil {
		t.Fatalf("put the transcript right and was refused: %v", err)
	}
	if told := heard(t, api); told.GetCues()[1].GetText() != "what Rupa said next" {
		t.Errorf("the window is told %+v", told.GetCues())
	}

	delete(held, derived.Corrections(asr, hashed))
	if told := heard(t, api); told.GetCues()[1].GetText() != "what was said next" {
		t.Errorf("what was heard did not come back: %+v", told.GetCues())
	}
}

// heard is the transcript the window is told about.
func heard(t *testing.T, api *API) *v1.ReadTranscriptResponse {
	t.Helper()
	out, err := api.ReadTranscript(t.Context(), connect.NewRequest(&v1.ReadTranscriptRequest{
		Path: talk,
	}))
	if err != nil {
		t.Fatalf("asked what was heard and was refused: %v", err)
	}
	return out.Msg
}

// Speech runs forward, and a transcript that says otherwise was not cut from a
// recording. Nothing of it is written.
func TestATranscriptThatRunsBackwardsIsRefused(t *testing.T) {
	for _, one := range []struct {
		what string
		cues []*v1.Cue
	}{
		{"a cue ending before it began", []*v1.Cue{cueOf("said", 4200, 1500)}},
		{"a cue before the one above it", []*v1.Cue{
			cueOf("said", 4200, 9100),
			cueOf("said next", 1500, 2000),
		}},
		{"a cue overlapping the one above it", []*v1.Cue{
			cueOf("said", 1500, 4200),
			cueOf("said next", 3000, 9100),
		}},
		{"a cue beginning before the recording", []*v1.Cue{cueOf("said", -1, 4200)}},
		// One cue is one line, and the window edits it as one.
		{"a cue broken over two lines", []*v1.Cue{cueOf("said\nand said next", 1500, 4200)}},
		{"a cue carrying a carriage return", []*v1.Cue{cueOf("said\rand said next", 1500, 4200)}},
	} {
		t.Run(one.what, func(t *testing.T) {
			held := whole(spoke())
			api, _ := listeningTo(t, held)

			err := putting(api, one.cues...)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("the transcript was answered with %v", err)
			}
			if err.Error() == "" {
				t.Error("the transcript was refused without saying why")
			}
			if _, kept := held[derived.Corrections(asr, hashed)]; kept {
				t.Error("a transcript that was refused was written")
			}
		})
	}
}

// A cue whose words trim away is silence, and Marshal writes none.
func TestACueWithNoWordsIsDropped(t *testing.T) {
	held := whole(spoke())
	api, _ := listeningTo(t, held)

	if err := putting(api,
		cueOf("what was said", 1500, 4200),
		cueOf("   ", 4200, 6000),
		cueOf("what was said last", 6000, 9100),
	); err != nil {
		t.Fatalf("put the transcript right and was refused: %v", err)
	}
	if _, cues := transcript.Parse(held[derived.Corrections(asr, hashed)]); len(cues) != 2 {
		t.Errorf("the transcript was written down as %v", cues)
	}
}

// A run appends to the transcript, and what is being appended to is not edited
// underneath.
func TestATranscriptIsNotEditedWhileTheRecordingIsBeingListenedTo(t *testing.T) {
	held := heldBy{stored: partly(spoke(), 9100), name: derived.Partial(asr, hashed)}
	api, _ := windowOn(t, held)

	err := putting(api, cueOf("what was said", 1500, 4200))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("edited a recording being listened to and was refused %v", err)
	}
	if _, kept := held.stored[derived.Corrections(asr, hashed)]; kept {
		t.Error("the transcript was written while a run held the recording")
	}

	if told := heard(t, api); told.GetEditable() {
		t.Error("the window was told it may edit a transcript a run is writing")
	}
}

// A transcript nothing is writing may be put right, and the window is told so.
func TestTheWindowIsToldATranscriptMayBePutRight(t *testing.T) {
	api, _ := listeningTo(t, whole(spoke()))

	if told := heard(t, api); !told.GetEditable() {
		t.Error("the window was told it may not edit a transcript nothing is writing")
	}
}

// A recording nobody has listened to has no transcript to put right.
func TestARecordingNobodyHasListenedToHasNoTranscriptToPutRight(t *testing.T) {
	held := stored{}
	api, _ := listeningTo(t, held)
	api.Highlight.Sources = indexed{}

	err := putting(api, cueOf("what was said", 1500, 4200))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("put right a recording nothing heard and was refused %v", err)
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
		cues []*v1.Cue
	}{
		{"a transcript naming no cues", nil},
		{"a transcript of empty cues", []*v1.Cue{cueOf("  ", 1500, 4200)}},
	} {
		t.Run(one.what, func(t *testing.T) {
			held := whole(spoke())
			api, _ := listeningTo(t, held)

			err := putting(api, one.cues...)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("the transcript was answered with %v", err)
			}
			if _, kept := held[derived.Corrections(asr, hashed)]; kept {
				t.Error("a transcript of no words was written over the recording")
			}
			if told := heard(t, api); len(told.GetCues()) != 2 {
				t.Errorf("the recording now says %+v", told.GetCues())
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

func (s stating) Stat(ctx context.Context, path string) (domain.Fingerprint, error) {
	ref, err := s.VaultReader.Stat(ctx, path)
	s.then()
	return ref, err
}

// The vault a path belongs to is the vault it is cut again in. A window moved
// to another vault while the write was being made does not cut this recording
// there.
func TestATranscriptIsCutAgainInTheVaultItBelongsTo(t *testing.T) {
	held := whole(spoke())
	api, _ := listeningTo(t, held)
	showing := api.Showing()
	elsewhere := testsupport.NewVault(t, map[string]string{talk: sound})

	api.Readers = swapping{VaultReaders: filesystem.VaultReaders{}, then: func() { api.show(elsewhere) }}

	var cutIn []domain.Vault
	runningBehind(api, func(on *passes) {
		on.cut = func(_ context.Context, v domain.Vault, _ string) error {
			cutIn = append(cutIn, v)
			return nil
		}
	})

	if err := putting(api, cueOf("what Rupa said", 1500, 4200)); err != nil {
		t.Fatalf("put the transcript right and was refused: %v", err)
	}
	if len(cutIn) != 1 {
		t.Fatalf("the recording was cut again in %v", cutIn)
	}
	if cutIn[0].ID != showing.ID {
		t.Errorf("the recording was cut again in %q, and it belongs to %q", cutIn[0].ID, showing.ID)
	}
}

// refusing is a store that will not take what a transcript was put right to.
type refusing struct {
	stored
	why error
}

func (r refusing) Write(ctx context.Context, name string, content []byte) error {
	if name == derived.Corrections(asr, hashed) {
		return r.why
	}
	return r.stored.Write(ctx, name, content)
}

// A correction that could not be written is said so, and the source is not cut
// again from words that are not on disk.
func TestATranscriptThatCouldNotBeWrittenIsRefused(t *testing.T) {
	held := refusing{stored: whole(spoke()), why: errors.New("the disk is full")}
	api, _ := windowOn(t, held)

	var asked []string
	runningBehind(api, func(on *passes) {
		on.cut = func(_ context.Context, _ domain.Vault, path string) error {
			asked = append(asked, path)
			return nil
		}
	})

	if err := putting(api, cueOf("what Rupa said", 1500, 4200)); err == nil {
		t.Fatal("a correction that was not written was answered as one that was")
	}
	if len(asked) != 0 {
		t.Errorf("the source was cut again from words nothing holds: %v", asked)
	}
	if told := heard(t, api); told.GetCues()[1].GetText() != "what was said next" {
		t.Errorf("the recording now says %+v", told.GetCues())
	}
}

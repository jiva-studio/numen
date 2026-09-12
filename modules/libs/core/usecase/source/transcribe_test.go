package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A voice is a recording made of spans given in advance, and the model that
// hears them. It counts the spans it was handed, so a test can say that a
// run taken up again did not hear one twice.
type voice struct {
	words []string
	// refuse is what opening the recording answers, standing for a file in a
	// container nothing here reads.
	refuse error

	opens int
	heard []int // the millisecond each span handed over began at
	stop  func(int)
}

// getSpan is where the words spoken n-th sit in the recording. Every span is
// a second long with a fifth of a second of silence after it.
func getSpan(n int) port.Audio {
	return port.Audio{From: n * 1000, To: n*1000 + 800}
}

func (s *voice) Transcription() port.TranscriptionModel {
	return port.TranscriptionModel{Model: "ear", Segmenter: "pauses", Cutting: "0.50/500/200/30000/100/2500", From: "a test"}
}

func (s *voice) Open(_ context.Context, _ []byte) (port.Recording, error) {
	if s.refuse != nil {
		return nil, s.refuse
	}
	s.opens++
	return played{s}, nil
}

func (s *voice) Transcribe(ctx context.Context, audio port.Audio) (string, error) {
	s.heard = append(s.heard, audio.From)
	if s.stop != nil {
		s.stop(len(s.heard))
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return s.words[audio.From/1000], nil
}

func (s *voice) Close() error { return nil }

// played is the voice's recording, opened.
type played struct{ *voice }

func (p played) Length() int { return len(p.words) * 1000 }

func (p played) Segments(_ context.Context, from, count int) ([]port.Audio, error) {
	var out []port.Audio
	for n := range p.words {
		if at := getSpan(n); at.From >= from {
			out = append(out, at)
			if len(out) == count {
				break
			}
		}
	}
	return out, nil
}

func (p played) Close() error { return nil }

// recordingPath is where the recording sits in the vault under test.
const recordingPath = "library/talk.mp3"

// recorded is the bytes of the file the words were spoken into. Nothing here
// reads them: what the recording says comes from the model, and the model is a
// fake.
func recorded(words []string) []byte {
	return []byte("a recording of " + strings.Join(words, "|"))
}

// listener is a Transcribe over one vault holding one recording, and the hash
// its artifact is named by.
func listener(t *testing.T, words ...string) (Transcribe, domain.Vault, *store, *shelf, *voice, string) {
	t.Helper()
	raw := recorded(words)
	shelved := newLibrary()
	shelved.hold(recordingPath, domain.KindRecording, raw, 1)
	index := newStore()
	kept := newShelf()
	model := &voice{words: words}
	return Transcribe{
		Readers: vaults{first.ID: shelved},
		Sources: index,
		Derived: shelves{kept},
		By:      model,
		Batch:   1,
	}, first, index, kept, model, text.Fingerprint(raw)
}

// spoken is the words of an artifact, in the order they were said.
func spoken(t *testing.T, raw []byte) []string {
	t.Helper()
	_, cues := transcript.Parse(raw)
	out := make([]string, 0, len(cues))
	for _, cue := range cues {
		out = append(out, cue.Text)
	}
	return out
}

func TestWhatIsHeardIsWrittenDownAndClaimed(t *testing.T) {
	u, v, index, shelf, model, hash := listener(t, "first thing", "second thing", "third thing")

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Length != 3000 {
		t.Errorf("the recording is %d ms long", res.Length)
	}
	if res.Heard != getSpan(2).To {
		t.Errorf("heard %d ms of it", res.Heard)
	}
	if res.Silent || res.Unopened || res.Busy {
		t.Errorf("a recording that was written down came back as %+v", res)
	}

	src := index.sources[v.ID][recordingPath]
	if src.Producer != "asr" {
		t.Fatalf("the source says its text comes from %q", src.Producer)
	}
	if src.Hash != hash {
		t.Errorf("the source is named %q and the recording hashes to %q", src.Hash, hash)
	}
	raw, err := shelf.Read(t.Context(), text.Artifact(src.Producer, src.Hash))
	if err != nil {
		t.Fatalf("the artifact is not where the source says: %v", err)
	}
	if !bytes.HasPrefix(raw, []byte(transcript.Head+"\n")) {
		t.Fatalf("the artifact is not WebVTT: %q", raw)
	}
	if said := spoken(t, raw); !slices.Equal(said, model.words) {
		t.Errorf("the artifact says %q and the recording says %q", said, model.words)
	}
	if _, err := shelf.Read(t.Context(), text.Partial("asr", hash)); err == nil {
		t.Error("the partial is still there")
	}
}

func TestARunThatStoppedIsTakenUpWhereItStopped(t *testing.T) {
	// An hour of speech is an hour of listening, and a person may stop one.
	// What was heard is on disk, and the next run begins where this one
	// stopped.
	u, v, _, shelf, model, hash := listener(t, "one", "two", "three", "four")

	ctx, stop := context.WithCancel(t.Context())
	model.stop = func(heard int) {
		if heard == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopping gave %v", err)
	}
	// A half-heard recording is not an artifact, and nothing reads it as one.
	if _, err := shelf.Read(t.Context(), text.Artifact("asr", hash)); err == nil {
		t.Fatal("a stopped run left an artifact behind")
	}

	model.stop = nil
	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != getSpan(0).To {
		t.Errorf("the second run began at %d ms", res.Resumed)
	}

	raw, err := shelf.Read(t.Context(), text.Artifact("asr", hash))
	if err != nil {
		t.Fatal(err)
	}
	if said := spoken(t, raw); !slices.Equal(said, model.words) {
		t.Errorf("the artifact says %q and the recording says %q", said, model.words)
	}
	// The stretch the stopped run did not write down is the only one heard
	// twice.
	if want := []int{0, 1000, 1000, 2000, 3000}; !slices.Equal(model.heard, want) {
		t.Errorf("the model was handed %v and there are %v to hand it", model.heard, want)
	}
}

func TestABatchThatDidNotLandWholeIsCutBack(t *testing.T) {
	// A note stands after the cues it claims, so cues written after the last
	// note are a batch nothing claims.
	u, v, _, shelf, model, hash := listener(t, "one", "two", "three")

	torn := transcript.Marshal([]transcript.Cue{{Text: "one", From: 0, To: 800}})
	torn = append(torn, transcript.Reaches(800)...)
	loose := transcript.Marshal([]transcript.Cue{{Text: "half a thought", From: 1000, To: 1800}})
	torn = append(torn, bytes.TrimPrefix(loose, []byte(transcript.Head+"\n"))...)
	if err := shelf.Write(t.Context(), text.Partial("asr", hash), torn); err != nil {
		t.Fatal(err)
	}

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 800 {
		t.Errorf("the run began at %d ms and the note says 800", res.Resumed)
	}

	raw, err := shelf.Read(t.Context(), text.Artifact("asr", hash))
	if err != nil {
		t.Fatal(err)
	}
	if said := spoken(t, raw); !slices.Equal(said, model.words) {
		t.Errorf("the artifact says %q and the recording says %q", said, model.words)
	}
	if bytes.Contains(raw, []byte("half a thought")) {
		t.Error("the artifact holds a cue no note claimed")
	}
}

func TestOneRunToARecording(t *testing.T) {
	u, v, index, shelf, model, hash := listener(t, "one", "two")
	shelf.hold(text.Partial("asr", hash))

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Busy {
		t.Error("a recording another run holds was listened to")
	}
	if len(model.heard) != 0 {
		t.Errorf("the model was handed %d stretches", len(model.heard))
	}
	if names := shelf.names(); len(names) != 0 {
		t.Errorf("it wrote %v", names)
	}
	if _, held := index.sources[v.ID][recordingPath]; held {
		t.Error("it recorded a source")
	}
}

func TestARecordingWithNothingToHearIsAnsweredOnce(t *testing.T) {
	// A queue hands over every recording it finds. Silence is what the
	// recording had to say, and saying it again changes nothing.
	u, v, index, shelf, model, hash := listener(t, "", "", "")

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Silent {
		t.Error("a recording carrying no speech was not reported as silent")
	}
	for _, name := range []string{text.Artifact("asr", hash), text.Partial("asr", hash)} {
		if _, err := shelf.Read(t.Context(), name); err == nil {
			t.Errorf("it left %q behind", name)
		}
	}
	if src := index.sources[v.ID][recordingPath]; src.Producer != "" {
		t.Errorf("the source was pointed at %q", src.Producer)
	}

	heard := len(model.heard)
	again, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Silent {
		t.Error("the answer was not read back")
	}
	if len(model.heard) != heard {
		t.Errorf("the model was handed %d more stretches", len(model.heard)-heard)
	}
	if _, err := shelf.Read(t.Context(), text.Answer("asr", hash)); err != nil {
		t.Errorf("nothing says what the recording answered: %v", err)
	}
}

func TestARecordingNothingCanOpenIsAnsweredOnce(t *testing.T) {
	u, v, index, shelf, model, hash := listener(t, "one", "two")
	model.refuse = errors.New("no reader for this container")

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unopened {
		t.Error("a recording nothing can open was not reported as such")
	}
	if src := index.sources[v.ID][recordingPath]; src.Producer != "" {
		t.Errorf("the source was pointed at %q", src.Producer)
	}
	raw, err := shelf.Read(t.Context(), text.Answer("asr", hash))
	if err != nil {
		t.Fatalf("nothing says what the recording answered: %v", err)
	}
	if !strings.Contains(string(raw), "no reader for this container") {
		t.Errorf("the answer says %q", raw)
	}

	// The file will not open on any run, and a queue is told once.
	again, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Unopened || again.Silent {
		t.Errorf("the answer came back as %+v", again)
	}
	if model.opens != 0 {
		t.Errorf("the recording was opened %d times", model.opens)
	}
}

func TestWhatHasBeenHeardIsCutBeforeTheRestIs(t *testing.T) {
	// A recording is searchable as it is listened to, so cutting happens as the
	// speech is written down.
	u, v, _, _, model, _ := listener(t, "one", "two", "three")

	var cuts, at []int
	u.Cut = func(_ context.Context, _ domain.Vault, path string) error {
		if path != recordingPath {
			return fmt.Errorf("cut %s", path)
		}
		cuts = append(cuts, len(model.heard))
		return nil
	}
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	// One cut to a batch, and one more when the source is stood on the
	// artifact.
	for n := range model.words {
		at = append(at, n+1)
	}
	if want := slices.Concat(at, []int{len(model.words)}); !slices.Equal(cuts, want) {
		t.Errorf("it cut after %v stretches and the batches end at %v", cuts, want)
	}
}

func TestWhatListenedIsKeptBesideWhatItHeard(t *testing.T) {
	u, v, _, shelf, model, hash := listener(t, "one", "two")

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	raw, err := shelf.Read(t.Context(), text.Beside("asr", hash))
	if err != nil {
		t.Fatalf("nothing says what listened: %v", err)
	}
	if !bytes.Contains(raw, []byte(model.Transcription().Recipe())) {
		t.Errorf("what is kept beside the artifact says %q", raw)
	}
}

// A transcript is kept and never made twice, so a person who changed the model
// has to be able to say "hear this one again". Nothing sets it on its own.
func TestAskingForARecordingToBeHeardAgain(t *testing.T) {
	u, v, _, shelf, model, hash := listener(t, "one", "two")

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	first := len(model.heard)
	if first == 0 {
		t.Fatal("the model was handed nothing")
	}

	// Asked again without saying so, the transcript stands and the model is
	// left alone.
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if len(model.heard) != first {
		t.Errorf("the model was handed %d more stretches", len(model.heard)-first)
	}

	u.Again = true
	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.heard) != first*2 {
		t.Errorf("asked again, the model was handed %d stretches, want %d", len(model.heard), first*2)
	}
	if res.Resumed != 0 {
		t.Errorf("asked again, it took up a run at %d ms", res.Resumed)
	}
	raw, err := shelf.Read(t.Context(), text.Artifact("asr", hash))
	if err != nil {
		t.Fatalf("nothing was written the second time: %v", err)
	}
	if said := spoken(t, raw); len(said) != 2 {
		t.Errorf("the transcript reads back as %v", said)
	}
}

// An answer stops a recording being offered again, so asking for it again has
// to get past it. What it says the second time is whatever it says.
func TestAskingAgainAfterAnAnswer(t *testing.T) {
	u, v, _, shelf, model, hash := listener(t, "", "", "")
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if _, err := shelf.Read(t.Context(), text.Answer("asr", hash)); err != nil {
		t.Fatalf("the recording left no answer: %v", err)
	}
	first := len(model.heard)

	u.Again = true
	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.heard) <= first {
		t.Error("the answer stood, and the recording was not heard again")
	}
	if !res.Silent {
		t.Error("a recording still carrying no speech was not answered")
	}
}

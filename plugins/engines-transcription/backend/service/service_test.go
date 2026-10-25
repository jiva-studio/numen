package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

const (
	lecture      = "assets/lecture.wav"
	lectureSound = "RIFF the bytes of a lecture"
)

// heardModel hears every stretch as the words given for the moment it begins, and records each
// moment a recording was asked for segments from.
type heardModel struct {
	mu            sync.Mutex
	segments      []engine.Audio
	words         map[int]string
	opened        int
	segmentStarts []int
	isBroken      bool
}

func (m *heardModel) Open(context.Context, []byte) (engine.Recording, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isBroken {
		return nil, errors.New("no decoder reads this")
	}
	m.opened++
	return heardRecording{model: m}, nil
}

func (m *heardModel) Transcribe(_ context.Context, audio engine.Audio) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.words[audio.From], nil
}

func (m *heardModel) Close() error { return nil }

func (m *heardModel) getSegmentStarts() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.segmentStarts)
}

type heardRecording struct {
	model *heardModel
}

func (r heardRecording) GetDuration() int {
	r.model.mu.Lock()
	defer r.model.mu.Unlock()
	return r.model.segments[len(r.model.segments)-1].To
}

func (r heardRecording) GetSegments(_ context.Context, from, count int) ([]engine.Audio, error) {
	r.model.mu.Lock()
	defer r.model.mu.Unlock()
	r.model.segmentStarts = append(r.model.segmentStarts, from)
	var out []engine.Audio
	for _, one := range r.model.segments {
		if one.From >= from && len(out) < count {
			out = append(out, one)
		}
	}
	return out, nil
}

func (heardRecording) Close() error { return nil }

// scene is the plugin over a vault, its store, its texts and the queue, all in memory.
type scene struct {
	vault       *hosttest.Vault
	store       host.Store
	texts       *hosttest.Texts
	queue       *hosttest.Queue
	model       *heardModel
	transcriber *Transcriber
	provider    provider
	transcripts transcriptServer
}

func newScene(t *testing.T) *scene {
	t.Helper()
	vault := hosttest.New("Lectures")
	vault.File(lecture, lectureSound)
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.File(lecture, []byte(lectureSound))
	s := &scene{
		vault: vault, store: vault.Store(), texts: texts, queue: hosttest.NewQueue(),
		model: &heardModel{
			segments: []engine.Audio{{From: 0, To: 1000}, {From: 1000, To: 2500}},
			words:    map[int]string{0: "first words", 1000: "second words"},
		},
	}
	s.transcriber = NewTranscriber(vaultFiles{vault: vault}, s.store, texts, s.queue,
		func(context.Context) (engine.TranscriberPort, error) { return s.model, nil })
	t.Cleanup(s.transcriber.Close)
	s.provider = provider{stores: s.store, transcriber: s.transcriber}
	texts.Provide(PluginID, s.provider)
	s.transcripts = transcriptServer{stores: s.store, texts: texts}
	return s
}

func (s *scene) transcribe(t *testing.T) {
	t.Helper()
	if err := s.provider.CreateText(t.Context(), host.TextCreation{Vault: s.vault.Vault().ID, Path: lecture}); err != nil {
		t.Fatal(err)
	}
	s.transcriber.Wait()
}

func (s *scene) readLive(t *testing.T) (domain.TextVersion, string) {
	t.Helper()
	listing, err := s.texts.ListVersions(t.Context(), lecture)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.GetID() == listing.Live })
	if i < 0 {
		t.Fatalf("no version of %s is live: %+v", lecture, listing)
	}
	text, err := host.ReadWholeText(t.Context(), s.texts, listing.Versions[i])
	if err != nil {
		t.Fatal(err)
	}
	return listing.Versions[i], text.Text
}

func (s *scene) readCues(t *testing.T, span *numenv1.Span) []*numenv1.Cue {
	t.Helper()
	answer, err := s.transcripts.ReadTranscript(t.Context(), connect.NewRequest(&numenv1.ReadTranscriptRequest{Path: lecture, Span: span}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg.GetCues()
}

func TestCreateTextTimedVersionInStore(t *testing.T) {
	s := newScene(t)
	s.transcribe(t)

	version, text := s.readLive(t)
	fileHash := domain.HashContent([]byte(lectureSound))
	if version.Format != domain.TextFormatTimed || version.Key != fileHash || version.InputHash != fileHash || !version.IsFromFile() {
		t.Errorf("the version is %+v, want a timed version from the file under its hash", version)
	}
	if text != "first words\nsecond words" {
		t.Errorf("the version reads %q", text)
	}
	if version.Hash != domain.HashContent([]byte(text)) {
		t.Errorf("the version says its text hashes to %s", version.Hash)
	}
	listed, err := s.store.List(t.Context())
	if err != nil || len(listed) != 1 || listed[0].Name != fileHash+".vtt" {
		t.Errorf("the folder holds %v (%v), want the whole transcript alone", listed, err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are still shown", tasks)
	}
}

func TestCreateTextSkipsWholeTranscript(t *testing.T) {
	s := newScene(t)
	s.transcribe(t)
	s.transcribe(t)

	if s.model.opened != 1 {
		t.Errorf("the recording was opened %d times, want once", s.model.opened)
	}
}

func TestCreateTextResumesFromPartialTranscript(t *testing.T) {
	s := newScene(t)
	key := domain.HashContent([]byte(lectureSound))
	partial := append(transcript.Marshal([]transcript.Cue{{Text: "heard before", From: 0, To: 1000}}), transcript.GetReachMarker(1000)...)
	if err := s.store.Write(t.Context(), getPartialName(key), append(partial, []byte("\n00:00:01.000 --> 00:0")...)); err != nil {
		t.Fatal(err)
	}

	s.transcribe(t)

	if starts := s.model.getSegmentStarts(); len(starts) == 0 || starts[0] != 1000 {
		t.Errorf("the recording was asked for segments from %v, want from 1000 first", starts)
	}
	if _, text := s.readLive(t); text != "heard before\nsecond words" {
		t.Errorf("the version reads %q", text)
	}
}

func TestCreateTextReportsUnopenableRecording(t *testing.T) {
	s := newScene(t)
	s.model.isBroken = true

	s.transcribe(t)

	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].About != lecture || !strings.Contains(tasks[0].Error, "cannot be opened") {
		t.Errorf("the tasks are %+v, want one saying the recording cannot be opened", tasks)
	}
	if listing, _ := s.texts.ListVersions(t.Context(), lecture); len(listing.Versions) != 0 {
		t.Errorf("a recording that could not be opened has versions %+v", listing.Versions)
	}
}

func TestCreateTextWaitsForQueueTurn(t *testing.T) {
	s := newScene(t)
	held, err := s.queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.provider.CreateText(t.Context(), host.TextCreation{Vault: s.vault.Vault().ID, Path: lecture}); err != nil {
		t.Fatal(err)
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].About != lecture {
		t.Fatalf("the tasks are %+v once the transcription was asked for, want it shown", tasks)
	}
	held.Release()
	s.transcriber.Wait()

	if _, text := s.readLive(t); text != "first words\nsecond words" {
		t.Errorf("the version reads %q", text)
	}
}

func TestCreateTextRejectsInputVersion(t *testing.T) {
	s := newScene(t)

	err := s.provider.CreateText(t.Context(), host.TextCreation{Vault: s.vault.Vault().ID, Path: lecture, Input: "a/b"})

	if !errors.Is(err, host.ErrInvalidVersion) {
		t.Errorf("a request naming an input was answered %v", err)
	}
}

func TestReadTranscriptReturnsLiveCuesAndSpanCue(t *testing.T) {
	s := newScene(t)
	s.transcribe(t)

	cues := s.readCues(t, nil)
	if len(cues) != 2 || cues[0].GetText() != "first words" || cues[1].GetFrom() != 1000 || cues[1].GetTo() != 2500 {
		t.Fatalf("the transcript reads %v", cues)
	}
	// "second" begins after "first words\n".
	if got := s.readCues(t, &numenv1.Span{From: 12, To: 18}); len(got) != 1 || got[0].GetFrom() != 1000 {
		t.Errorf("the span is in %v, want the second cue", got)
	}
}

func TestWriteTranscriptEditVersion(t *testing.T) {
	s := newScene(t)
	s.transcribe(t)
	heard, _ := s.readLive(t)

	edit := func(text string) {
		t.Helper()
		_, err := s.transcripts.WriteTranscript(t.Context(), connect.NewRequest(&numenv1.WriteTranscriptRequest{
			Path: lecture, Cues: []*numenv1.Cue{{Text: "first words", From: 0, To: 1000}, {Text: text, From: 1000, To: 2500}},
		}))
		if err != nil {
			t.Fatal(err)
		}
	}
	edit("second words, put right")
	version, text := s.readLive(t)
	if version.Input != heard.GetID() || version.InputHash != heard.Hash || version.PluginID != PluginID {
		t.Errorf("the edit is %+v, want a version made from %s", version, heard.GetID())
	}
	if text != "first words\nsecond words, put right" {
		t.Errorf("the edit reads %q", text)
	}

	edit("second words, put right again")
	again, text := s.readLive(t)
	listing, _ := s.texts.ListVersions(t.Context(), lecture)
	if again.GetID() != version.GetID() || again.Input != heard.GetID() || len(listing.Versions) != 2 {
		t.Errorf("an edit of the edit is %+v among %d versions, want the edit rewritten", again, len(listing.Versions))
	}
	if text != "first words\nsecond words, put right again" {
		t.Errorf("the edit reads %q", text)
	}
}

func TestWriteTranscriptRejectsRecordingWithoutTranscript(t *testing.T) {
	s := newScene(t)

	_, err := s.transcripts.WriteTranscript(t.Context(), connect.NewRequest(&numenv1.WriteTranscriptRequest{
		Path: lecture, Cues: []*numenv1.Cue{{Text: "words", From: 0, To: 1000}},
	}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("an edit of nothing was answered %v", err)
	}
	if cues := s.readCues(t, nil); len(cues) != 0 {
		t.Errorf("a recording with no version reads %v", cues)
	}
}

func TestDeleteTextRemovesStoreFiles(t *testing.T) {
	s := newScene(t)
	s.transcribe(t)
	version, _ := s.readLive(t)

	if err := s.provider.DeleteText(t.Context(), s.vault.Vault().ID, lecture, version.Key); err != nil {
		t.Fatal(err)
	}

	if listed, err := s.store.List(t.Context()); err != nil || len(listed) != 0 {
		t.Errorf("the folder holds %v (%v) after the text was deleted", listed, err)
	}
	if _, err := s.provider.GetText(t.Context(), host.TextRequest{Vault: s.vault.Vault().ID, Path: lecture, Key: version.Key}); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a deleted text was read with %v", err)
	}
}

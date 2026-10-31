package service

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

const (
	lecture      = "assets/lecture.wav"
	lectureSound = "RIFF the bytes of a lecture"
)

// heardModel returns, for every audio segment, the words configured for the moment the segment
// begins, and records each moment a recording was requested for segments from.
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
	texts.SetStore(PluginID, s.store)
	return s
}

func (s *scene) transcribe(t *testing.T) {
	t.Helper()
	if err := s.transcriber.Transcribe(t.Context(), s.vault.Vault().ID, lecture); err != nil {
		t.Fatal(err)
	}
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

func TestTranscribeWritesATranscriptVersionInTheCommonFormat(t *testing.T) {
	s := newScene(t)
	s.transcribe(t)

	version, text := s.readLive(t)
	fileHash := domain.HashContent([]byte(lectureSound))
	if version.Kind != textKind || version.Key != fileHash || version.SourceHash != fileHash || version.File != fileHash+".jsonl" || !version.IsFromFile() {
		t.Errorf("the version is %+v, want a transcript from the file in %s.jsonl", version, fileHash)
	}
	if text != "first words\nsecond words\n" {
		t.Errorf("the version reads %q", text)
	}
	raw, err := s.store.Read(t.Context(), fileHash+".jsonl")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"text":"first words\n","selector":{"t":[0,1]}}` + "\n" + `{"text":"second words\n","selector":{"t":[1,2.5]}}` + "\n"
	if string(raw) != want {
		t.Errorf("the file holds\n%s\nwant\n%s", raw, want)
	}
	listed, err := s.store.List(t.Context())
	if err != nil || len(listed) != 1 {
		t.Errorf("the folder holds %v (%v), want the whole transcript alone", listed, err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are still shown", tasks)
	}
}

func TestTranscribeOfASilentRecordingRecordsAnEmptyTranscript(t *testing.T) {
	s := newScene(t)
	s.model.words = map[int]string{}
	s.transcribe(t)

	if _, text := s.readLive(t); text != "" {
		t.Errorf("a silent recording reads %q", text)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are still shown", tasks)
	}
}

func TestTranscribeSkipsWholeTranscript(t *testing.T) {
	s := newScene(t)
	s.transcribe(t)
	s.transcribe(t)

	if s.model.opened != 1 {
		t.Errorf("the recording was opened %d times, want once", s.model.opened)
	}
}

func TestTranscribeResumesFromPartialTranscript(t *testing.T) {
	s := newScene(t)
	key := domain.HashContent([]byte(lectureSound))
	if err := (transcripts{store: s.store}).appendBatch(t.Context(), key, []transcript.Cue{{Text: "heard before", From: 0, To: 1000}}, 1000); err != nil {
		t.Fatal(err)
	}
	if err := s.store.Append(t.Context(), getPartialName(key), []byte(`{"text":"a batch that did not la`)); err != nil {
		t.Fatal(err)
	}

	s.transcribe(t)

	if starts := s.model.getSegmentStarts(); len(starts) == 0 || starts[0] != 1000 {
		t.Errorf("the recording was asked for segments from %v, want from 1000 first", starts)
	}
	if _, text := s.readLive(t); text != "heard before\nsecond words\n" {
		t.Errorf("the version reads %q", text)
	}
}

func TestTranscribeReportsUnopenableRecording(t *testing.T) {
	s := newScene(t)
	s.model.isBroken = true

	if err := s.transcriber.Transcribe(t.Context(), s.vault.Vault().ID, lecture); err == nil {
		t.Fatal("a recording that cannot be opened was transcribed")
	}

	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].About != lecture || !strings.Contains(tasks[0].Error, "cannot be opened") {
		t.Errorf("the tasks are %+v, want one saying the recording cannot be opened", tasks)
	}
	listing, err := s.texts.ListVersions(t.Context(), lecture)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Errorf("a recording that could not be opened has versions %+v", listing.Versions)
	}
}

// waitUntil returns once the condition holds, and fails the test where it does not within five
// seconds.
func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for !condition() {
		if ctx.Err() != nil {
			t.Fatal("the condition did not hold within five seconds")
		}
		runtime.Gosched()
	}
}

func TestTranscribeWaitsForQueueTurn(t *testing.T) {
	s := newScene(t)
	held, err := s.queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- s.transcriber.Transcribe(t.Context(), s.vault.Vault().ID, lecture) }()
	waitUntil(t, func() bool { return len(s.queue.ListTasks()) == 1 })
	if tasks := s.queue.ListTasks(); tasks[0].About != lecture {
		t.Fatalf("the tasks are %+v once the transcription was asked for, want it shown", tasks)
	}
	held.Release()
	if err := <-result; err != nil {
		t.Fatal(err)
	}

	if _, text := s.readLive(t); text != "first words\nsecond words\n" {
		t.Errorf("the version reads %q", text)
	}
}

func TestTranscribeCommandWaitsForTranscript(t *testing.T) {
	s := newScene(t)
	run := newTranscribeCommand(s.transcriber)

	res, err := run(t.Context(), command.Request{Command: "transcribe", Path: lecture})
	if err != nil || res.Message != "transcribed "+lecture {
		t.Fatalf("answered %+v (%v)", res, err)
	}
	if _, text := s.readLive(t); text != "first words\nsecond words\n" {
		t.Errorf("the version reads %q once the command answered", text)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are still shown", tasks)
	}
}

func TestTranscribeCommandReportsUnopenableRecording(t *testing.T) {
	s := newScene(t)
	s.model.isBroken = true

	_, err := newTranscribeCommand(s.transcriber)(t.Context(), command.Request{Command: "transcribe", Path: lecture})

	if err == nil || !strings.Contains(err.Error(), "cannot be opened") {
		t.Errorf("a recording that cannot be opened was answered %v", err)
	}
}

func TestTranscribeRecordsTheVersionInTheVaultTheRequestNames(t *testing.T) {
	s := newScene(t)
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	chemistry.File(lecture, lectureSound)
	other, isTexts := s.texts.With("Chemistry").(*hosttest.Texts)
	if !isTexts {
		t.Fatal("the texts of Chemistry are not hosttest texts")
	}
	other.SetStore(PluginID, chemistry.Store())
	other.File(lecture, []byte(lectureSound))

	if err := s.transcriber.Transcribe(t.Context(), "Chemistry", lecture); err != nil {
		t.Fatal(err)
	}

	inChemistry, err := other.ListVersions(t.Context(), lecture)
	if err != nil || inChemistry.Live == "" {
		t.Fatalf("Chemistry lists %+v (%v), want the transcript live", inChemistry, err)
	}
	inShown, err := s.texts.ListVersions(t.Context(), lecture)
	if err != nil {
		t.Fatal(err)
	}
	if len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
}

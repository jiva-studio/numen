package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

const (
	ocrID    = "md.numen.engines.ocr"
	layerID  = "md.numen.readers.pdf"
	speechID = "md.numen.engines.transcription"
	scanPath = "books/flywheel.pdf"
	talkPath = "talks/storage.wav"
)

// scene is a vault with a scan and a recording, the versions other plugins made of them, and the
// proofreading over them.
type scene struct {
	texts    *hosttest.Texts
	queue    *hosttest.Queue
	store    host.Store
	others   *hosttest.MemoryProvider
	model    *modelHost
	settings *settings
	run      *Proofreading
	provider *provider
}

// modelHost is a completer that answers every prompt through its script and keeps what it was asked.
type modelHost struct {
	script    func(prompt string) string
	failure   error
	questions []*numenv1.CompletionQuestion
}

func (m *modelHost) complete(_ context.Context, question *numenv1.CompletionQuestion) (string, error) {
	m.questions = append(m.questions, question)
	if m.failure != nil {
		return "", m.failure
	}
	return m.script(question.GetMessages()[1].GetText()), nil
}

func newScene(t *testing.T) *scene {
	t.Helper()
	vault := hosttest.New("Physics")
	s := &scene{
		texts:    hosttest.NewTexts("Physics", PluginID),
		queue:    hosttest.NewQueue(),
		store:    vault.Store(),
		others:   hosttest.NewMemoryProvider(),
		model:    &modelHost{script: func(string) string { return "" }},
		settings: newSettings(),
	}
	s.texts.File(scanPath, []byte("the scan"))
	s.texts.File(talkPath, []byte("the recording"))
	for _, id := range []string{ocrID, layerID, speechID} {
		s.texts.Provide(id, s.others)
	}
	s.run = newProofreading(Host{Store: s.store, Texts: s.texts, Queue: s.queue}, newAskModel(s.model.complete), s.settings.getConfig)
	s.provider = newProvider(s.run, s.store)
	t.Cleanup(s.provider.Close)
	s.texts.Provide(PluginID, s.provider)
	return s
}

// hold puts a version another plugin made from the file.
func (s *scene) hold(t *testing.T, pluginID, path, key string, format domain.TextFormat, raw string, parts []domain.TextPart) domain.TextVersion {
	t.Helper()
	s.others.Hold(path, key, parts)
	version, err := s.texts.As(pluginID).PutVersion(t.Context(), host.TextPut{
		Path: path, Key: key, Format: format, InputHash: domain.HashContent([]byte(raw)), Hash: hashParts(parts),
	})
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func (s *scene) holdScan(t *testing.T) domain.TextVersion {
	t.Helper()
	rect := domain.Rect{MinX: 0.1, MinY: 0.1, MaxX: 0.9, MaxY: 0.2}
	return s.hold(t, ocrID, scanPath, "recognition", domain.TextFormatPositioned, "the scan", []domain.TextPart{
		{Text: "A flywheel keeps energy ", Section: "Storage", Place: domain.TextPlace{Page: 0, Rects: []domain.Rect{rect}}},
		{Text: "in a rotor spinnig in a vacum.\n\n", Place: domain.TextPlace{Page: 0, Rects: []domain.Rect{rect}}},
		{Text: "Pumped storage lifts water.\n\n", Place: domain.TextPlace{Page: 1, Rects: []domain.Rect{rect}}},
	})
}

func (s *scene) holdTalk(t *testing.T, lines ...string) domain.TextVersion {
	t.Helper()
	parts := make([]domain.TextPart, len(lines))
	for i, line := range lines {
		parts[i] = domain.TextPart{Text: line + "\n", Place: domain.TextPlace{
			Start: time.Duration(i) * time.Second, End: time.Duration(i+1) * time.Second,
		}}
	}
	return s.hold(t, speechID, talkPath, "heard", domain.TextFormatTimed, "the recording", parts)
}

// readLive is the parts of the file's live version and its record.
func (s *scene) readLive(t *testing.T, path string) ([]domain.TextPart, domain.TextVersion) {
	t.Helper()
	listing, err := s.texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			parts, err := s.run.readParts(t.Context(), version)
			if err != nil {
				t.Fatal(err)
			}
			return parts, version
		}
	}
	t.Fatalf("%s has no live version: %+v", path, listing)
	return nil, domain.TextVersion{}
}

// getRows answers every numbered line of a prompt whose words the replace function changes.
func getRows(prompt string, replace func(number, text string) string) string {
	var rows []string
	for _, segment := range strings.Split(prompt, Opens)[1:] {
		number, text, _ := strings.Cut(segment, Closes)
		text = strings.TrimSpace(text)
		if fixed := replace(number, text); fixed != text {
			rows = append(rows, number+"|"+fixed)
		}
	}
	return strings.Join(rows, "\n")
}

func TestProofreadMakesTheLiveVersionFromTheScan(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.model.script = func(prompt string) string {
		return getRows(prompt, func(_, text string) string {
			return strings.NewReplacer("spinnig", "spinning", "vacum", "vacuum").Replace(text)
		})
	}

	res, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Input: scan.GetID(), IsUserStarted: true})
	if err != nil {
		t.Fatal(err)
	}

	parts, live := s.readLive(t, scanPath)
	if live.PluginID != PluginID || live.Input != scan.GetID() || live.InputHash != scan.Hash || live.Format != domain.TextFormatPositioned {
		t.Fatalf("the live version is %+v", live)
	}
	if res.Lines != 3 || res.Corrected != 1 {
		t.Errorf("result %+v", res)
	}
	if len(parts) != 3 || parts[1].Text != "in a rotor spinning in a vacuum.\n\n" {
		t.Fatalf("parts %+v", parts)
	}
	if parts[0].Section != "Storage" || parts[2].Place.Page != 1 || len(parts[1].Place.Rects) != 1 {
		t.Errorf("the places were not kept: %+v", parts)
	}
	if len(s.model.questions) != 2 {
		t.Errorf("asked %d times, want once a page", len(s.model.questions))
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks left %+v", tasks)
	}
}

func TestProofreadJoinsARunOfCues(t *testing.T) {
	s := newScene(t)
	talk := s.holdTalk(t, "pumped storage lifts", "water to an upper reservoir", "and lets it fall")
	s.model.script = func(prompt string) string {
		if strings.Contains(prompt, Opens+"1"+Closes) {
			return "1-2|Pumped storage lifts water to an upper reservoir."
		}
		return ""
	}

	if _, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Input: talk.GetID()}); err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLive(t, talkPath)
	if len(parts) != 2 {
		t.Fatalf("parts %+v", parts)
	}
	if parts[0].Text != "Pumped storage lifts water to an upper reservoir.\n" ||
		parts[0].Place.Start != 0 || parts[0].Place.End != 2*time.Second {
		t.Errorf("the run is %+v", parts[0])
	}
	if parts[1].Text != "and lets it fall\n" || parts[1].Place.Start != 2*time.Second {
		t.Errorf("the cue after the run is %+v", parts[1])
	}
}

func TestProofreadAsksAboutALongTranscriptAWindowAtATime(t *testing.T) {
	s := newScene(t)
	s.settings.updateConfig(map[string]string{"batch_size": "2", "overlap": "1", "in_flight": "1"})
	talk := s.holdTalk(t, "one", "two", "three", "four", "five")
	s.model.script = func(prompt string) string {
		return getRows(prompt, func(_, text string) string { return text + "!" })
	}

	res, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Input: talk.GetID()})
	if err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLive(t, talkPath)
	var got []string
	for _, part := range parts {
		got = append(got, strings.TrimSpace(part.Text))
	}
	if strings.Join(got, " ") != "one! two! three! four! five!" || res.Corrected != 5 {
		t.Errorf("the transcript reads %q after %+v", got, res)
	}
}

func TestProofreadWithoutAVersionUsesTheLiveOne(t *testing.T) {
	s := newScene(t)
	s.holdScan(t)
	s.model.script = func(prompt string) string {
		return getRows(prompt, func(_, text string) string { return strings.ReplaceAll(text, "vacum", "vacuum") })
	}

	if _, err := newProofreadCommand(s.run)(t.Context(), command.Request{Command: "proofread", Path: scanPath}); err != nil {
		t.Fatal(err)
	}

	if _, live := s.readLive(t, scanPath); live.PluginID != PluginID {
		t.Errorf("the live version is %+v", live)
	}
}

func TestTheVersionMadeFromARecognitionDiesWithTheOCRPlugin(t *testing.T) {
	s := newScene(t)
	s.hold(t, layerID, scanPath, "layer", domain.TextFormatPositioned, "the scan", []domain.TextPart{{Text: "shipping manifest"}})
	scan := s.holdScan(t)
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Input: scan.GetID()}); err != nil {
		t.Fatal(err)
	}
	if _, live := s.readLive(t, scanPath); live.PluginID != PluginID {
		t.Fatalf("the live version is %+v", live)
	}

	s.texts.SetEnabled(ocrID, false)

	if parts, live := s.readLive(t, scanPath); live.PluginID != layerID || parts[0].Text != "shipping manifest" {
		t.Errorf("with the OCR plugin off the live version is %+v", live)
	}
	s.texts.SetEnabled(ocrID, true)
	if _, live := s.readLive(t, scanPath); live.PluginID != PluginID {
		t.Errorf("with the OCR plugin on again the live version is %+v", live)
	}
}

func TestProofreadRefusesWhatItCannotRead(t *testing.T) {
	s := newScene(t)
	article := s.hold(t, layerID, "notes/article.url", "article", domain.TextFormatPlain, "", []domain.TextPart{{Text: "words"}})

	for name, req := range map[string]struct {
		request Request
		want    error
	}{
		"a file with no text": {Request{Path: talkPath}, errNoText},
		"plain text":          {Request{Path: "notes/article.url", Input: article.GetID()}, errFormat},
		"a version not there": {Request{Path: scanPath, Input: "md.numen.engines.ocr/gone"}, host.ErrNotExist},
	} {
		if _, err := s.run.Proofread(t.Context(), req.request); !errors.Is(err, req.want) {
			t.Errorf("%s: got %v, want %v", name, err, req.want)
		}
	}
}

func TestAFailureOfTheModelIsShownAsATask(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.model.failure = errors.New("the model is away")

	_, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Input: scan.GetID()})

	if !errors.Is(err, s.model.failure) {
		t.Fatalf("got %v", err)
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || !strings.Contains(tasks[0].Error, "the model is away") || tasks[0].About != scanPath {
		t.Errorf("tasks %+v", tasks)
	}
	if _, live := s.readLive(t, scanPath); live.PluginID != ocrID {
		t.Errorf("a failed proofreading left the live version %+v", live)
	}
}

func TestCreateTextProofreadsInTheBackground(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)

	if err := s.provider.CreateText(t.Context(), host.TextCreation{Vault: "Physics", Path: scanPath, Input: scan.GetID(), IsUserStarted: true}); err != nil {
		t.Fatal(err)
	}
	s.provider.Wait()

	if _, live := s.readLive(t, scanPath); live.PluginID != PluginID {
		t.Errorf("the live version is %+v", live)
	}
}

func TestCreateTextOfTheFileIsInvalid(t *testing.T) {
	s := newScene(t)
	err := s.provider.CreateText(t.Context(), host.TextCreation{Vault: "Physics", Path: scanPath})
	if !errors.Is(err, host.ErrInvalidVersion) {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteTextRemovesTheTextFile(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	res, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Input: scan.GetID()})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.provider.DeleteText(t.Context(), "Physics", scanPath, res.Version.Key); err != nil {
		t.Fatal(err)
	}

	_, err = s.provider.GetText(t.Context(), host.TextRequest{Vault: "Physics", Path: scanPath, Key: res.Version.Key})
	if !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a deleted text answered %v", err)
	}
}

func TestGetTextOfAnotherKeyIsNotThere(t *testing.T) {
	s := newScene(t)
	for name, req := range map[string]host.TextRequest{
		"a key of no proofreading": {Path: scanPath, Key: "recognition"},
		"a cursor":                 {Path: scanPath, Key: getVersionKey("x"), Cursor: "1"},
		"nothing kept":             {Path: scanPath, Key: getVersionKey("x")},
	} {
		if _, err := s.provider.GetText(t.Context(), req); !errors.Is(err, host.ErrNotExist) {
			t.Errorf("%s answered %v", name, err)
		}
	}
}

func TestTheModelIsAskedByPurposeAlone(t *testing.T) {
	m := &modelHost{script: func(string) string { return "" }}

	if _, err := newAskModel(m.complete)(t.Context(), "be careful", "the prompt"); err != nil {
		t.Fatal(err)
	}

	q := m.questions[0]
	if q.GetPurpose() != "proofreading" || len(q.GetMessages()) != 2 ||
		q.GetMessages()[0].GetRole() != "system" || q.GetMessages()[0].GetText() != "be careful" ||
		q.GetMessages()[1].GetRole() != "user" || q.GetMessages()[1].GetText() != "the prompt" {
		t.Errorf("question %v", q)
	}
}

func TestTheMockMarksTheLinesTheGatesLetThrough(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.run.ask = newAskModel(completeMock)

	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Input: scan.GetID()}); err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLive(t, scanPath)
	if parts[0].Text != "A flywheel keeps energy " {
		t.Errorf("the gate let a mark on a short line through: %q", parts[0].Text)
	}
	for _, part := range parts[1:] {
		if !strings.HasSuffix(strings.TrimSpace(part.Text), " "+MockMark) {
			t.Errorf("the mock left %q", part.Text)
		}
	}
}

func TestTheMockAnswersTheSameQuestionTheSameWay(t *testing.T) {
	question := &numenv1.CompletionQuestion{Messages: []*numenv1.CompletionMessage{
		{Role: "system", Text: speechInstruction},
		{Role: "user", Text: "What it holds\n\n" + buildPrompt(Batch{Lines: []Line{{Number: 4, Text: "one"}, {Number: 5, Text: "two"}}})},
	}}

	first, err := completeMock(t.Context(), question)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := completeMock(t.Context(), question)

	if want := "4|one " + MockMark + "\n5|two " + MockMark; first != want || second != want {
		t.Errorf("the mock answered %q and %q, want %q", first, second, want)
	}
}

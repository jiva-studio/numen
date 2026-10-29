package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/ai"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	proofreading "github.com/jiva-studio/numen/plugins/tools-proofreading"
)

const (
	ocrID    = "md.test.alpha"
	layerID  = "md.test.beta"
	speechID = "md.test.gamma"
	scanPath = "books/flywheel.pdf"
	talkPath = "talks/storage.wav"

	kindTranscript  domain.TextKind = "transcript"
	kindRecognition domain.TextKind = "recognition"
	kindPageText    domain.TextKind = "page-text"
)

// scene is a vault with a scan and a recording, the versions other plugins made of them, and the
// proofreading over them.
type scene struct {
	vault    *hosttest.Vault
	texts    *hosttest.Texts
	queue    *hosttest.Queue
	store    host.Store
	model    *modelHost
	settings *settings
	run      *Proofreading
}

// modelHost is a completer that answers every prompt through its script and keeps what it was asked.
type modelHost struct {
	script    func(prompt string) string
	failure   error
	questions []*ai.Question
}

func (m *modelHost) complete(_ context.Context, question *ai.Question) (string, error) {
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
		vault:    vault,
		texts:    hosttest.NewTexts("Physics", PluginID),
		queue:    hosttest.NewQueue(),
		store:    vault.Store(),
		model:    &modelHost{script: func(string) string { return "" }},
		settings: newSettings(),
	}
	s.texts.File(scanPath, []byte("the scan"))
	s.texts.File(talkPath, []byte("the recording"))
	s.texts.SetStore(PluginID, s.store)
	kinds, err := readKinds(proofreading.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	s.run = newProofreading(Host{Store: s.store, Texts: s.texts, Queue: s.queue}, kinds, newAskModel(s.model.complete, s.settings.getConfig), s.settings.getConfig)
	return s
}

// hold puts a version another plugin made from the file.
func (s *scene) hold(t *testing.T, pluginID, path, key string, kind domain.TextKind, raw string, parts []domain.TextSegment) domain.TextVersion {
	t.Helper()
	if err := host.WriteText(t.Context(), s.texts.GetStore(pluginID), key+".jsonl", parts); err != nil {
		t.Fatal(err)
	}
	version, err := s.texts.As(pluginID).PutVersion(t.Context(), host.TextPut{
		Path: path, Key: key, Kind: kind, SourceHash: domain.HashContent([]byte(raw)), File: key + ".jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func (s *scene) holdScan(t *testing.T) domain.TextVersion {
	t.Helper()
	rect := [][4]float32{{0.1, 0.1, 0.9, 0.2}}
	first, second := host.Selector{Page: 1, Boxes: rect}.Encode(), host.Selector{Page: 2, Boxes: rect}.Encode()
	return s.hold(t, ocrID, scanPath, "recognition", kindRecognition, "the scan", []domain.TextSegment{
		{Text: "A flywheel keeps energy ", Heading: "Storage", Selector: first},
		{Text: "in a rotor spinnig in a vacum.\n\n", Selector: first},
		{Text: "Pumped storage lifts water.\n\n", Selector: second},
	})
}

func (s *scene) holdTalk(t *testing.T, lines ...string) domain.TextVersion {
	t.Helper()
	parts := make([]domain.TextSegment, len(lines))
	for i, line := range lines {
		parts[i] = domain.TextSegment{Text: line + "\n", Selector: host.Selector{T: []float64{float64(i), float64(i + 1)}}.Encode()}
	}
	return s.hold(t, speechID, talkPath, "heard", kindTranscript, "the recording", parts)
}

// readLive is the parts of the file's live version and its record.
func (s *scene) readLive(t *testing.T, path string) ([]domain.TextSegment, domain.TextVersion) {
	t.Helper()
	listing, err := s.texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			parts, err := readParts(t.Context(), s.texts, version)
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

	res, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID(), IsUserStarted: true})
	if err != nil {
		t.Fatal(err)
	}

	parts, live := s.readLive(t, scanPath)
	if live.PluginID != PluginID || live.Source != scan.GetID() || live.SourceHash != scan.Hash || live.Kind != kindRecognition {
		t.Fatalf("the live version is %+v", live)
	}
	if res.Lines != 3 || res.Corrected != 1 {
		t.Errorf("result %+v", res)
	}
	if len(parts) != 3 || parts[1].Text != "in a rotor spinning in a vacuum.\n\n" {
		t.Fatalf("parts %+v", parts)
	}
	if parts[0].Heading != "Storage" || parts[2].Selector != `{"page":2,"boxes":[[0.1,0.1,0.9,0.2]]}` || parts[1].Selector != `{"page":1,"boxes":[[0.1,0.1,0.9,0.2]]}` {
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

	if _, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: talk.GetID()}); err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLive(t, talkPath)
	if len(parts) != 2 {
		t.Fatalf("parts %+v", parts)
	}
	if parts[0].Text != "Pumped storage lifts water to an upper reservoir.\n" ||
		parts[0].Selector != `{"t":[0,2]}` {
		t.Errorf("the run is %+v", parts[0])
	}
	if parts[1].Text != "and lets it fall\n" || parts[1].Selector != `{"t":[2,3]}` {
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

	res, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: talk.GetID()})
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

func TestTheVersionMadeFromARecognitionStaysLiveWithTheOCRPluginOff(t *testing.T) {
	s := newScene(t)
	s.hold(t, layerID, scanPath, "layer", kindPageText, "the scan", []domain.TextSegment{{Text: "shipping manifest"}})
	scan := s.holdScan(t)
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); err != nil {
		t.Fatal(err)
	}
	if _, live := s.readLive(t, scanPath); live.PluginID != PluginID {
		t.Fatalf("the live version is %+v", live)
	}
}

func TestProofreadRefusesWhatItCannotRead(t *testing.T) {
	s := newScene(t)
	s.run.kinds = map[domain.TextKind]bool{kindRecognition: true}
	article := s.hold(t, layerID, "notes/article.url", "article", "article", "", []domain.TextSegment{{Text: "words"}})

	for name, req := range map[string]struct {
		request Request
		want    error
	}{
		"a file with no text": {Request{Path: talkPath}, errNoText},
		"an article":          {Request{Path: "notes/article.url", Source: article.GetID()}, errFormat},
		"a version not there": {Request{Path: scanPath, Source: ocrID + "/gone"}, host.ErrNotExist},
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

	_, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()})

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

func TestProofreadMakesAVersionOfTheSourceThePluginIsGiven(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)

	if _, err := s.run.Proofread(t.Context(), Request{Vault: "Physics", Path: scanPath, Source: scan.GetID(), IsUserStarted: true}); err != nil {
		t.Fatal(err)
	}

	if _, live := s.readLive(t, scanPath); live.PluginID != PluginID {
		t.Errorf("the live version is %+v", live)
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

	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); err != nil {
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
	question := &ai.Question{Messages: []ai.Message{
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

func TestProofreadRecordsTheVersionInTheVaultTheRequestNames(t *testing.T) {
	s := newScene(t)
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	other := s.texts.With("Chemistry").(*hosttest.Texts)
	other.File(scanPath, []byte("the scan"))
	other.SetStore(PluginID, chemistry.Store())
	if err := host.WriteText(t.Context(), other.GetStore(ocrID), "recognition.jsonl", []domain.TextSegment{{Text: "A rotor spinnig.\n\n"}}); err != nil {
		t.Fatal(err)
	}
	scan, err := other.As(ocrID).PutVersion(t.Context(), host.TextPut{
		Path: scanPath, Key: "recognition", Kind: kindRecognition, SourceHash: domain.HashContent([]byte("the scan")), File: "recognition.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.run.Proofread(t.Context(), Request{Vault: "Chemistry", Path: scanPath, Source: scan.GetID(), IsUserStarted: true}); err != nil {
		t.Fatal(err)
	}

	inChemistry, err := other.ListVersions(t.Context(), scanPath)
	if err != nil || len(inChemistry.Versions) != 2 || !strings.HasPrefix(string(inChemistry.Live), PluginID+"/") {
		t.Fatalf("Chemistry lists %+v (%v), want the proofreading live beside the scan", inChemistry, err)
	}
	if inShown, _ := s.texts.ListVersions(t.Context(), scanPath); len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
	if names, _ := s.store.List(t.Context()); len(names) != 0 {
		t.Errorf("the shown vault's folder holds %v, want nothing", names)
	}
}

func TestProofreadUsesDirectChatAPIWhenProviderIsConfigured(t *testing.T) {
	var gotAuth, gotModel, gotSystem, gotPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotModel = req.Model
		if len(req.Messages) == 2 {
			gotSystem = req.Messages[0].Content
			gotPrompt = req.Messages[1].Content
		}
		reply := getRows(gotPrompt, func(_, text string) string {
			return strings.NewReplacer("spinnig", "spinning", "vacum", "vacuum").Replace(text)
		})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": reply}},
			},
		})
	}))
	defer srv.Close()

	s := newScene(t)
	s.settings.updateConfig(map[string]string{
		"provider": "custom",
		"baseUrl":  srv.URL,
		"model":    "google/gemini-2.5-flash",
		"apiKey":   "secret-key",
	})
	scan := s.holdScan(t)

	res, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID(), IsUserStarted: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Corrected != 1 {
		t.Errorf("corrected = %d, want 1", res.Corrected)
	}
	if len(s.model.questions) != 0 {
		t.Errorf("agent completer was called %d times, want 0", len(s.model.questions))
	}
	if gotAuth != "Bearer secret-key" {
		t.Errorf("Authorization = %q, want Bearer secret-key", gotAuth)
	}
	if gotModel != "google/gemini-2.5-flash" {
		t.Errorf("model = %q, want google/gemini-2.5-flash", gotModel)
	}
	if gotSystem != scanInstruction {
		t.Errorf("system instruction mismatch")
	}
}

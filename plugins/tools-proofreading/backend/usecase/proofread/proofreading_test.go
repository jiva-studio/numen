package proofread

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/settings"
)

const (
	pluginID = "md.numen.tools.proofreading"
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
	vault  *hosttest.Vault
	texts  *hosttest.Texts
	queue  *hosttest.Queue
	store  *hosttest.Store
	model  *model
	config settings.Config
	run    *Proofreader
}

func newScene(t *testing.T) *scene {
	t.Helper()
	vault := hosttest.New("Physics")
	s := &scene{
		vault:  vault,
		texts:  hosttest.NewTexts("Physics", pluginID),
		queue:  hosttest.NewQueue(),
		store:  vault.Store(),
		model:  newReplyModel(""),
		config: settings.DefaultConfig(),
	}
	s.texts.WriteFile(scanPath, []byte("the scan"))
	s.texts.WriteFile(talkPath, []byte("the recording"))
	s.texts.SetStore(pluginID, s.store)
	s.run = New(Host{
		Store:    func(id domain.VaultID) host.StoreWriter { return s.store.With(id) },
		GetTexts: func(id domain.VaultID) Texts { return s.texts.With(id) },
		Queue:    s.queue,
	}, GetProofreadKinds(), s.model.ask, func() settings.Config { return s.config })
	return s
}

func (s *scene) hold(t *testing.T, owner, path, key string, kind domain.TextKind, raw string, parts []domain.TextSegment) domain.TextVersion {
	t.Helper()
	if err := host.WriteText(t.Context(), s.texts.GetStore(owner), key+".jsonl", parts); err != nil {
		t.Fatal(err)
	}
	version, err := s.texts.UsePlugin(owner).PutVersion(t.Context(), host.TextPut{
		Path: path, Key: key, Kind: kind, InputHash: domain.HashContent([]byte(raw)), File: key + ".jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func (s *scene) holdScan(t *testing.T) domain.TextVersion {
	t.Helper()
	rect := [][4]float32{{0.1, 0.1, 0.9, 0.2}}
	first, second := encodeSelector(t, numbered.Selector{Page: 1, Boxes: rect}), encodeSelector(t, numbered.Selector{Page: 2, Boxes: rect})
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
		parts[i] = domain.TextSegment{Text: line + "\n", Selector: encodeSelector(t, numbered.Selector{T: []float64{float64(i), float64(i + 1)}})}
	}
	return s.hold(t, speechID, talkPath, "heard", kindTranscript, "the recording", parts)
}

func (s *scene) readLiveSegments(t *testing.T, path string) ([]domain.TextSegment, domain.TextVersion) {
	t.Helper()
	listing, err := s.texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			parts, err := ReadParts(t.Context(), s.texts, version)
			if err != nil {
				t.Fatal(err)
			}
			return parts, version
		}
	}
	t.Fatalf("%s has no live version: %+v", path, listing)
	return nil, domain.TextVersion{}
}

func getRows(prompt string, replace func(number, text string) string) string {
	var rows []string
	for _, segment := range strings.Split(prompt, numbered.Opens)[1:] {
		number, text, _ := strings.Cut(segment, numbered.Closes)
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

	res, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()})
	if err != nil {
		t.Fatal(err)
	}

	parts, live := s.readLiveSegments(t, scanPath)
	if live.PluginID != pluginID || live.Source != scan.GetID() || live.InputHash != scan.TextHash || live.Kind != ProofreadingKind || len(live.Key) != len(keyPrefix)+16 {
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
	if len(s.model.prompts) != 2 {
		t.Errorf("asked %d times, want once a page", len(s.model.prompts))
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks left %+v", tasks)
	}
}

func TestProofreadJoinsARunOfCues(t *testing.T) {
	s := newScene(t)
	talk := s.holdTalk(t, "pumped storage lifts", "water to an upper reservoir", "and lets it fall")
	s.model.script = func(prompt string) string {
		if strings.Contains(prompt, numbered.Opens+"1"+numbered.Closes) {
			return "1-2|Pumped storage lifts water to an upper reservoir."
		}
		return ""
	}

	if _, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: talk.GetID()}); err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLiveSegments(t, talkPath)
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

func TestProofread_PromptsLongTranscriptByWindow(t *testing.T) {
	s := newScene(t)
	s.config = settings.Config{BatchSize: 2, Overlap: 1, InFlight: 1, MaxEditDistance: s.config.MaxEditDistance}
	talk := s.holdTalk(t, "one", "two", "three", "four", "five")
	s.model.script = func(prompt string) string {
		return getRows(prompt, func(_, text string) string { return text + "!" })
	}

	res, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: talk.GetID()})
	if err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLiveSegments(t, talkPath)
	var got []string
	for _, part := range parts {
		got = append(got, strings.TrimSpace(part.Text))
	}
	if strings.Join(got, " ") != "one! two! three! four! five!" || res.Corrected != 5 {
		t.Errorf("the transcript reads %q after %+v", got, res)
	}
}

func TestProofread_OCRVersionStaysLiveWithoutOCRPlugin(t *testing.T) {
	s := newScene(t)
	s.run.kinds = map[domain.TextKind]bool{kindRecognition: true}
	s.hold(t, layerID, scanPath, "layer", kindPageText, "the scan", []domain.TextSegment{{Text: "shipping manifest"}})
	scan := s.holdScan(t)
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); err != nil {
		t.Fatal(err)
	}
	if _, live := s.readLiveSegments(t, scanPath); live.PluginID != pluginID {
		t.Fatalf("the live version is %+v", live)
	}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath}); !errors.Is(err, errFormat) {
		t.Errorf("live proofreading version outside kinds got %v, want %v", err, errFormat)
	}
}

func TestProofread_ErrorsOnUnreadableInput(t *testing.T) {
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
		if _, err := s.run.Proofread(t.Context(), req.request); !errors.Is(err, req.want) || strings.HasPrefix(err.Error(), "read ") {
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
	if len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "the model is away") || tasks[0].Detail != scanPath {
		t.Errorf("tasks %+v", tasks)
	}
	if _, live := s.readLiveSegments(t, scanPath); live.PluginID != ocrID {
		t.Errorf("a failed proofreading left the live version %+v", live)
	}
}

func TestProofreadMakesAVersionOfTheSourceThePluginIsGiven(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)

	if _, err := s.run.Proofread(t.Context(), Request{Vault: "Physics", Path: scanPath, Source: scan.GetID()}); err != nil {
		t.Fatal(err)
	}

	if _, live := s.readLiveSegments(t, scanPath); live.PluginID != pluginID {
		t.Errorf("the live version is %+v", live)
	}
}

func TestProofreadRecordsTheVersionInTheVaultTheRequestNames(t *testing.T) {
	s := newScene(t)
	chemistry := hosttest.New("Chemistry")
	s.vault.LinkBeside(chemistry)
	other := s.texts.With("Chemistry")
	other.WriteFile(scanPath, []byte("the scan"))
	other.SetStore(pluginID, chemistry.Store())
	if err := host.WriteText(t.Context(), other.GetStore(ocrID), "recognition.jsonl", []domain.TextSegment{{Text: "A rotor spinnig.\n\n"}}); err != nil {
		t.Fatal(err)
	}
	scan, err := other.UsePlugin(ocrID).PutVersion(t.Context(), host.TextPut{
		Path: scanPath, Key: "recognition", Kind: kindRecognition, InputHash: domain.HashContent([]byte("the scan")), File: "recognition.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.run.Proofread(t.Context(), Request{Vault: "Chemistry", Path: scanPath, Source: scan.GetID()}); err != nil {
		t.Fatal(err)
	}

	inChemistry, err := other.ListVersions(t.Context(), scanPath)
	if err != nil || len(inChemistry.Versions) != 2 || !strings.HasPrefix(string(inChemistry.Live), pluginID+"/") {
		t.Fatalf("Chemistry lists %+v (%v), want the proofreading live beside the scan", inChemistry, err)
	}
	inShown, err := s.texts.ListVersions(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
	names, err := s.store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("the shown vault's folder holds %v, want nothing", names)
	}
}

type stubTurn struct {
	releaseErr error
}

func (s stubTurn) Release() error { return s.releaseErr }

func (s stubTurn) WatchWaiting() <-chan struct{} { return nil }

type stubQueue struct {
	enqueueErr error
	releaseErr error
	reportErr  error
	reportAt   int
	finishErr  error
	reported   []host.Task
}

func (q *stubQueue) EnqueueWork(context.Context, host.Work) (host.Turn, error) {
	if q.enqueueErr != nil {
		return nil, q.enqueueErr
	}
	return stubTurn{releaseErr: q.releaseErr}, nil
}

func (q *stubQueue) ReportTask(_ context.Context, task host.Task) error {
	q.reported = append(q.reported, task)
	if q.reportErr != nil && (q.reportAt == 0 || len(q.reported) == q.reportAt) {
		return q.reportErr
	}
	return nil
}

func (q *stubQueue) FinishTask(context.Context, string) error {
	return q.finishErr
}

type stubTexts struct {
	Texts
	listErr error
	readErr error
	putErr  error
	pages   map[string]host.TextPage
}

func (s *stubTexts) ListVersions(ctx context.Context, path string) (host.TextVersionList, error) {
	if s.listErr != nil {
		return host.TextVersionList{}, s.listErr
	}
	return s.Texts.ListVersions(ctx, path)
}

func (s *stubTexts) ReadText(ctx context.Context, path string, version domain.TextVersionID, cursor string) (host.TextPage, error) {
	if s.readErr != nil {
		return host.TextPage{}, s.readErr
	}
	if s.pages != nil {
		return s.pages[cursor], nil
	}
	return s.Texts.ReadText(ctx, path, version, cursor)
}

func (s *stubTexts) PutVersion(ctx context.Context, req host.TextPut) (domain.TextVersion, error) {
	if s.putErr != nil {
		return domain.TextVersion{}, s.putErr
	}
	return s.Texts.PutVersion(ctx, req)
}

type failingStore struct {
	host.StoreWriter
	writeErr error
}

func (f failingStore) Write(ctx context.Context, name string, data []byte) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	return f.StoreWriter.Write(ctx, name, data)
}

func TestProofreadQueueAndHostFailures(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	finishCancelBoom := errors.New("finish failed on cancel")
	qCancel := &stubQueue{finishErr: finishCancelBoom}
	s.run.host.Queue = qCancel
	if _, err := s.run.Proofread(ctx, Request{Path: scanPath, Source: scan.GetID()}); err == nil || errors.Is(err, finishCancelBoom) || len(qCancel.reported) != 0 {
		t.Errorf("canceled run got %v, reported %+v", err, qCancel.reported)
	}

	reportBoom := errors.New("report failed")
	qFailBoth := &stubQueue{reportErr: reportBoom}
	s.run.host.Queue = qFailBoth
	_, err := s.run.Proofread(t.Context(), Request{Path: talkPath})
	if !errors.Is(err, errNoText) || !errors.Is(err, reportBoom) {
		t.Errorf("joined failure got %v", err)
	}

	releaseBoom := errors.New("release failed")
	qRelease := &stubQueue{releaseErr: releaseBoom}
	s.run.host.Queue = qRelease
	res, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()})
	if !errors.Is(err, releaseBoom) || res.Lines != 3 {
		t.Errorf("release failure got res=%+v err=%v", res, err)
	}

	finishBoom := errors.New("finish failed")
	s.run.host.Queue = &stubQueue{finishErr: finishBoom}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, finishBoom) {
		t.Errorf("finish failure got %v", err)
	}

	enqueueBoom := errors.New("enqueue failed")
	s.run.host.Queue = &stubQueue{enqueueErr: enqueueBoom}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, enqueueBoom) {
		t.Errorf("enqueue failure got %v", err)
	}

	for _, call := range []int{1, 2} {
		s.run.host.Queue = &stubQueue{reportErr: reportBoom, reportAt: call}
		if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, reportBoom) {
			t.Errorf("reportAt %d got %v", call, err)
		}
	}

	talk := s.holdTalk(t, "one", "two")
	s.run.host.Queue = &stubQueue{reportErr: reportBoom, reportAt: 2}
	if _, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: talk.GetID()}); !errors.Is(err, reportBoom) {
		t.Errorf("speech reportAt 2 got %v", err)
	}

	s.run.host.Queue = s.queue
	listBoom := errors.New("list failed")
	s.run.host.GetTexts = func(id domain.VaultID) Texts {
		return &stubTexts{Texts: s.texts.With(id), listErr: listBoom}
	}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, listBoom) {
		t.Errorf("list failure got %v", err)
	}

	readBoom := errors.New("read failed")
	s.run.host.GetTexts = func(id domain.VaultID) Texts {
		return &stubTexts{Texts: s.texts.With(id), readErr: readBoom}
	}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, readBoom) {
		t.Errorf("read failure got %v", err)
	}

	putBoom := errors.New("put failed")
	s.run.host.GetTexts = func(id domain.VaultID) Texts {
		return &stubTexts{Texts: s.texts.With(id), putErr: putBoom}
	}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, putBoom) {
		t.Errorf("put failure got %v", err)
	}

	s.run.host.GetTexts = func(id domain.VaultID) Texts { return s.texts.With(id) }
	writeBoom := errors.New("write failed")
	s.run.host.Store = func(id domain.VaultID) host.StoreWriter {
		return failingStore{StoreWriter: s.store.With(id), writeErr: writeBoom}
	}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, writeBoom) {
		t.Errorf("write failure got %v", err)
	}
}

func TestReadPartsPaginationAndCorrectWindows(t *testing.T) {
	paged := &stubTexts{
		pages: map[string]host.TextPage{
			"":   {Segments: []domain.TextSegment{{Text: "a"}}, NextCursor: "p2"},
			"p2": {Segments: []domain.TextSegment{{Text: "b"}}, NextCursor: "p2"},
		},
	}
	parts, err := ReadParts(t.Context(), paged, domain.TextVersion{Path: scanPath})
	if err != nil || len(parts) != 2 || parts[0].Text != "a" || parts[1].Text != "b" {
		t.Errorf("ReadParts = %+v, %v", parts, err)
	}

	s := newScene(t)
	qProgress := &stubQueue{}
	s.run.host.Queue = qProgress
	scan := s.holdScan(t)
	s.model.script = func(string) string { return "999|refused" }
	scanRes, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()})
	if err != nil || scanRes.Uncorrected != 2 {
		t.Errorf("scan refused = %+v, %v", scanRes, err)
	}
	if len(qProgress.reported) != 3 ||
		qProgress.reported[0].TotalCount != 3 || qProgress.reported[0].CompletedCount != 0 ||
		qProgress.reported[1].CompletedCount != 2 ||
		qProgress.reported[2].CompletedCount != 3 {
		t.Errorf("scan task reports = %+v", qProgress.reported)
	}

	s.config = settings.Config{BatchSize: 2, Overlap: 1, InFlight: 2, MaxEditDistance: s.config.MaxEditDistance}
	talk := s.holdTalk(t, "one", "two", "three", "four", "five")
	qSpeech := &stubQueue{}
	s.run.host.Queue = qSpeech
	s.model.prompts = nil
	s.model.script = func(string) string { return "999|refused" }
	speechRes, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: talk.GetID()})
	if err != nil || speechRes.Uncorrected != 4 {
		t.Errorf("speech refused = %+v, %v", speechRes, err)
	}
	if len(qSpeech.reported) != 3 || qSpeech.reported[1].CompletedCount != 3 || qSpeech.reported[2].CompletedCount != 5 {
		t.Errorf("speech task reports = %+v", qSpeech.reported)
	}

	// A joined run reaching the end of the first window advances next past its last line.
	s.config = settings.Config{BatchSize: 2, Overlap: 1, InFlight: 1, MaxEditDistance: s.config.MaxEditDistance}
	s.model.prompts = nil
	s.model.script = func(prompt string) string {
		if strings.Contains(prompt, numbered.Opens+"1"+numbered.Closes) {
			return "1-2|joined one and two"
		}
		return getRows(prompt, func(_, text string) string { return text + "!" })
	}
	joinedRes, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: talk.GetID()})
	if err != nil || joinedRes.Corrected != 4 {
		t.Errorf("joined window res = %+v, %v", joinedRes, err)
	}

	// Direct calls to correctSpeech and correctScan verify start line 1 and zero uncorrected on error.
	s.model.script = func(prompt string) string {
		return getRows(prompt, func(_, text string) string { return text + "!" })
	}
	var task host.Task
	correctedLines, uncorr, err := s.run.correctSpeech(t.Context(), []numbered.Line{
		{Number: 0, Text: "zero"},
		{Number: 1, Text: "one"},
	}, &task)
	if err != nil || uncorr != 0 || !reflect.DeepEqual(correctedLines, []numbered.Line{{Number: 1, Last: 1, Text: "one!"}}) {
		t.Errorf("correctSpeech from 1 = %+v, %d, %v", correctedLines, uncorr, err)
	}

	s.model.failure = errors.New("ask boom")
	if _, uncorr, err := s.run.correctSpeech(t.Context(), []numbered.Line{{Number: 1, Text: "one"}}, &task); err == nil || uncorr != 0 {
		t.Errorf("correctSpeech error = %d, %v", uncorr, err)
	}
	if _, uncorr, err := s.run.correctScan(t.Context(), []numbered.Batch{{Number: 1, Lines: []numbered.Line{{Number: 1, Text: "one"}}}}, &task); err == nil || uncorr != 0 {
		t.Errorf("correctScan error = %d, %v", uncorr, err)
	}
}

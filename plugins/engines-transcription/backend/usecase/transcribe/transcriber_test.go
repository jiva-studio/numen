package transcribe

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/transcript"
)

const (
	lecture      = "assets/lecture.wav"
	lectureSound = "RIFF the bytes of a lecture"
)

// heardModel returns, for every audio segment, the words configured for the moment the segment
// begins, and records each moment a recording was requested for segments from.
type heardModel struct {
	mu            sync.Mutex
	segments      []Audio
	words         map[int]string
	opened        int
	segmentStarts []int
	batchCounts   []int
	isBroken      bool
	segmentErr    error
	transcribeErr error
}

var errNoDecoder = errors.New("no decoder reads this")

func (m *heardModel) Open(context.Context, []byte) (Recording, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isBroken {
		return nil, errNoDecoder
	}
	m.opened++
	return heardRecording{model: m}, nil
}

func (m *heardModel) Transcribe(_ context.Context, audio Audio) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.transcribeErr != nil {
		return "", m.transcribeErr
	}
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
	if len(r.model.segments) == 0 {
		return 0
	}
	return r.model.segments[len(r.model.segments)-1].To
}

func (r heardRecording) GetSegments(_ context.Context, from, count int) ([]Audio, error) {
	r.model.mu.Lock()
	defer r.model.mu.Unlock()
	if r.model.segmentErr != nil {
		return nil, r.model.segmentErr
	}
	r.model.segmentStarts = append(r.model.segmentStarts, from)
	r.model.batchCounts = append(r.model.batchCounts, count)
	var out []Audio
	for _, one := range r.model.segments {
		if one.From >= from && len(out) < count {
			out = append(out, one)
		}
	}
	return out, nil
}

func (heardRecording) Close() error { return nil }

func storesOf(store *hosttest.Store) func(domain.VaultID) Folder {
	return func(id domain.VaultID) Folder { return store.With(id) }
}

func foldersOf(folder Folder) func(domain.VaultID) Folder {
	return func(domain.VaultID) Folder { return folder }
}

func textsOf(texts *hosttest.Texts) func(domain.VaultID) host.TextVersions {
	return func(id domain.VaultID) host.TextVersions { return texts.With(id) }
}

type vaultReader interface {
	host.FileReader
	GetVault(ctx context.Context) (domain.Vault, error)
}

// vaultFiles reads files through the vaults of the test host.
type vaultFiles struct {
	vaults func(domain.VaultID) vaultReader
}

func (f vaultFiles) ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error) {
	raw, _, err := f.vaults(vault).Read(ctx, path)
	return raw, err
}

func (f vaultFiles) GetActiveVaultID(ctx context.Context) (domain.VaultID, error) {
	shown, err := f.vaults("").GetVault(ctx)
	if err != nil {
		return "", err
	}
	return shown.ID, nil
}

type scene struct {
	vault       *hosttest.Vault
	store       *hosttest.Store
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
			segments: []Audio{{From: 0, To: 1000}, {From: 1000, To: 2500}},
			words:    map[int]string{0: "first words", 1000: "second words"},
		},
	}
	s.transcriber = NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return vault.With(id) }}, storesOf(s.store), textsOf(texts), s.queue,
		func(context.Context) (Recognizer, error) { return s.model, nil }, noReport)
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
	if version.Kind != textKind || version.Key != fileHash || version.InputHash != fileHash || version.File != fileHash+".jsonl" || !version.IsFromFile() {
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

	err := s.transcriber.Transcribe(t.Context(), s.vault.Vault().ID, lecture)
	if !errors.Is(err, errNoDecoder) {
		t.Fatalf("a recording that cannot be opened returned %v, want it to wrap the decoder error", err)
	}

	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].Detail != lecture || !strings.Contains(tasks[0].FailureReason, "cannot be opened") {
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
	if tasks := s.queue.ListTasks(); tasks[0].Detail != lecture {
		t.Fatalf("the tasks are %+v once the transcription was asked for, want it shown", tasks)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}

	if _, text := s.readLive(t); text != "first words\nsecond words\n" {
		t.Errorf("the version reads %q", text)
	}
}

func TestTranscribeRecordsTheVersionInTheVaultTheRequestNames(t *testing.T) {
	s := newScene(t)
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	chemistry.File(lecture, lectureSound)
	other := s.texts.With("Chemistry")
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
	if len(s.model.batchCounts) == 0 || s.model.batchCounts[0] != 16 {
		t.Errorf("expected batchSize 16, got %v", s.model.batchCounts)
	}
}

type faultyStore struct {
	Folder
	readErr   func(name string) error
	writeErr  func(name string) error
	appendErr error
	deleteErr error
	renameErr error
}

func (f *faultyStore) Read(ctx context.Context, name string) ([]byte, error) {
	if f.readErr != nil {
		if err := f.readErr(name); err != nil {
			return nil, err
		}
	}
	return f.Folder.Read(ctx, name)
}

func (f *faultyStore) Write(ctx context.Context, name string, content []byte) error {
	if f.writeErr != nil {
		if err := f.writeErr(name); err != nil {
			return err
		}
	}
	return f.Folder.Write(ctx, name, content)
}

func (f *faultyStore) Append(ctx context.Context, name string, content []byte) error {
	if f.appendErr != nil {
		return f.appendErr
	}
	return f.Folder.Append(ctx, name, content)
}

func (f *faultyStore) Delete(ctx context.Context, name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return f.Folder.Delete(ctx, name)
}

func (f *faultyStore) Rename(ctx context.Context, from, to string) error {
	if f.renameErr != nil {
		return f.renameErr
	}
	return f.Folder.Rename(ctx, from, to)
}

type faultyTexts struct {
	host.TextVersions
	listRes host.TextListing
	listErr error
	putErr  error
	puts    int
}

func (f *faultyTexts) ListVersions(ctx context.Context, path string) (host.TextListing, error) {
	if f.listErr != nil {
		return host.TextListing{}, f.listErr
	}
	if len(f.listRes.Versions) > 0 {
		return f.listRes, nil
	}
	return f.TextVersions.ListVersions(ctx, path)
}

func (f *faultyTexts) PutVersion(ctx context.Context, req host.TextPut) (domain.TextVersion, error) {
	f.puts++
	if f.putErr != nil {
		return domain.TextVersion{}, f.putErr
	}
	return f.TextVersions.PutVersion(ctx, req)
}

type faultyTurn struct {
	waiting    chan struct{}
	releaseErr error
}

func (t *faultyTurn) Release() error                { return t.releaseErr }
func (t *faultyTurn) WatchWaiting() <-chan struct{} { return t.waiting }

type faultyQueue struct {
	host.Queue
	enqueueErr error
	reportErr  error
	turn       host.Turn
	tasks      []host.Task
}

func (q *faultyQueue) EnqueueWork(ctx context.Context, work host.Work) (host.Turn, error) {
	if q.enqueueErr != nil {
		return nil, q.enqueueErr
	}
	if q.turn != nil {
		return q.turn, nil
	}
	return q.Queue.EnqueueWork(ctx, work)
}

func (q *faultyQueue) ReportTask(ctx context.Context, task host.Task) error {
	q.tasks = append(q.tasks, task)
	if q.reportErr != nil {
		return q.reportErr
	}
	return q.Queue.ReportTask(ctx, task)
}

func TestStoreEdgeCasesAndErrorPropagation(t *testing.T) {
	ctx := t.Context()
	s := newScene(t)
	errDisk := errors.New("disk error")

	// hasWhole error propagation.
	fs := &faultyStore{Folder: s.store, readErr: func(string) error { return errDisk }}
	ok, err := (transcripts{store: fs}).hasWhole(ctx, "k")
	if ok || !errors.Is(err, errDisk) {
		t.Fatalf("expected (false, errDisk), got (%v, %v)", ok, err)
	}

	// readReached when reached does not exist returns (0, Delete error).
	ms, err := (transcripts{store: s.store}).readReached(ctx, "none")
	if ms != 0 || err != nil {
		t.Fatalf("expected (0, nil), got (%d, %v)", ms, err)
	}
	fsDel := &faultyStore{Folder: s.store, deleteErr: errDisk}
	if _, err := (transcripts{store: fsDel}).readReached(ctx, "none"); !errors.Is(err, errDisk) {
		t.Fatalf("expected errDisk from Delete, got %v", err)
	}

	// readReached when Read(reached) fails with non-ErrNotExist.
	if _, err := (transcripts{store: fs}).readReached(ctx, "k"); !errors.Is(err, errDisk) {
		t.Fatalf("expected errDisk on Read(reached), got %v", err)
	}

	// readReached when reached contains invalid JSON deletes both partial and reached.
	if err := s.store.Write(ctx, getReachedName("corrupt"), []byte("{bad")); err != nil {
		t.Fatal(err)
	}
	if err := s.store.Write(ctx, getPartialName("corrupt"), []byte("partial")); err != nil {
		t.Fatal(err)
	}
	ms, err = (transcripts{store: s.store}).readReached(ctx, "corrupt")
	if ms != 0 || err != nil {
		t.Fatalf("expected (0, nil) on corrupt reached JSON, got (%d, %v)", ms, err)
	}
	if _, err := s.store.Read(ctx, getPartialName("corrupt")); !errors.Is(err, host.ErrNotExist) {
		t.Fatalf("expected corrupt partial deleted, got %v", err)
	}
	if _, err := s.store.Read(ctx, getReachedName("corrupt")); !errors.Is(err, host.ErrNotExist) {
		t.Fatalf("expected corrupt reached deleted, got %v", err)
	}

	// readReached when reached exists but partial is missing deletes reached.
	if err := s.store.Write(ctx, getReachedName("nopart"), []byte(`{"ms":500,"size":10}`)); err != nil {
		t.Fatal(err)
	}
	ms, err = (transcripts{store: s.store}).readReached(ctx, "nopart")
	if ms != 0 || err != nil {
		t.Fatalf("expected (0, nil) when partial missing, got (%d, %v)", ms, err)
	}
	if _, err := s.store.Read(ctx, getReachedName("nopart")); !errors.Is(err, host.ErrNotExist) {
		t.Fatalf("expected orphaned reached deleted, got %v", err)
	}

	// readReached when Read(partial) fails with non-ErrNotExist.
	if err := s.store.Write(ctx, getReachedName("parterr"), []byte(`{"ms":500,"size":2}`)); err != nil {
		t.Fatal(err)
	}
	fsPartRead := &faultyStore{Folder: s.store, readErr: func(name string) error {
		if name == getPartialName("parterr") {
			return errDisk
		}
		return nil
	}}
	if _, err := (transcripts{store: fsPartRead}).readReached(ctx, "parterr"); !errors.Is(err, errDisk) {
		t.Fatalf("expected errDisk reading partial, got %v", err)
	}

	// readReached when truncating oversized partial fails on Write.
	if err := s.store.Write(ctx, getPartialName("parterr"), []byte("12345")); err != nil {
		t.Fatal(err)
	}
	fsPartWrite := &faultyStore{Folder: s.store, writeErr: func(string) error { return errDisk }}
	if _, err := (transcripts{store: fsPartWrite}).readReached(ctx, "parterr"); !errors.Is(err, errDisk) {
		t.Fatalf("expected errDisk truncating partial, got %v", err)
	}

	// appendBatch errors on Append and Read(partial).
	fsAppend := &faultyStore{Folder: s.store, appendErr: errDisk}
	if err := (transcripts{store: fsAppend}).appendBatch(ctx, "b", []transcript.Cue{{Text: "hi", From: 0, To: 100}}, 100); !errors.Is(err, errDisk) {
		t.Fatalf("expected errDisk from Append, got %v", err)
	}
	fsReadAfterAppend := &faultyStore{Folder: s.store, readErr: func(string) error { return errDisk }}
	if err := (transcripts{store: fsReadAfterAppend}).appendBatch(ctx, "b", []transcript.Cue{{Text: "hi", From: 0, To: 100}}, 100); !errors.Is(err, errDisk) {
		t.Fatalf("expected errDisk reading partial after append, got %v", err)
	}

	// finish error on Rename.
	fsRename := &faultyStore{Folder: s.store, renameErr: errDisk}
	if err := (transcripts{store: fsRename}).finish(ctx, "b"); !errors.Is(err, errDisk) {
		t.Fatalf("expected errDisk from Rename, got %v", err)
	}
}

func TestTranscriberErrorPathsAndProgressBoundaries(t *testing.T) {
	ctx := t.Context()
	s := newScene(t)
	errBoom := errors.New("boom")

	// Duplicate claim returns ErrAlreadyTranscribing.
	key, ok := s.transcriber.claim(s.vault.Vault().ID, lecture)
	if !ok {
		t.Fatal("expected first claim to succeed")
	}
	if _, ok2 := s.transcriber.claim(s.vault.Vault().ID, lecture); ok2 {
		t.Fatal("expected second claim while running to fail")
	}
	if err := s.transcriber.Transcribe(ctx, s.vault.Vault().ID, lecture); !errors.Is(err, ErrAlreadyTranscribing) {
		t.Fatalf("expected ErrAlreadyTranscribing, got %v", err)
	}
	s.transcriber.clearRunning(key)

	// ReportTask error at start of Transcribe.
	fq := &faultyQueue{Queue: s.queue, reportErr: errBoom}
	trReportErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s.vault.With(id) }}, storesOf(s.store), textsOf(s.texts), fq,
		func(context.Context) (Recognizer, error) { return s.model, nil }, noReport)
	if err := trReportErr.Transcribe(ctx, s.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected ReportTask errBoom, got %v", err)
	}

	// ReadFile error on missing recording.
	if err := s.transcriber.Transcribe(ctx, s.vault.Vault().ID, "missing.wav"); err == nil {
		t.Fatal("expected error for missing recording")
	}

	// hasWhole store error in run.
	fsRead := &faultyStore{Folder: s.store, readErr: func(string) error { return errBoom }}
	trStoreErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s.vault.With(id) }}, foldersOf(fsRead), textsOf(s.texts), s.queue,
		func(context.Context) (Recognizer, error) { return s.model, nil }, noReport)
	if err := trStoreErr.Transcribe(ctx, s.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected store error from hasWhole, got %v", err)
	}

	// putWholeVersion branches: ListVersions error, non-matching versions vs matching version, PutVersion error.
	ft := &faultyTexts{TextVersions: s.texts, listErr: errBoom}
	j := &job{path: lecture, key: "k1", fileHash: "h1", texts: ft}
	if err := s.transcriber.putWholeVersion(ctx, j); !errors.Is(err, errBoom) {
		t.Fatalf("expected listErr, got %v", err)
	}
	ft.listErr = nil
	ft.listRes = host.TextListing{
		Versions: []domain.TextVersion{
			{Key: "other", PluginID: PluginID},
			{Key: "k1", PluginID: "other.plugin"},
			{Key: "k1", PluginID: PluginID, File: getFileName("k1")},
		},
	}
	if err := s.transcriber.putWholeVersion(ctx, j); err != nil || ft.puts != 0 {
		t.Fatalf("expected existing matching version to skip PutVersion, puts=%d, err=%v", ft.puts, err)
	}
	ft.listRes = host.TextListing{
		Versions: []domain.TextVersion{
			{Key: "other", PluginID: PluginID},
			{Key: "k1", PluginID: "other.plugin"},
		},
	}
	ft.putErr = errBoom
	if err := s.transcriber.putWholeVersion(ctx, j); !errors.Is(err, errBoom) || ft.puts != 1 {
		t.Fatalf("expected non-matching versions to call PutVersion and return putErr, puts=%d, err=%v", ft.puts, err)
	}

	// takeTurn error when EnqueueWork fails.
	fqEnq := &faultyQueue{Queue: s.queue, enqueueErr: errBoom}
	trEnqErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s.vault.With(id) }}, storesOf(s.store), textsOf(s.texts), fqEnq,
		func(context.Context) (Recognizer, error) { return s.model, nil }, noReport)
	if err := trEnqErr.Transcribe(ctx, s.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected enqueueErr, got %v", err)
	}

	// engines callback error.
	trEngineErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s.vault.With(id) }}, storesOf(s.store), textsOf(s.texts), s.queue,
		func(context.Context) (Recognizer, error) { return nil, errBoom }, noReport)
	if err := trEngineErr.Transcribe(ctx, s.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected engine open error, got %v", err)
	}

	// Zero segments in recording: hearSegments returns nil without creating partial file, finish returns ErrNotExist -> nil.
	emptyModel := &heardModel{segments: nil}
	trEmpty := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s.vault.With(id) }}, storesOf(s.store), textsOf(s.texts), s.queue,
		func(context.Context) (Recognizer, error) { return emptyModel, nil }, noReport)
	if err := trEmpty.Transcribe(ctx, s.vault.Vault().ID, lecture); err != nil {
		t.Fatalf("expected 0-segment recording to succeed, got %v", err)
	}

	// finish returning non-ErrNotExist error in run.
	fsRename := &faultyStore{Folder: s.store, renameErr: errBoom}
	trRenameErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s.vault.With(id) }}, foldersOf(fsRename), textsOf(s.texts), s.queue,
		func(context.Context) (Recognizer, error) { return s.model, nil }, noReport)
	if err := trRenameErr.Transcribe(ctx, s.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected finish renameErr, got %v", err)
	}

	// hearSegments errors: readReached error, GetSegments error, Transcribe error, appendBatch error, putVersion error, yieldTurn error.
	s2 := newScene(t)
	s2.model.segmentErr = errBoom
	if err := s2.transcriber.Transcribe(ctx, s2.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected segmentErr, got %v", err)
	}
	s2.model.segmentErr = nil
	s2.model.transcribeErr = errBoom
	if err := s2.transcriber.Transcribe(ctx, s2.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected transcribeErr, got %v", err)
	}
	s2.model.transcribeErr = nil

	fsReachedErr := &faultyStore{Folder: s2.store, readErr: func(name string) error {
		if strings.HasSuffix(name, ".reached") {
			return errBoom
		}
		return nil
	}}
	trReachedErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s2.vault.With(id) }}, foldersOf(fsReachedErr), textsOf(s2.texts), s2.queue,
		func(context.Context) (Recognizer, error) { return s2.model, nil }, noReport)
	if err := trReachedErr.Transcribe(ctx, s2.vault.Vault().ID, lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected readReached error, got %v", err)
	}

	fsAppendErr := &faultyStore{Folder: hosttest.New("Tmp").Store(), appendErr: errBoom}
	trAppendErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s2.vault.With(id) }}, foldersOf(fsAppendErr), textsOf(s2.texts), s2.queue,
		func(context.Context) (Recognizer, error) { return s2.model, nil }, noReport)
	if err := trAppendErr.Transcribe(ctx, "", lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected appendBatch error, got %v", err)
	}

	ftPutErr := &faultyTexts{TextVersions: s2.texts, putErr: errBoom}
	trPutErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s2.vault.With(id) }}, storesOf(hosttest.New("Lectures").Store()), func(domain.VaultID) host.TextVersions { return ftPutErr }, s2.queue,
		func(context.Context) (Recognizer, error) { return s2.model, nil }, noReport)
	if err := trPutErr.Transcribe(ctx, "", lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected putVersion error, got %v", err)
	}

	waitingCh := make(chan struct{})
	close(waitingCh)
	fqYieldErr := &faultyQueue{Queue: s2.queue, turn: &faultyTurn{waiting: waitingCh, releaseErr: errBoom}}
	trYieldErr := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s2.vault.With(id) }}, storesOf(hosttest.New("Lectures").Store()), textsOf(s2.texts), fqYieldErr,
		func(context.Context) (Recognizer, error) { return s2.model, nil }, noReport)
	if err := trYieldErr.Transcribe(ctx, "", lecture); !errors.Is(err, errBoom) {
		t.Fatalf("expected yieldTurn releaseErr, got %v", err)
	}

	// reportProgress division by 1000 boundaries.
	fqProgress := &faultyQueue{Queue: s2.queue}
	trProgress := NewTranscriber(vaultFiles{vaults: func(id domain.VaultID) vaultReader { return s2.vault.With(id) }}, storesOf(s2.store), textsOf(s2.texts), fqProgress,
		func(context.Context) (Recognizer, error) { return s2.model, nil }, noReport)
	trProgress.reportProgress(ctx, "Lectures", lecture, 999, 1998)
	trProgress.reportProgress(ctx, "Lectures", lecture, 1000, 2000)
	if len(fqProgress.tasks) != 2 {
		t.Fatalf("expected 2 progress tasks, got %d", len(fqProgress.tasks))
	}
	if fqProgress.tasks[0].CompletedCount != 0 || fqProgress.tasks[0].TotalCount != 1 {
		t.Fatalf("expected (0, 1) for (999, 1998), got (%d, %d)", fqProgress.tasks[0].CompletedCount, fqProgress.tasks[0].TotalCount)
	}
	if fqProgress.tasks[1].CompletedCount != 1 || fqProgress.tasks[1].TotalCount != 2 {
		t.Fatalf("expected (1, 2) for (1000, 2000), got (%d, %d)", fqProgress.tasks[1].CompletedCount, fqProgress.tasks[1].TotalCount)
	}
}

func TestStartReportsTaskBeforeReturning(t *testing.T) {
	s := newRequestFixture(t)

	started, err := s.transcriber.Start(t.Context(), s.vault.Vault().ID, lecture)
	if err != nil || !started {
		t.Fatalf("Start returned (%v, %v)", started, err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 1 || tasks[0].Detail != lecture {
		t.Errorf("tasks right after Start = %+v, want the task reported", tasks)
	}
	close(s.blocking.release)
	s.waitFinished(t, getTaskID("Lectures", lecture))
}

func noReport(string, error) {}

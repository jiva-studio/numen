package link

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/linkrecord"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/selector"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/ytdlp"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/random"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
)

func TestTheSameAddressInAnotherVaultIsOtherWork(t *testing.T) {
	l := &Links{running: map[string]chan struct{}{}}
	physics := Link{Vault: "Physics", Key: "k1"}
	chemistry := Link{Vault: "Chemistry", Key: "k1"}

	if !l.claim("text", physics, physics.Key) || !l.claim("text", chemistry, chemistry.Key) {
		t.Fatal("a fetch of one address in a second vault waits for the first")
	}
	if l.claim("text", physics, physics.Key) {
		t.Error("a second fetch of one address in one vault is let run")
	}
	l.release("text", physics, physics.Key)
	if !l.claim("text", physics, physics.Key) {
		t.Error("a released fetch is still held")
	}
}

func TestTimeSelectorEncodeRejectsNaN(t *testing.T) {
	if _, err := (selector.Time{T: []float64{math.NaN()}}).Encode(); err == nil {
		t.Fatal("expected error encoding NaN timeSelector")
	}
}

type hookStore struct {
	*hosttest.Store
	withVault domain.VaultID
	readErr   func(name string) error
	writeErr  func(name string) error
	appendErr func(name string) error
	deleteErr func(name string) error
	listErr   func(call int) error
	renameErr error
	listCalls int
}

func (s *hookStore) Read(ctx context.Context, name string) ([]byte, error) {
	if s.readErr != nil {
		if err := s.readErr(name); err != nil {
			return nil, err
		}
	}
	return s.Store.Read(ctx, name)
}

func (s *hookStore) Write(ctx context.Context, name string, data []byte) error {
	if s.writeErr != nil {
		if err := s.writeErr(name); err != nil {
			return err
		}
	}
	return s.Store.Write(ctx, name, data)
}

func (s *hookStore) Append(ctx context.Context, name string, data []byte) error {
	if s.appendErr != nil {
		if err := s.appendErr(name); err != nil {
			return err
		}
	}
	return s.Store.Append(ctx, name, data)
}

func (s *hookStore) Delete(ctx context.Context, name string) error {
	if s.deleteErr != nil {
		if err := s.deleteErr(name); err != nil {
			return err
		}
	}
	return s.Store.Delete(ctx, name)
}

func (s *hookStore) List(ctx context.Context) ([]host.StoreFile, error) {
	s.listCalls++
	if s.listErr != nil {
		if err := s.listErr(s.listCalls); err != nil {
			return nil, err
		}
	}
	return s.Store.List(ctx)
}

func (s *hookStore) Rename(ctx context.Context, from, to string) error {
	if s.renameErr != nil {
		return s.renameErr
	}
	return s.Store.Rename(ctx, from, to)
}

type hookTexts struct {
	Texts
	fake      *hosttest.Texts
	withVault domain.VaultID
	putErr    error
}

func (x *hookTexts) PutVersion(ctx context.Context, req host.TextPut) (domain.TextVersion, error) {
	if x.putErr != nil {
		return domain.TextVersion{}, x.putErr
	}
	return x.Texts.PutVersion(ctx, req)
}

type hookVault struct {
	*hosttest.Vault
	getVaultErr error
}

func (v *hookVault) GetVault(ctx context.Context) (domain.Vault, error) {
	if v.getVaultErr != nil {
		return domain.Vault{}, v.getVaultErr
	}
	return v.Vault.GetVault(ctx)
}

type hookQueue struct {
	host.Queue
	enqueueErr error
	releaseErr error
	tasks      []host.Task
	finishCtx  context.Context
}

type hookTurn struct {
	host.Turn
	releaseErr error
}

func (t hookTurn) Release() error {
	return errors.Join(t.Turn.Release(), t.releaseErr)
}

func (q *hookQueue) EnqueueWork(ctx context.Context, work host.Work) (host.Turn, error) {
	if q.enqueueErr != nil {
		return nil, q.enqueueErr
	}
	turn, err := q.Queue.EnqueueWork(ctx, work)
	if err != nil {
		return nil, err
	}
	return hookTurn{Turn: turn, releaseErr: q.releaseErr}, nil
}

func (q *hookQueue) ReportTask(ctx context.Context, task host.Task) error {
	q.tasks = append(q.tasks, task)
	return q.Queue.ReportTask(ctx, task)
}

func (q *hookQueue) FinishTask(ctx context.Context, id string) error {
	q.finishCtx = ctx
	return q.Queue.FinishTask(ctx, id)
}

type stubFetcher struct {
	canCopy   bool
	meta      fetch.Metadata
	metaErr   error
	text      fetch.Text
	textErr   error
	copyRes   fetch.Copy
	copyBytes []byte
	copyErr   error
	downloads int
	onText    func(ctx context.Context) (fetch.Text, error)
}

func (f *stubFetcher) CanCopy(string) bool { return f.canCopy }

func (f *stubFetcher) GetDownloadModel(string) fetch.Model {
	return fetch.Model{Tool: "tool", Version: "v2", Producer: "captions"}
}

func (f *stubFetcher) GetMetadata(context.Context, string) (fetch.Metadata, error) {
	return f.meta, f.metaErr
}

func (f *stubFetcher) GetText(ctx context.Context, _ string, _ fetch.PreferredCaptions) (fetch.Text, error) {
	if f.onText != nil {
		return f.onText(ctx)
	}
	return f.text, f.textErr
}

func (f *stubFetcher) Download(_ context.Context, _ string, _ fetch.Media, into io.Writer) (fetch.Copy, error) {
	f.downloads++
	if len(f.copyBytes) > 0 {
		if _, err := into.Write(f.copyBytes); err != nil {
			return fetch.Copy{}, err
		}
	}
	return f.copyRes, f.copyErr
}

func TestFolderOperationsAndErrorBranches(t *testing.T) {
	ctx := t.Context()
	if got := getKind(fetch.Text{}); got != kindArticle {
		t.Errorf("getKind(empty) = %q, want %q", got, kindArticle)
	}
	if got := getKind(fetch.Text{Cues: []ytdlp.Cue{{Text: "one"}}}); got != kindCaptions {
		t.Errorf("getKind(1 cue) = %q, want %q", got, kindCaptions)
	}

	v := hosttest.New("Vault")
	hs := &hookStore{Store: v.Store()}
	f := folder{store: Folder{Files: hs, Writer: hs}}

	rec, ok, err := f.findRecord(ctx, "missing")
	if err != nil || ok || rec != (linkrecord.Record{}) {
		t.Errorf("findRecord(missing) = %+v, %v, %v", rec, ok, err)
	}

	boom := errors.New("boom")
	hs.readErr = func(string) error { return boom }
	if _, ok, err := f.findRecord(ctx, "k"); !errors.Is(err, boom) || ok {
		t.Errorf("findRecord(error) = %v, %v", ok, err)
	}
	hs.readErr = nil

	if err := f.writeRecord(ctx, "k", linkrecord.Record{Address: "https://a.example", Kind: kindArticle}); err != nil {
		t.Fatal(err)
	}
	rec, ok, err = f.findRecord(ctx, "k")
	if err != nil || !ok || rec.Address != "https://a.example" {
		t.Errorf("findRecord(present) = %+v, %v, %v", rec, ok, err)
	}

	hs.writeErr = func(string) error { return boom }
	if err := f.writeText(ctx, "k", []domain.TextSegment{{Text: "hi"}}, rec); !errors.Is(err, boom) {
		t.Errorf("writeText(error) = %v", err)
	}
	hs.writeErr = nil

	hs.listErr = func(int) error { return boom }
	if _, ok, err := f.findCopy(ctx, "h1", fetch.MediaVideo); !errors.Is(err, boom) || ok {
		t.Errorf("findCopy(error) = %v, %v", ok, err)
		if _, err := f.listCopies(ctx, "h1"); !errors.Is(err, boom) {
			t.Errorf("listCopies(list error) = %v", err)
		}
	}
	hs.listErr = nil

	found, ok, err := f.findCopy(ctx, "h1", fetch.MediaVideo)
	if err != nil || ok || found != (host.StoreFile{}) {
		t.Errorf("findCopy(empty) = %+v, %v, %v", found, ok, err)
	}

	if err := hs.Write(ctx, "000_other.txt", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := hs.Write(ctx, "h1.json", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if err := hs.Write(ctx, "h1.part", []byte("part")); err != nil {
		t.Fatal(err)
	}
	if err := hs.Write(ctx, "h1.video.mp4", []byte("mp4")); err != nil {
		t.Fatal(err)
	}
	found, ok, err = f.findCopy(ctx, "h1", fetch.MediaVideo)
	if err != nil || !ok || found.Name != "h1.video.mp4" || found.Size != 3 {
		t.Errorf("findCopy(match) = %+v, %v, %v", found, ok, err)
	}
	if err := hs.Write(ctx, "h1.audio.m4a", []byte("m4a")); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := f.findCopy(ctx, "h1", fetch.MediaAudio); err != nil || !ok {
		t.Errorf("findCopy(audio) = %v, %v", ok, err)
	}
	names, err := f.listCopies(ctx, "h1")
	if err != nil || len(names) != 2 {
		t.Errorf("listCopies = %v, %v, want the video and the sound alone", names, err)
	}
}

func TestStoreWriterLimitsAndChunks(t *testing.T) {
	var chunks [][]byte
	var reports []int64
	w := &storeWriter{
		limit: copyChunk + 5,
		appendChunk: func(chunk []byte) error {
			chunks = append(chunks, bytes.Clone(chunk))
			return nil
		},
		report: func(n int64) {
			reports = append(reports, n)
		},
	}
	if err := w.flush(); err != nil || len(chunks) != 0 {
		t.Fatalf("empty flush = %v, chunks = %d", err, len(chunks))
	}
	block := make([]byte, copyChunk)
	n, err := w.Write(block)
	if err != nil || n != copyChunk || len(chunks) != 1 || len(chunks[0]) != copyChunk || len(reports) != 1 || reports[0] != copyChunk {
		t.Fatalf("Write(copyChunk) = (%d, %v), chunks=%d, reports=%v", n, err, len(chunks), reports)
	}
	n, err = w.Write([]byte("12345"))
	if err != nil || n != 5 {
		t.Fatalf("Write(5) = (%d, %v)", n, err)
	}
	if n, err := w.Write([]byte("6")); !errors.Is(err, ErrTooLarge) || n != 0 {
		t.Fatalf("Write past limit = (%d, %v), want (0, ErrTooLarge)", n, err)
	}
	if err := w.flush(); err != nil || len(chunks) != 2 || string(chunks[1]) != "12345" || reports[1] != copyChunk+5 {
		t.Fatalf("flush = %v, chunks=%d, reports=%v", err, len(chunks), reports)
	}

	boom := errors.New("append failed")
	wErr := &storeWriter{
		limit:       1,
		appendChunk: func([]byte) error { return boom },
		report:      func(int64) {},
	}
	if _, err := wErr.Write([]byte("ab")); !errors.Is(err, ErrTooLarge) {
		t.Errorf("Write(2) with limit 1 = %v, want ErrTooLarge", err)
	}
	if _, err := wErr.Write([]byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := wErr.flush(); !errors.Is(err, boom) {
		t.Errorf("flush with append error = %v, want %v", err, boom)
	}
}

func createTestLinks() (l *Links, v *hosttest.Vault, hv *hookVault, hs *hookStore, ht *hookTexts, hq *hookQueue, sf *stubFetcher, copyMaxSize *int64) {
	v = hosttest.New("Mills")
	hv = &hookVault{Vault: v}
	hs = &hookStore{Store: v.Store()}
	texts := hosttest.NewTexts(v.Vault().ID, "plug")
	texts.SetStore("plug", v.Store())
	ht = &hookTexts{Texts: texts, fake: texts}
	hq = &hookQueue{Queue: hosttest.NewQueue()}
	sf = &stubFetcher{
		canCopy:   true,
		meta:      fetch.Metadata{Bytes: 4},
		text:      fetch.Text{Producer: "article", Title: "T", Prose: "body"},
		copyRes:   fetch.Copy{MediaType: "video/mp4", Extension: ".mp4"},
		copyBytes: []byte("data"),
	}
	copyMaxSize = new(int64)
	l = New(Host{Vaults: func(id domain.VaultID) Vault {
		if id == "" {
			return Vault{Files: hv, Writer: hv}
		}
		other := v.With(id)
		return Vault{Files: other, Writer: other}
	}, Store: func(id domain.VaultID) Folder {
		hs.withVault = id
		return Folder{Files: hs, Writer: hs}
	}, Texts: func(id domain.VaultID) Texts {
		ht.withVault = id
		return ht
	}, Queue: hq, Clock: newFakeClock(time.Unix(1_700_000_000, 0)), Entropy: random.Entropy{}},
		func() ports.Fetcher { return sf },
		func() fetch.Settings { return fetch.Settings{CopyMaxSize: *copyMaxSize} })
	return l, v, hv, hs, ht, hq, sf, copyMaxSize
}

func TestLinksCloseReadAndStoreScoping(t *testing.T) {
	ctx := t.Context()
	l, v, hv, hs, ht, _, sf, _ := createTestLinks()

	// Scoping with empty vs non-empty Vault.
	hs.withVault = "dirty"
	_ = l.getStore(Link{Vault: ""})
	if hs.withVault != "" {
		t.Errorf("getStore(empty) asked for the folder of %q", hs.withVault)
	}
	_ = l.getStore(Link{Vault: "v-1"})
	if hs.withVault != "v-1" {
		t.Errorf("getStore(v-1) withVault = %q", hs.withVault)
	}
	ht.withVault = "dirty"
	_ = l.getTexts(Link{Vault: ""})
	if ht.withVault != "" {
		t.Errorf("getTexts(empty) asked for the texts of %q", ht.withVault)
	}
	_ = l.getTexts(Link{Vault: "v-1"})
	if ht.withVault != "v-1" {
		t.Errorf("getTexts(v-1) withVault = %q", ht.withVault)
	}

	// Read edge cases.
	v.File("notes/note.txt", "---\ntype: link\nurl: https://example.com/\n---\n")
	if _, err := l.Read(ctx, "notes/note.txt"); !errors.Is(err, ErrNotALink) {
		t.Errorf("Read(.txt) = %v, want ErrNotALink", err)
	}
	if _, err := l.Read(ctx, "notes/missing.md"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("Read(missing.md) = %v, want ErrNotExist", err)
	}
	v.File("notes/empty-url.md", "---\ntype: link\nurl: \"\"\n---\n")
	if _, err := l.Read(ctx, "notes/empty-url.md"); !errors.Is(err, ErrNotALink) {
		t.Errorf("Read(empty url) = %v, want ErrNotALink", err)
	}
	v.File("notes/num-url.md", "---\ntype: link\nurl: 42\n---\n")
	if _, err := l.Read(ctx, "notes/num-url.md"); !errors.Is(err, ErrNotALink) {
		t.Errorf("Read(numeric url) = %v, want ErrNotALink", err)
	}
	v.File("notes/bad-url.md", "---\ntype: link\nurl: ftp://example.com/file\n---\n")
	if _, err := l.Read(ctx, "notes/bad-url.md"); !errors.Is(err, ErrNotALink) || !errors.Is(err, address.ErrNotAURL) {
		t.Errorf("Read(ftp url) = %v, want ErrNotALink and ErrNotAURL", err)
	}

	boom := errors.New("vault error")
	hv.getVaultErr = boom
	v.File("notes/good.md", "---\ntype: link\nurl: https://example.com/\n---\n")
	if _, err := l.Read(ctx, "notes/good.md"); !errors.Is(err, boom) {
		t.Errorf("Read(getVault error) = %v, want %v", err, boom)
	}
	hv.getVaultErr = nil

	// Close cancels in-flight background work and waits for it.
	started := make(chan struct{})
	var waited atomic.Bool
	sf.onText = func(runCtx context.Context) (fetch.Text, error) {
		close(started)
		select {
		case <-runCtx.Done():
			time.Sleep(15 * time.Millisecond)
			waited.Store(true)
			return fetch.Text{}, runCtx.Err()
		case <-time.After(500 * time.Millisecond):
			return fetch.Text{}, errors.New("close did not cancel context")
		}
	}
	l.StartFetch(ctx, Link{Path: "notes/good.md", Address: "https://example.com/", Key: "k1"})
	<-started
	l.Close()
	if !waited.Load() {
		t.Error("Close returned before background fetch finished")
	}
}

func TestLinksFetchCopyAndDeleteErrorBranches(t *testing.T) {
	ctx := t.Context()
	l, v, _, hs, ht, hq, sf, maxSize := createTestLinks()
	t.Cleanup(l.Close)

	made, err := l.Create(ctx, "", "links", "https://example.com/a")
	if err != nil {
		t.Fatal(err)
	}
	rawNote, _ := v.Content(made.Path)
	ht.fake.File(made.Path, []byte(rawNote))

	// Claim contention on FetchText and Copy.
	l.claim("text", made, made.Key)
	if err := l.FetchText(ctx, made); !errors.Is(err, ErrBeingFetched) {
		t.Errorf("FetchText while claimed = %v, want ErrBeingFetched", err)
	}
	l.release("text", made, made.Key)

	l.claim("copy", made, made.Key)
	if size, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, ErrBeingFetched) || size != 0 {
		t.Errorf("Copy while claimed = (%d, %v), want (0, ErrBeingFetched)", size, err)
	}
	l.release("copy", made, made.Key)

	// readRecord error paths.
	boom := errors.New("store failure")
	hs.readErr = func(name string) error {
		if name == made.Key+recordExt {
			return boom
		}
		return nil
	}
	if err := l.FetchText(ctx, made); !errors.Is(err, boom) {
		t.Errorf("FetchText(readRecord err) = %v, want %v", err, boom)
	}
	hs.readErr = nil

	// FetchText when fetcher returns error, and when writeText fails.
	sf.textErr = boom
	if err := l.FetchText(ctx, made); !errors.Is(err, boom) {
		t.Errorf("FetchText(fetcher err) = %v, want %v", err, boom)
	}
	sf.textErr = nil
	hs.writeErr = func(string) error { return boom }
	if err := l.FetchText(ctx, made); !errors.Is(err, boom) {
		t.Errorf("FetchText(writeText err) = %v, want %v", err, boom)
	}
	hs.writeErr = nil

	// Successful FetchText records downloader and reports initial task with 0/0 counts.
	hq.tasks = nil
	if err := l.FetchText(ctx, made); err != nil {
		t.Fatal(err)
	}
	if len(hq.tasks) == 0 || hq.tasks[0].CompletedCount != 0 || hq.tasks[0].TotalCount != 0 {
		t.Errorf("FetchText initial task = %+v", hq.tasks)
	}
	rec, err := (folder{store: Folder{Files: hs, Writer: hs}}).readRecord(ctx, made.Key)
	if err != nil || rec.Downloader != "tool v2" {
		t.Errorf("record Downloader = %q (%v), want %q", rec.Downloader, err, "tool v2")
	}

	// endTask with canceled context still finishes the task using an uncanceled context.
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	l.endTask(canceledCtx, host.Task{ID: "t1"}, context.Canceled)
	if hq.finishCtx == nil || hq.finishCtx.Err() != nil {
		t.Errorf("endTask finishCtx err = %v, want nil", hq.finishCtx)
	}

	// Copy fails where the folder cannot be listed.
	hs.listErr = func(int) error { return boom }
	if _, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, boom) {
		t.Errorf("Copy(list err) = %v, want %v", err, boom)
	}
	hs.listErr = nil

	// Copy error and boundary paths.
	sf.canCopy = false
	if size, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, ErrNotCopyable) || size != 0 {
		t.Errorf("Copy(not copyable) = (%d, %v)", size, err)
	}
	sf.canCopy = true

	sf.metaErr = boom
	if _, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, boom) {
		t.Errorf("Copy(metaErr) = %v", err)
	}
	sf.metaErr = nil

	*maxSize = 3
	sf.meta = fetch.Metadata{Bytes: 4}
	if size, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, ErrTooLarge) || size != 4 {
		t.Errorf("Copy(too large meta) = (%d, %v), want (4, ErrTooLarge)", size, err)
	}
	*maxSize = 1
	sf.meta = fetch.Metadata{Bytes: 2}
	if size, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, ErrTooLarge) || size != 2 {
		t.Errorf("Copy(meta 2 > limit 1) = (%d, %v), want (2, ErrTooLarge)", size, err)
	}
	*maxSize = 4
	sf.meta = fetch.Metadata{Bytes: 4}

	hq.enqueueErr = boom
	if _, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, boom) {
		t.Errorf("Copy(enqueueErr) = %v", err)
	}
	hq.enqueueErr = nil

	hs.deleteErr = func(name string) error {
		if name == made.Key+partExt {
			return boom
		}
		return nil
	}
	if _, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, boom) {
		t.Errorf("Copy(delete part err) = %v", err)
	}
	hs.deleteErr = nil

	hs.appendErr = func(string) error { return boom }
	if size, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, boom) || size != 0 {
		t.Errorf("Copy(flush appendErr) = (%d, %v)", size, err)
	}
	hs.appendErr = nil

	sf.copyRes = fetch.Copy{MediaType: "application/octet-stream", Extension: ".exe"}
	if _, err := l.Copy(ctx, made, fetch.MediaVideo); err == nil {
		t.Error("Copy(.exe) succeeded")
	}
	sf.copyRes = fetch.Copy{MediaType: "video/mp4", Extension: ".mp4"}

	hs.renameErr = boom
	if _, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, boom) {
		t.Errorf("Copy(renameErr) = %v", err)
	}
	hs.renameErr = nil

	hq.releaseErr = boom
	if _, err := l.Copy(ctx, made, fetch.MediaVideo); !errors.Is(err, boom) {
		t.Errorf("Copy(releaseErr) = %v", err)
	}
	hq.releaseErr = nil
	if err := hs.Delete(ctx, made.Key+".video.mp4"); err != nil {
		t.Fatal(err)
	}

	// Copy with 1-byte file at exact maxSize=1, then 2nd Copy reuses existing 1-byte file without downloading.
	*maxSize = 1
	sf.meta = fetch.Metadata{Bytes: 1}
	sf.copyBytes = []byte{0x7f}
	sf.downloads = 0
	hq.tasks = nil
	size, err := l.Copy(ctx, made, fetch.MediaVideo)
	if err != nil || size != 1 || sf.downloads != 1 {
		t.Fatalf("Copy(1 byte) = (%d, %v), downloads=%d", size, err, sf.downloads)
	}
	if len(hq.tasks) < 2 || hq.tasks[0].CompletedCount != 0 || hq.tasks[0].TotalCount != 0 || hq.tasks[1].CompletedCount != 1 || hq.tasks[1].TotalCount != 1 {
		t.Errorf("Copy tasks = %+v", hq.tasks)
	}
	size, err = l.Copy(ctx, made, fetch.MediaVideo)
	if err != nil || size != 1 || sf.downloads != 1 {
		t.Fatalf("second Copy(1 byte) = (%d, %v), downloads=%d", size, err, sf.downloads)
	}

	// DeleteCopy fails where the folder cannot be listed, and deletes the copy no note refers to.
	hs.listErr = func(int) error { return boom }
	if err := l.DeleteCopy(ctx, made); !errors.Is(err, boom) {
		t.Errorf("DeleteCopy(listErr) = %v", err)
	}
	hs.listErr = nil
	if err := l.DeleteCopy(ctx, made); err != nil {
		t.Fatal(err)
	}
	if _, err := hs.Read(ctx, made.Key+".video.mp4"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the copy no note refers to is kept: %v", err)
	}
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}

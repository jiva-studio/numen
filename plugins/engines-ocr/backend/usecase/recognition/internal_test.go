package recognition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/progress"
)

func encodeTestPNG(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

type stubDocument struct {
	pages     int
	renderErr error
}

func (d stubDocument) GetPageCount() int  { return d.pages }
func (d stubDocument) HasTextLayer() bool { return false }
func (d stubDocument) Close() error       { return nil }

func (d stubDocument) RenderPage(_, _ int) (image.Image, error) {
	if d.renderErr != nil {
		return nil, d.renderErr
	}
	return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
}

type stubRecognizer struct {
	blocks []Block
	err    error
	fn     func(ctx context.Context, page image.Image) ([]Block, error)
}

func (s *stubRecognizer) GetRecognitionModel() Model {
	return Model{Layout: "layout", Recognizer: "rec", DPI: 72, From: "stub"}
}

func (s *stubRecognizer) Recognize(ctx context.Context, page image.Image) ([]Block, error) {
	if s.fn != nil {
		return s.fn(ctx, page)
	}
	return s.blocks, s.err
}

type stubTurn struct {
	onRelease  func()
	releaseErr error
	waiting    chan struct{}
}

func (t stubTurn) Release() error {
	if t.onRelease != nil {
		t.onRelease()
	}
	return t.releaseErr
}

func (t stubTurn) WatchWaiting() <-chan struct{} { return t.waiting }

type stubQueue struct {
	mu          sync.Mutex
	enqueueErr  error
	onRelease   func()
	releaseErr  error
	reportErr   error
	reportErrAt int
	finishErr   error
	reported    []host.Task
	finished    []string
}

func (q *stubQueue) EnqueueWork(context.Context, host.Work) (host.Turn, error) {
	if q.enqueueErr != nil {
		return nil, q.enqueueErr
	}
	return stubTurn{onRelease: q.onRelease, releaseErr: q.releaseErr, waiting: make(chan struct{})}, nil
}

func (q *stubQueue) ReportTask(_ context.Context, task host.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.reported = append(q.reported, task)
	if q.reportErrAt > 0 && len(q.reported) == q.reportErrAt {
		return q.reportErr
	}
	if q.reportErrAt == 0 && q.reportErr != nil {
		return q.reportErr
	}
	return nil
}

func (q *stubQueue) FinishTask(_ context.Context, id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.finished = append(q.finished, id)
	return q.finishErr
}

type stubStore struct {
	Folder
	readData        []byte
	readErr         error
	writeErr        error
	deleteErr       error
	recordedLengths []int
}

func (s *stubStore) Read(ctx context.Context, name string) ([]byte, error) {
	if s.readErr != nil || s.readData != nil {
		return s.readData, s.readErr
	}
	return s.Folder.Read(ctx, name)
}

func (s *stubStore) Write(ctx context.Context, name string, data []byte) error {
	if strings.HasSuffix(name, ".progress.json") {
		var rec progress.Record
		if err := json.Unmarshal(data, &rec); err == nil {
			s.recordedLengths = append(s.recordedLengths, len(rec.Pages))
		}
	}
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.Folder.Write(ctx, name, data)
}

func (s *stubStore) Delete(ctx context.Context, name string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.Folder.Delete(ctx, name)
}

type stubTexts struct {
	host.TextVersions
	listRes host.TextListing
	listErr error
	putErr  error
}

func (t *stubTexts) ListVersions(ctx context.Context, path string) (host.TextListing, error) {
	if t.listErr != nil || len(t.listRes.Versions) > 0 {
		return t.listRes, t.listErr
	}
	return t.TextVersions.ListVersions(ctx, path)
}

func (t *stubTexts) PutVersion(ctx context.Context, req host.TextPut) (domain.TextVersion, error) {
	if t.putErr != nil {
		return domain.TextVersion{}, t.putErr
	}
	return t.TextVersions.PutVersion(ctx, req)
}

func TestReadRecordHandlesErrorsAndCorruptJSON(t *testing.T) {
	v := hosttest.New("Physics")
	readSentinel := errors.New("disk read failed")
	s := &stubStore{Folder: v.Store(), readData: []byte("{}"), readErr: readSentinel}
	if rec, err := readRecord(t.Context(), s, "books/scan.pdf"); rec != nil || !errors.Is(err, readSentinel) {
		t.Fatalf("readRecord on read error: %+v, %v", rec, err)
	}

	s = &stubStore{Folder: v.Store(), readData: []byte("{invalid-json"), readErr: nil}
	if rec, err := readRecord(t.Context(), s, "books/scan.pdf"); rec != nil || err == nil {
		t.Fatalf("readRecord on invalid JSON: %+v, %v", rec, err)
	}
}

func TestStoringVersionFailsOnInvalidSelector(t *testing.T) {
	rec := &progress.Record{
		Path:      "books/scan.pdf",
		InputHash: "h1",
		Recipe:    "r1",
		PageCount: 1,
		Pages:     [][]progress.Part{{{Text: "bad", Rects: [][4]float32{{float32(math.NaN()), 0, 1, 1}}}}},
	}
	v := hosttest.New("Physics")
	texts := hosttest.NewTexts("Physics", "md.numen.engines.ocr")
	if _, err := putVersion(t.Context(), reach{vault: v, store: v.Store(), texts: texts}, rec); err == nil {
		t.Fatal("putVersion succeeded on NaN rect")
	}
}

func TestNewBlockPartsAndFractions(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 100)
	parts := newBlockParts(bounds, Block{
		Text: "abcdef",
		Boxes: []Box{
			{Rect: image.Rect(0, 0, 10, 10), Range: ByteRange{Start: 0, End: 0}},
			{Rect: image.Rect(0, 20, 50, 40), Range: ByteRange{Start: 0, End: 3}},
			{Rect: image.Rect(0, 40, 50, 80), Range: ByteRange{Start: 3, End: 6}},
		},
	})
	if len(parts) != 2 || parts[0].Text != "abc" || parts[0].Rects[0][1] != 0.2 || parts[1].Text != "def" {
		t.Fatalf("duplicate start parts: %+v", parts)
	}

	prefixParts := newBlockParts(bounds, Block{
		Text: "abcdef",
		Boxes: []Box{
			{Rect: image.Rect(0, 0, 50, 50), Range: ByteRange{Start: 2, End: 4}},
			{Rect: image.Rect(0, 50, 50, 100), Range: ByteRange{Start: 4, End: 6}},
		},
	})
	if len(prefixParts) != 2 || prefixParts[0].Text != "abcd" || prefixParts[1].Text != "ef" {
		t.Fatalf("non-zero first box start parts: %+v", prefixParts)
	}

	negParts := newBlockParts(bounds, Block{
		Text: "abcdef",
		Boxes: []Box{
			{Rect: image.Rect(0, 0, 50, 50), Range: ByteRange{Start: -5, End: 0}},
			{Rect: image.Rect(0, 50, 50, 100), Range: ByteRange{Start: -1, End: 6}},
		},
	})
	if len(negParts) != 1 || negParts[0].Text != "abcdef" {
		t.Fatalf("negative box start parts: %+v", negParts)
	}

	for _, emptyPage := range []image.Rectangle{
		{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 0, Y: 10}},
		{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 10, Y: 0}},
		{Min: image.Point{X: 5, Y: 0}, Max: image.Point{X: 2, Y: 10}},
		{Min: image.Point{X: 0, Y: 5}, Max: image.Point{X: 10, Y: 2}},
	} {
		if got := getFractions(emptyPage, image.Rect(0, 0, 5, 5)); got != [4]float32{} {
			t.Fatalf("getFractions(%v) = %v, want zero", emptyPage, got)
		}
	}

	if got := getFractions(image.Rect(0, 0, 1, 2), image.Rect(0, 0, 1, 2)); got != [4]float32{0, 0, 1, 1} {
		t.Fatalf("getFractions 1x2 = %v", got)
	}
	if got := getFractions(image.Rect(0, 0, 2, 1), image.Rect(0, 0, 2, 1)); got != [4]float32{0, 0, 1, 1} {
		t.Fatalf("getFractions 2x1 = %v", got)
	}
	if got := getFractions(image.Rect(10, 20, 110, 220), image.Rect(35, 70, 85, 170)); got != [4]float32{0.25, 0.25, 0.75, 0.75} {
		t.Fatalf("getFractions offset rect = %v, want [0.25 0.25 0.75 0.75]", got)
	}
}

func TestReadPagesBatchesAndReportsProgress(t *testing.T) {
	v := hosttest.New("Physics")
	q := &stubQueue{}
	s := &stubStore{Folder: v.Store()}
	texts := hosttest.NewTexts("Physics", "md.numen.engines.ocr")
	recEngine := &stubRecognizer{blocks: []Block{{Text: "line"}}}
	r, err := New(Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(s), Texts: versionsOf(texts), Queue: q}, func(context.Context) (Recognizer, error) {
		return recEngine, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := &progress.Record{Path: "books/scan.pdf", InputHash: "h1", Recipe: "r1", PageCount: 9}
	task := &host.Task{ID: "recognize:books/scan.pdf"}
	err = r.readPages(t.Context(), pageReading{
		doc:        stubDocument{pages: 9},
		recognizer: recEngine,
		dpi:        72,
		store:      s,
		task:       task,
	}, rec)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.recordedLengths) != 1 || s.recordedLengths[0] != 8 {
		t.Fatalf("progress writes at page counts %v, want [8]", s.recordedLengths)
	}
	if len(q.reported) != 10 {
		t.Fatalf("reported %d tasks, want 10", len(q.reported))
	}
	for i, snap := range q.reported {
		if snap.TotalCount != 9 || snap.CompletedCount != int64(i) {
			t.Fatalf("report %d: got %d/%d, want %d/9", i, snap.CompletedCount, snap.TotalCount, i)
		}
	}

	qInitErr := &stubQueue{reportErr: errors.New("initial report failed"), reportErrAt: 1}
	rInitErr, err := New(Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(s), Texts: versionsOf(texts), Queue: qInitErr}, func(context.Context) (Recognizer, error) {
		return recEngine, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rInitErr.readPages(t.Context(), pageReading{
		doc: stubDocument{pages: 1}, recognizer: recEngine, dpi: 72, store: s, task: &host.Task{},
	}, &progress.Record{PageCount: 1}); err == nil {
		t.Fatal("readPages succeeded when initial ReportTask failed")
	}

	if err := r.readPages(t.Context(), pageReading{
		doc: stubDocument{pages: 1, renderErr: errors.New("render failed")}, recognizer: recEngine, dpi: 72, store: s, task: &host.Task{},
	}, &progress.Record{PageCount: 1}); err == nil {
		t.Fatal("readPages succeeded when RenderPage failed")
	}

	recSentinel := errors.New("ocr engine fault")
	err = r.readPages(t.Context(), pageReading{
		doc: stubDocument{pages: 1}, recognizer: &stubRecognizer{err: recSentinel}, dpi: 72, store: s, task: &host.Task{},
	}, &progress.Record{Path: "books/scan.pdf", PageCount: 1})
	if !errors.Is(err, recSentinel) || !strings.Contains(err.Error(), "recognize page 1 of books/scan.pdf") {
		t.Fatalf("readPages recognize error = %v", err)
	}

	sWriteErr := &stubStore{Folder: v.Store(), writeErr: errors.New("store write failed")}
	if err := r.readPages(t.Context(), pageReading{
		doc: stubDocument{pages: 8}, recognizer: recEngine, dpi: 72, store: sWriteErr, task: &host.Task{},
	}, &progress.Record{Path: "books/scan.pdf", PageCount: 8}); err == nil {
		t.Fatal("readPages succeeded when batch writeRecord failed")
	}

	qStepErr := &stubQueue{reportErr: errors.New("step report failed"), reportErrAt: 2}
	rStepErr, err := New(Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(s), Texts: versionsOf(texts), Queue: qStepErr}, func(context.Context) (Recognizer, error) {
		return recEngine, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rStepErr.readPages(t.Context(), pageReading{
		doc: stubDocument{pages: 1}, recognizer: recEngine, dpi: 72, store: s, task: &host.Task{},
	}, &progress.Record{PageCount: 1}); err == nil {
		t.Fatal("readPages succeeded when step ReportTask failed")
	}
}

func TestRecognitionVersionStorage(t *testing.T) {
	v := hosttest.New("Physics")
	texts := hosttest.NewTexts("Physics", "md.numen.engines.ocr")
	texts.SetStore("md.numen.engines.ocr", v.Store())

	emptyRec := &progress.Record{Path: "books/scan.pdf", InputHash: "h1", PageCount: 1}
	delErrStore := &stubStore{Folder: v.Store(), deleteErr: errors.New("delete failed")}
	if _, err := putVersion(t.Context(), reach{vault: v, store: delErrStore, texts: texts}, emptyRec); err == nil {
		t.Fatal("putVersion on empty record succeeded when Delete failed")
	}

	fullRec := &progress.Record{
		Path:      "books/scan.pdf",
		InputHash: "h1",
		PageCount: 1,
		Pages:     [][]progress.Part{{{Text: "some text\n\n"}}},
	}
	writeErrStore := &stubStore{Folder: v.Store(), writeErr: errors.New("write failed")}
	if _, err := putVersion(t.Context(), reach{vault: v, store: writeErrStore, texts: texts}, fullRec); err == nil {
		t.Fatal("putVersion succeeded when WriteText failed")
	}

	putErrTexts := &stubTexts{TextVersions: texts, putErr: errors.New("put version failed")}
	if _, err := putVersion(t.Context(), reach{vault: v, store: v.Store(), texts: putErrTexts}, fullRec); err == nil {
		t.Fatal("putVersion succeeded when PutVersion failed")
	}

	if _, err := putVersion(t.Context(), reach{vault: v, store: delErrStore, texts: texts}, fullRec); err == nil {
		t.Fatal("putVersion on non-empty record succeeded when Delete failed")
	}

}

func TestRecognizeErrorAndCancelPaths(t *testing.T) {
	v := hosttest.New("Physics")
	pngRaw := encodeTestPNG(t)
	v.File("scan.png", pngRaw)
	texts := hosttest.NewTexts("Physics", "md.numen.engines.ocr")
	texts.SetStore("md.numen.engines.ocr", v.Store())
	texts.File("scan.png", []byte(pngRaw))

	// 1. A file that cannot be read reports a task with 0/0 counts; joined error when ReportTask also fails.
	readSentinel := errors.New("permission denied")
	v.File("refused.png", pngRaw)
	v.Refuse("refused.png", readSentinel)
	reportSentinel := errors.New("report task failed")
	qReportFail := &stubQueue{reportErr: reportSentinel}
	rReportFail, err := New(Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: qReportFail}, func(context.Context) (Recognizer, error) {
		return &stubRecognizer{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = rReportFail.Recognize(t.Context(), Request{Path: "refused.png"})
	if !errors.Is(err, readSentinel) || !errors.Is(err, reportSentinel) {
		t.Fatalf("expected joined read and report error, got %v", err)
	}
	if len(qReportFail.reported) != 1 || qReportFail.reported[0].CompletedCount != 0 || qReportFail.reported[0].TotalCount != 0 {
		t.Fatalf("unexpected initial task report: %+v", qReportFail.reported)
	}

	// 2. Canceled context skips failure ReportTask, runs FinishTask, and returns recognition result and error.
	cancelCtx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	qCancel := &stubQueue{onRelease: cancel, releaseErr: context.Canceled, finishErr: errors.New("finish ignored when recognition failed")}
	rCancel, err := New(Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: qCancel}, func(context.Context) (Recognizer, error) {
		return &stubRecognizer{blocks: []Block{{Text: "hello"}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res, err := rCancel.Recognize(cancelCtx, Request{Path: "scan.png"}); !errors.Is(err, context.Canceled) || res.Pages != 1 {
		t.Fatalf("canceled recognition returned %+v, %v; want Pages=1 and context.Canceled", res, err)
	}
	if len(qCancel.finished) != 1 {
		t.Fatalf("canceled recognition finished %v, want 1 FinishTask call", qCancel.finished)
	}

	// 3. FinishTask error returned with non-zero Result when recognition succeeds.
	finishSentinel := errors.New("finish task failed")
	qFinishErr := &stubQueue{finishErr: finishSentinel}
	rFinishErr, err := New(Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: qFinishErr}, func(context.Context) (Recognizer, error) {
		return &stubRecognizer{blocks: []Block{{Text: "hello"}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res, err := rFinishErr.Recognize(t.Context(), Request{Path: "scan.png"}); !errors.Is(err, finishSentinel) || res.Pages != 1 {
		t.Fatalf("FinishTask error returned %+v, %v", res, err)
	}

	qOK := &stubQueue{}
	rOK, err := New(Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: qOK}, func(context.Context) (Recognizer, error) {
		return &stubRecognizer{blocks: []Block{{Text: "hello"}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// 6. readRecord error in Recognize.
	storeReadSentinel := errors.New("store read failed")
	rStoreErr, err := New(
		Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(&stubStore{Folder: v.Store(), readErr: storeReadSentinel}), Texts: versionsOf(texts), Queue: qOK},
		func(context.Context) (Recognizer, error) { return &stubRecognizer{}, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rStoreErr.Recognize(t.Context(), Request{Path: "scan.png"}); !errors.Is(err, storeReadSentinel) {
		t.Fatalf("readRecord error returned %v", err)
	}

	// 7. Corrupt document open error in Recognize.
	v.File("corrupt.png", "not-a-png")
	if _, err := rOK.Recognize(t.Context(), Request{Path: "corrupt.png"}); err == nil {
		t.Fatal("corrupt image succeeded")
	}

	// 8. EnqueueWork error in Recognize.
	enqueueSentinel := errors.New("queue full")
	rEnqueueErr, err := New(
		Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: &stubQueue{enqueueErr: enqueueSentinel}},
		func(context.Context) (Recognizer, error) { return &stubRecognizer{}, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rEnqueueErr.Recognize(t.Context(), Request{Path: "scan.png"}); !errors.Is(err, enqueueSentinel) {
		t.Fatalf("EnqueueWork error returned %v", err)
	}

	// 9. Recognizer open error in Recognize.
	openSentinel := errors.New("open failed")
	rOpenErr, err := New(
		Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: qOK},
		func(context.Context) (Recognizer, error) { return nil, openSentinel },
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rOpenErr.Recognize(t.Context(), Request{Path: "scan.png"}); !errors.Is(err, openSentinel) {
		t.Fatalf("open error returned %v", err)
	}

	// 10. Turn Release error joins with nil err and preserves non-zero Result (both with and without ReportTask error).
	releaseSentinel := errors.New("turn release failed")
	rReleaseErr, err := New(
		Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: &stubQueue{releaseErr: releaseSentinel}},
		func(context.Context) (Recognizer, error) {
			return &stubRecognizer{blocks: []Block{{Text: "hello"}}}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := rReleaseErr.Recognize(t.Context(), Request{Path: "scan.png"}); !errors.Is(err, releaseSentinel) || res.Pages != 1 {
		t.Fatalf("turn release error returned %+v, %v", res, err)
	}

	rReleaseAndReportErr, err := New(
		Host{Documents: pictures{}, Vaults: func(id domain.VaultID) Files { return v.With(id) }, Store: foldersOf(v.Store()), Texts: versionsOf(texts), Queue: &stubQueue{releaseErr: releaseSentinel, reportErr: reportSentinel, reportErrAt: 3}},
		func(context.Context) (Recognizer, error) {
			return &stubRecognizer{blocks: []Block{{Text: "hello"}}}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := rReleaseAndReportErr.Recognize(t.Context(), Request{Path: "scan.png"}); !errors.Is(err, releaseSentinel) || !errors.Is(err, reportSentinel) || res.Pages != 1 {
		t.Fatalf("turn release + report error returned %+v, %v", res, err)
	}
}

func foldersOf(folder Folder) func(domain.VaultID) Folder {
	return func(domain.VaultID) Folder { return folder }
}

func versionsOf(texts host.TextVersions) func(domain.VaultID) host.TextVersions {
	return func(domain.VaultID) host.TextVersions { return texts }
}

// pictures opens a scan saved as an image as a document of one page.
type pictures struct{}

func (pictures) Open(filePath string, raw []byte) (Document, error) {
	drawing, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", filePath, err)
	}
	return picture{drawing: drawing}, nil
}

type picture struct{ drawing image.Image }

func (picture) GetPageCount() int  { return 1 }
func (picture) HasTextLayer() bool { return false }
func (picture) Close() error       { return nil }

func (p picture) RenderPage(_, _ int) (image.Image, error) { return p.drawing, nil }

package extract

import (
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/tetratelabs/wazero"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

const testPluginID = "md.numen.readers.pdf"

type fakeTurn struct {
	releaseErr error
}

func (t fakeTurn) Release() error { return t.releaseErr }

func (t fakeTurn) WatchWaiting() <-chan struct{} { return nil }

type fakeQueue struct {
	host.Queue
	enqueueErr error
	releaseErr error
	reportErr  error
	finishErr  error
}

func (q *fakeQueue) EnqueueWork(ctx context.Context, work host.Work) (host.Turn, error) {
	if q.enqueueErr != nil {
		return nil, q.enqueueErr
	}
	if q.releaseErr == nil {
		return q.Queue.EnqueueWork(ctx, work)
	}
	return fakeTurn{releaseErr: q.releaseErr}, nil
}

func (q *fakeQueue) ReportTask(ctx context.Context, task host.Task) error {
	if q.reportErr != nil {
		return q.reportErr
	}
	return q.Queue.ReportTask(ctx, task)
}

func (q *fakeQueue) FinishTask(ctx context.Context, id string) error {
	if q.finishErr != nil {
		return q.finishErr
	}
	return q.Queue.FinishTask(ctx, id)
}

type failingStore struct {
	host.StoreWriter
	writeErr error
}

func (s failingStore) Write(ctx context.Context, name string, content []byte) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.StoreWriter.Write(ctx, name, content)
}

type failingTexts struct {
	host.TextVersions
	putErr error
}

func (t failingTexts) PutVersion(ctx context.Context, params host.TextPut) (domain.TextVersion, error) {
	if t.putErr != nil {
		return domain.TextVersion{}, t.putErr
	}
	return t.TextVersions.PutVersion(ctx, params)
}

type slowCancelFiles struct {
	Files
	started    chan struct{}
	startOnce  *sync.Once
	canceled   chan struct{}
	cancelOnce *sync.Once
	release    chan struct{}
}

func (f slowCancelFiles) Read(ctx context.Context, _ string) ([]byte, domain.FileStat, error) {
	f.startOnce.Do(func() { close(f.started) })
	<-ctx.Done()
	f.cancelOnce.Do(func() { close(f.canceled) })
	<-f.release
	return nil, domain.FileStat{}, ctx.Err()
}

type corruptibleFiles struct {
	Files
	firstBuf []byte
	raw      []byte
	calls    int
}

func (f *corruptibleFiles) Read(_ context.Context, name string) ([]byte, domain.FileStat, error) {
	f.calls++
	if f.calls == 1 {
		return f.firstBuf, domain.FileStat{Path: name}, nil
	}
	return f.raw, domain.FileStat{Path: name}, nil
}

func newTestWorkers(t *testing.T) *pdf.Workers {
	t.Helper()
	compilation, err := wazero.NewCompilationCacheWithDir(os.Getenv("NUMEN_PDF_TEST_COMPILATION_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	workers := pdf.NewWorkers(pdf.Options{
		Report:           func(what string, err error) { t.Errorf("%s: %v", what, err) },
		CompilationCache: compilation,
	})
	t.Cleanup(func() {
		if err := errors.Join(workers.Close(), compilation.Close(context.WithoutCancel(t.Context()))); err != nil {
			t.Error(err)
		}
	})
	return workers
}

func readPDF(t *testing.T, name string) []byte {
	t.Helper()
	scan := "%PDF-1.4\n" +
		"1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n" +
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n" +
		"3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj\n" +
		"4 0 obj << /Length 44 >>\nstream\nBT /F1 12 Tf 20 100 Td (" + name + ") Tj ET\nendstream\nendobj\n" +
		"5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n" +
		"trailer << /Root 1 0 R /Size 6 >>\n%%EOF\n"
	return []byte(scan)
}

func TestCloseWaitsForRunningWorkAndRerunsFinishedFiles(t *testing.T) {
	firstRaw := readPDF(t, "Alpha")
	secondRaw := readPDF(t, "Beta")
	vault := hosttest.CreateOnDisk(t, "vault", map[string]string{
		"a.pdf": string(firstRaw),
		"b.pdf": string(secondRaw),
	})
	texts := hosttest.NewTexts("vault", testPluginID)
	texts.WriteFile("a.pdf", firstRaw)
	texts.WriteFile("b.pdf", secondRaw)
	texts.SetStore(testPluginID, vault.Store())
	workers := newTestWorkers(t)
	writer := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		hosttest.NewQueue(), workers, func(what string, err error) {
			t.Errorf("unexpected report %s: %v", what, err)
		},
	)
	t.Cleanup(writer.Close)

	if err := writer.CreateText(t.Context(), Request{Path: "a.pdf"}); err != nil {
		t.Fatal(err)
	}
	writer.Wait()

	// Running on a second file with different content does not reuse the first file's layer.
	if err := writer.CreateText(t.Context(), Request{Path: "b.pdf"}); err != nil {
		t.Fatal(err)
	}
	writer.Wait()
	listingB, err := texts.ListVersions(t.Context(), "b.pdf")
	if err != nil || len(listingB.Versions) != 1 {
		t.Fatalf("b.pdf versions = %v, %v; want 1", listingB.Versions, err)
	}
	textB, err := host.ReadWholeText(t.Context(), texts, listingB.Versions[0])
	if err != nil || !strings.Contains(textB.Text, "Beta") {
		t.Errorf("b.pdf text = %q, %v; want Beta", textB.Text, err)
	}

	// Running again on a.pdf after it finished updates its version.
	vault.WriteFile("a.pdf", string(secondRaw))
	texts.WriteFile("a.pdf", secondRaw)
	if err := writer.CreateText(t.Context(), Request{Path: "a.pdf"}); err != nil {
		t.Fatal(err)
	}
	writer.Wait()
	listingA, err := texts.ListVersions(t.Context(), "a.pdf")
	if err != nil || len(listingA.Versions) == 0 {
		t.Fatalf("a.pdf versions = %v, %v", listingA.Versions, err)
	}
	textA, err := host.ReadWholeText(t.Context(), texts, listingA.Versions[0])
	if err != nil || !strings.Contains(textA.Text, "Beta") {
		t.Errorf("updated a.pdf text = %q, %v; want Beta", textA.Text, err)
	}

	// Close waits for in-flight work to finish before returning.
	slow := slowCancelFiles{
		Files: nil, started: make(chan struct{}), startOnce: new(sync.Once),
		canceled: make(chan struct{}), cancelOnce: new(sync.Once), release: make(chan struct{}),
	}
	slowWriter := NewWriter(
		func(id domain.VaultID) Files {
			files := slow
			files.Files = vault.With(id)
			return files
		},
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		hosttest.NewQueue(), workers, func(string, error) {},
	)
	if err := slowWriter.CreateText(t.Context(), Request{Vault: "vault", Path: "a.pdf"}); err != nil {
		t.Fatal(err)
	}
	<-slow.started
	closed := make(chan struct{})
	go func() {
		slowWriter.Close()
		close(closed)
	}()
	<-slow.canceled
	select {
	case <-closed:
		t.Error("Close returned before the running read was released")
	default:
	}
	close(slow.release)
	<-closed

	// CreateText fails when no vault is shown.
	noVault := hosttest.New("vault")
	noVault.Hide()
	noVaultWriter := NewWriter(
		func(id domain.VaultID) Files { return noVault.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return noVault.Store().With(id) },
		hosttest.NewQueue(), workers, func(string, error) {},
	)
	if err := noVaultWriter.CreateText(t.Context(), Request{Vault: "", Path: "a.pdf"}); !errors.Is(err, host.ErrNoVault) {
		t.Errorf("CreateText with no shown vault = %v, want ErrNoVault", err)
	}
}

func TestLayerWriteReportsAndWrapsEveryFailure(t *testing.T) {
	raw := readPDF(t, "Alpha")
	vault := hosttest.CreateOnDisk(t, "vault", map[string]string{"a.pdf": string(raw)})
	texts := hosttest.NewTexts("vault", testPluginID)
	texts.WriteFile("a.pdf", raw)
	texts.SetStore(testPluginID, vault.Store())
	workers := newTestWorkers(t)
	task := host.Task{ID: "text:a.pdf", Title: "Reading a.pdf", Detail: "a.pdf"}

	// 1. EnqueueWork failure wraps cause.
	errEnqueue := errors.New("enqueue failed")
	qEnqueue := &fakeQueue{Queue: hosttest.NewQueue(), enqueueErr: errEnqueue}
	wEnqueue := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		qEnqueue, workers, func(string, error) {},
	)
	if err := wEnqueue.putLayer(t.Context(), Request{Vault: "vault", Path: "a.pdf"}, task); !errors.Is(err, errEnqueue) {
		t.Errorf("putLayer enqueue error = %v, want %v", err, errEnqueue)
	}

	// 2. Missing file wraps ErrNotExist and leaves failed task in queue via CreateText.
	qMissing := hosttest.NewQueue()
	wMissing := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		qMissing, workers, func(string, error) {},
	)
	if err := wMissing.putLayer(t.Context(), Request{Vault: "vault", Path: "missing.pdf"}, task); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("putLayer missing error = %v, want ErrNotExist", err)
	}
	if err := wMissing.CreateText(t.Context(), Request{Vault: "vault", Path: "missing.pdf"}); err != nil {
		t.Fatal(err)
	}
	wMissing.Wait()
	// The putLayer call above reported its own task, text:a.pdf, so the queue holds that one too.
	tasks := qMissing.ListTasks()
	failed := slices.IndexFunc(tasks, func(one host.Task) bool { return one.Detail == "missing.pdf" })
	if failed < 0 || !strings.Contains(tasks[failed].FailureReason, "read missing.pdf") {
		t.Errorf("tasks after missing file = %+v, want a failed task of missing.pdf", tasks)
	}

	// 3. Cached layer reuses book and fails in getSegments when cached raw bytes are corrupted.
	corrupt := &corruptibleFiles{Files: vault, firstBuf: bytes.Clone(raw), raw: bytes.Clone(raw)}
	qCorrupt := hosttest.NewQueue()
	wCorrupt := NewWriter(
		func(domain.VaultID) Files { return corrupt },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		qCorrupt, workers, func(string, error) {},
	)
	if err := wCorrupt.CreateText(t.Context(), Request{Vault: "vault", Path: "a.pdf"}); err != nil {
		t.Fatal(err)
	}
	wCorrupt.Wait()
	corrupt.firstBuf[0] = 0
	if err := wCorrupt.putLayer(t.Context(), Request{Vault: "vault", Path: "a.pdf"}, task); !errors.Is(err, pdf.ErrNotPDF) {
		t.Errorf("putLayer with corrupted cached raw = %v, want ErrNotPDF", err)
	}

	// 4. Turn.Release failure propagates from putLayer.
	errRelease := errors.New("release failed")
	qRelease := &fakeQueue{Queue: hosttest.NewQueue(), releaseErr: errRelease}
	wRelease := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		qRelease, workers, func(string, error) {},
	)
	if err := wRelease.putLayer(t.Context(), Request{Vault: "vault", Path: "a.pdf"}, task); !errors.Is(err, errRelease) {
		t.Errorf("putLayer release error = %v, want %v", err, errRelease)
	}

	// 5. ReportTask failure aborts putLayer and reports failure in writeLayer.
	errReport := errors.New("report task failed")
	var reportedWhat string
	var reportedErr error
	qReport := &fakeQueue{Queue: hosttest.NewQueue(), reportErr: errReport}
	noReportTexts := hosttest.NewTexts("vault", testPluginID)
	wReport := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions { return noReportTexts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		qReport, workers, func(what string, err error) {
			reportedWhat, reportedErr = what, err
		},
	)
	if err := wReport.CreateText(t.Context(), Request{Vault: "vault", Path: "a.pdf"}); err != nil {
		t.Fatal(err)
	}
	wReport.Wait()
	if !errors.Is(reportedErr, errReport) || !strings.HasPrefix(reportedWhat, "show the task ") {
		t.Errorf("reported (%q, %v), want show the task error %v", reportedWhat, reportedErr, errReport)
	}
	if listing, err := noReportTexts.ListVersions(t.Context(), "a.pdf"); err != nil || len(listing.Versions) != 0 {
		t.Errorf("versions when ReportTask fails = %v, %v; want none", listing.Versions, err)
	}

	// 6. Store write failure aborts putLayer.
	errStore := errors.New("store write failed")
	wStore := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter {
			return failingStore{StoreWriter: vault.Store().With(id), writeErr: errStore}
		},
		hosttest.NewQueue(), workers, func(string, error) {},
	)
	if err := wStore.putLayer(t.Context(), Request{Vault: "vault", Path: "a.pdf"}, task); !errors.Is(err, errStore) {
		t.Errorf("putLayer store error = %v, want %v", err, errStore)
	}

	// 7. PutVersion failure propagates from putLayer.
	errPut := errors.New("put version failed")
	wPut := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions {
			return failingTexts{TextVersions: texts.With(id), putErr: errPut}
		},
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		hosttest.NewQueue(), workers, func(string, error) {},
	)
	if err := wPut.putLayer(t.Context(), Request{Vault: "vault", Path: "a.pdf"}, task); !errors.Is(err, errPut) {
		t.Errorf("putLayer PutVersion error = %v, want %v", err, errPut)
	}

	// 8. FinishTask failure is reported in writeLayer.
	errFinish := errors.New("finish task failed")
	var finishWhat string
	var finishGot error
	qFinish := &fakeQueue{Queue: hosttest.NewQueue(), finishErr: errFinish}
	wFinish := NewWriter(
		func(id domain.VaultID) Files { return vault.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return vault.Store().With(id) },
		qFinish, workers, func(what string, err error) {
			finishWhat, finishGot = what, err
		},
	)
	if err := wFinish.CreateText(t.Context(), Request{Vault: "vault", Path: "a.pdf"}); err != nil {
		t.Fatal(err)
	}
	wFinish.Wait()
	if !errors.Is(finishGot, errFinish) || !strings.HasPrefix(finishWhat, "finish the task ") {
		t.Errorf("finish reported (%q, %v), want finish the task error %v", finishWhat, finishGot, errFinish)
	}
}

// Package recognition reads the text of a scanned file with the recogniser, keeps it in the plugin's
// folder of the vault's store, and reports it to the host as a positioned text version.
package recognition

import (
	"context"
	"errors"
	"fmt"
	"image"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/document"
)

// VersionKey is the key the plugin keeps the recognition of a file under: one recognition a file.
const VersionKey = "recognition"

// batch is the number of pages read between two writes of the progress record.
const batch = 8

// Model describes active OCR models.
type Model struct {
	Layout     string
	Recogniser string
	DPI        int
	From       string
}

// ByteRange represents a byte offset range.
type ByteRange struct {
	Start int
	End   int
}

// Box is one box of a region: the rectangle it covers on the page, and the run of bytes it produced.
type Box struct {
	Rect  image.Rectangle
	Range ByteRange
}

// Block is one region of a page, written out.
type Block struct {
	Label     string
	Text      string
	IsHeading bool
	Depth     int
	Boxes     []Box
}

// Recogniser reads the blocks of one page drawn as an image.
type Recogniser interface {
	GetRecognitionModel() Model
	Recognise(ctx context.Context, page image.Image) ([]Block, error)
}

// Host is what a recognition works through: the vault the file is read from, the plugin's folder
// the text is stored in, the text versions it is reported to, and the queue it runs in.
type Host struct {
	Vault host.Vault
	Store host.Store
	Texts host.Texts
	Queue host.Queue
}

// Recognition reads files with the recogniser it opens.
type Recognition struct {
	host Host
	open func(ctx context.Context) (Recogniser, error)
}

// New is the recognition through the host, with what opens the recogniser when a page is first read.
func New(h Host, open func(ctx context.Context) (Recogniser, error)) (*Recognition, error) {
	if h.Vault == nil || h.Store == nil || h.Texts == nil || h.Queue == nil || open == nil {
		return nil, errors.New("a recognition needs the vault, the store, the texts, the queue and a recogniser")
	}
	return &Recognition{host: h, open: open}, nil
}

// Request is one file to recognise. Vault is the vault that contains it, and is
// empty for the vault the host currently shows. IsUserStarted is set when the
// user requested it.
type Request struct {
	Vault         domain.VaultID
	Path          string
	IsUserStarted bool
}

// Result is the outcome of one recognition. IsSkipped is set for a file that
// the user did not request and that has embedded text or was already
// recognised. IsEmpty is set for a file from which the recogniser extracted no text.
type Result struct {
	Pages     int
	IsSkipped bool
	IsEmpty   bool
}

// Recognise reads the file and stores its text version. A run that the user
// did not request skips a file whose pages have embedded text, and a file
// already recognised in its current state. A failure is shown as a task until
// the file is recognised again.
func (r *Recognition) Recognise(ctx context.Context, req Request) (Result, error) {
	on := reach{vault: r.host.Vault, store: r.host.Store, texts: r.host.Texts}
	if req.Vault != "" {
		on = reach{vault: on.vault.With(req.Vault), store: on.store.With(req.Vault), texts: on.texts.With(req.Vault)}
	}
	task := host.Task{
		ID: "recognise:" + req.Path, Title: "Recognising " + path.Base(req.Path), About: req.Path,
		Done: 0, Total: 0, Unit: host.UnitThings, Error: "", IsUserStarted: req.IsUserStarted,
	}
	res, err := r.recognise(ctx, on, req, &task)
	if err != nil && ctx.Err() == nil {
		task.Error = err.Error()
		if reportErr := r.host.Queue.ReportTask(context.WithoutCancel(ctx), task); reportErr != nil {
			return res, errors.Join(err, reportErr)
		}
		return res, err
	}
	if finishErr := r.host.Queue.FinishTask(context.WithoutCancel(ctx), task.ID); finishErr != nil && err == nil {
		return res, finishErr
	}
	return res, err
}

// reach is the vault a recognition reads, its store and its text versions, all of one vault.
type reach struct {
	vault host.Vault
	store host.Store
	texts host.Texts
}

func (r *Recognition) recognise(ctx context.Context, on reach, req Request, task *host.Task) (res Result, err error) {
	raw, _, err := on.vault.Read(ctx, req.Path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", req.Path, err)
	}
	inputHash := domain.HashContent(raw)
	if !req.IsUserStarted {
		isRecognised, err := hasRecognition(ctx, on.texts, req.Path, inputHash)
		if err != nil || isRecognised {
			return Result{Pages: 0, IsSkipped: isRecognised, IsEmpty: false}, err
		}
	}
	rec, err := readRecord(ctx, on.store, req.Path)
	if err != nil {
		return Result{}, err
	}
	doc, err := document.Open(req.Path, raw)
	if err != nil {
		return Result{}, err
	}
	defer func() { err = errors.Join(err, doc.Close()) }()
	if !req.IsUserStarted && doc.HasTextLayer() {
		return Result{Pages: doc.GetPageCount(), IsSkipped: true, IsEmpty: false}, nil
	}

	turn, err := r.host.Queue.EnqueueWork(ctx, host.Work{IsUserStarted: req.IsUserStarted, OnWaiting: nil})
	if err != nil {
		return Result{}, fmt.Errorf("wait for a turn to recognise %s: %w", req.Path, err)
	}
	defer turn.Release()
	recogniser, err := r.open(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("open the recogniser: %w", err)
	}
	model := recogniser.GetRecognitionModel()
	recipe := fmt.Sprintf("%s|%s|%s|%d", model.From, model.Layout, model.Recogniser, model.DPI)
	if !rec.isFor(inputHash, recipe) {
		rec = &record{Path: req.Path, InputHash: inputHash, Recipe: recipe, PageCount: doc.GetPageCount(), Pages: nil}
	}

	if err := r.readPages(ctx, pageReading{doc: doc, recogniser: recogniser, dpi: model.DPI, store: on.store, task: task}, rec); err != nil {
		return Result{}, err
	}
	return putVersion(ctx, on, rec)
}

// pageReading is what the pages of one file are read with.
type pageReading struct {
	doc        document.Document
	recogniser Recogniser
	dpi        int
	store      host.Store
	task       *host.Task
}

// readPages reads the pages the record does not hold yet, keeping them a batch at a time and showing
// how far it came in the task.
func (r *Recognition) readPages(ctx context.Context, on pageReading, rec *record) error {
	on.task.Total, on.task.Done = int64(rec.PageCount), int64(len(rec.Pages))
	if err := r.host.Queue.ReportTask(ctx, *on.task); err != nil {
		return err
	}
	for index := len(rec.Pages); index < rec.PageCount; index++ {
		page, err := on.doc.RenderPage(index, on.dpi)
		if err != nil {
			return err
		}
		blocks, err := on.recogniser.Recognise(ctx, page)
		if err != nil {
			return fmt.Errorf("recognise page %d of %s: %w", index+1, rec.Path, err)
		}
		rec.Pages = append(rec.Pages, newPage(page.Bounds(), blocks))
		if (index+1)%batch == 0 {
			if err := writeRecord(ctx, on.store, rec); err != nil {
				return err
			}
		}
		on.task.Done = int64(index + 1)
		if err := r.host.Queue.ReportTask(ctx, *on.task); err != nil {
			return err
		}
	}
	return nil
}

// putVersion stores the text of a complete record as the file's version,
// reports it, and deletes the record. A record with no text is deleted and
// reported as no version.
func putVersion(ctx context.Context, on reach, rec *record) (Result, error) {
	store := on.store
	text, err := rec.getText()
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(text.Text) == "" {
		if err := store.Delete(ctx, getRecordName(rec.Path)); err != nil {
			return Result{}, err
		}
		return Result{Pages: rec.PageCount, IsSkipped: false, IsEmpty: true}, nil
	}
	file := getVersionName(rec.Path)
	if err := host.WriteText(ctx, store, file, text.GetSegments()); err != nil {
		return Result{}, err
	}
	_, err = on.texts.PutVersion(ctx, host.TextPut{
		Path: rec.Path, Key: VersionKey, Source: "", Kind: textKind, SourceHash: rec.InputHash, File: file,
	})
	if err != nil {
		return Result{}, fmt.Errorf("report the recognition of %s: %w", rec.Path, err)
	}
	if err := store.Delete(ctx, getRecordName(rec.Path)); err != nil {
		return Result{}, err
	}
	return Result{Pages: rec.PageCount, IsSkipped: false, IsEmpty: false}, nil
}

// hasRecognition reports whether the file has a version of the plugin made from the content hash.
func hasRecognition(ctx context.Context, texts host.Texts, path, inputHash string) (bool, error) {
	listing, err := texts.ListVersions(ctx, path)
	if err != nil {
		return false, fmt.Errorf("list the versions of %s: %w", path, err)
	}
	for _, version := range listing.Versions {
		if version.Key == VersionKey && version.Kind == textKind && version.IsFromFile() && version.SourceHash == inputHash {
			return true, nil
		}
	}
	return false, nil
}

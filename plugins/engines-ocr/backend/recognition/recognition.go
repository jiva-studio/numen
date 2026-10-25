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

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
)

// VersionKey is the key the plugin keeps the recognition of a file under: one recognition a file.
const VersionKey = "recognition"

// batch is the number of pages read between two writes of the record.
const batch = 8

// Recogniser reads the blocks of one page drawn as an image.
type Recogniser interface {
	GetRecognitionModel() engine.Model
	Recognise(ctx context.Context, page image.Image) ([]engine.Block, error)
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

// Request is one file to recognise. Vault is the vault it is in, and empty for the one the host
// shows. IsUserStarted is set where a person asked for it.
type Request struct {
	Vault         domain.VaultID
	Path          string
	IsUserStarted bool
}

// Result is what one recognition did. IsSkipped is a file nobody asked for that carries text of its
// own or was recognised already, and IsEmpty is a file the recogniser read nothing on.
type Result struct {
	Pages     int
	IsSkipped bool
	IsEmpty   bool
}

// Recognise reads the file and puts its text version. A run nobody asked for skips a file whose
// pages carry text of their own, and one already recognised as it is now. A failure is shown as a
// task until the file is recognised again.
func (r *Recognition) Recognise(ctx context.Context, req Request) (Result, error) {
	vault, store := r.host.Vault, r.host.Store
	if req.Vault != "" {
		vault, store = vault.With(req.Vault), store.With(req.Vault)
	}
	task := host.Task{
		ID: "recognise:" + req.Path, Title: "Recognising " + path.Base(req.Path), About: req.Path,
		Unit: host.UnitThings, IsUserStarted: req.IsUserStarted,
	}
	res, err := r.recognise(ctx, vault, store, req, &task)
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

func (r *Recognition) recognise(ctx context.Context, vault host.Vault, store host.Store, req Request, task *host.Task) (Result, error) {
	raw, _, err := vault.Read(ctx, req.Path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", req.Path, err)
	}
	inputHash := domain.HashContent(raw)
	rec, err := readRecord(ctx, store, req.Path)
	if err != nil {
		return Result{}, err
	}
	if !req.IsUserStarted && rec.isCompleteFor(inputHash) {
		return Result{Pages: rec.PageCount, IsSkipped: true}, nil
	}
	doc, err := document.Open(req.Path, raw)
	if err != nil {
		return Result{}, err
	}
	defer doc.Close()
	if !req.IsUserStarted && doc.HasTextLayer() {
		return Result{Pages: doc.GetPageCount(), IsSkipped: true}, nil
	}

	turn, err := r.host.Queue.EnqueueWork(ctx, host.Work{IsUserStarted: req.IsUserStarted})
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
		rec = &record{Path: req.Path, InputHash: inputHash, Recipe: recipe, PageCount: doc.GetPageCount()}
	}

	if err := r.readPages(ctx, pageReading{doc: doc, recogniser: recogniser, dpi: model.DPI, store: store, task: task}, rec); err != nil {
		return Result{}, err
	}
	rec.IsComplete = true
	return r.putVersion(ctx, store, rec)
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

// putVersion keeps a complete record and reports it as the file's version. A record that says
// no text is deleted and reported as no version.
func (r *Recognition) putVersion(ctx context.Context, store host.Store, rec *record) (Result, error) {
	text := rec.getText()
	if strings.TrimSpace(text.Text) == "" {
		if err := store.Delete(ctx, getRecordName(rec.Path)); err != nil {
			return Result{}, err
		}
		return Result{Pages: rec.PageCount, IsEmpty: true}, nil
	}
	if err := writeRecord(ctx, store, rec); err != nil {
		return Result{}, err
	}
	_, err := r.host.Texts.PutVersion(ctx, host.TextPut{
		Path: rec.Path, Key: VersionKey, Format: domain.TextFormatPositioned,
		InputHash: rec.InputHash, Hash: domain.HashContent([]byte(text.Text)),
	})
	if err != nil {
		return Result{}, fmt.Errorf("report the recognition of %s: %w", rec.Path, err)
	}
	return Result{Pages: rec.PageCount}, nil
}

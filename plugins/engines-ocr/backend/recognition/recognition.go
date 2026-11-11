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
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/document"
	"github.com/tetratelabs/wazero"
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
	Recognize(ctx context.Context, page image.Image) ([]Block, error)
}

// Files is what a recognition reads of a vault: its files and its name.
type Files interface {
	host.FileReader
	GetVault(ctx context.Context) (domain.Vault, error)
}

// Folder is the part of the plugin's folder in the store of a vault that a recognition keeps its progress and
// its text in.
type Folder interface {
	host.StoreWriter
	Read(ctx context.Context, name string) ([]byte, error)
}

// Host is what a recognition works through: the vaults the file is read from, the plugin's folder
// the text is stored in, the text versions it is reported to, and the queue it runs in.
type Host struct {
	Vaults func(domain.VaultID) Files
	Store  func(domain.VaultID) Folder
	Texts  func(domain.VaultID) host.TextVersions
	Queue  host.Queue
	Report func(what string, err error)
}

// ErrAlreadyRecognising is returned for a file another run is recognizing.
var ErrAlreadyRecognising = errors.New("the file is already being recognized")

// Recognition reads files with the recogniser it opens. One run at a time locks a file of a vault.
type Recognition struct {
	host      Host
	open      func(ctx context.Context) (Recogniser, error)
	documents *document.Reader

	mu      sync.Mutex
	running map[string]bool
}

// New returns the recognition through the host, with the function that opens the recogniser when a
// page is first read.
func New(h Host, open func(ctx context.Context) (Recogniser, error)) (*Recognition, error) {
	if h.Vaults == nil || h.Store == nil || h.Texts == nil || h.Queue == nil || open == nil {
		return nil, errors.New("a recognition needs the vault, the store, the texts, the queue and a recogniser")
	}
	return &Recognition{
		host: h, open: open, documents: document.NewReader(),
		mu: sync.Mutex{}, running: map[string]bool{},
	}, nil
}

// SetCompilationCache makes the reader of documents compile PDFium through cache.
func (r *Recognition) SetCompilationCache(cache wazero.CompilationCache) {
	r.documents.SetCompilationCache(cache)
}

// Request is one file to recognize. Vault is the vault that contains it, and is
// empty for the vault the host currently shows.
type Request struct {
	Vault domain.VaultID
	Path  string
}

// Result is the outcome of one recognition. IsEmpty is set for a file from which the recogniser
// extracted no text.
type Result struct {
	Pages   int
	IsEmpty bool
}

// Recognize reads the file and stores its text version. A failure is shown as a task until
// the file is recognized again, and so is a file from which nothing is read. A failure of a missing
// file ends the task. For a file another run has locked, it returns ErrAlreadyRecognising.
func (r *Recognition) Recognize(ctx context.Context, req Request) (Result, error) {
	req, key, err := r.claim(ctx, req)
	if err != nil {
		return Result{}, err
	}
	release := sync.OnceFunc(func() { r.release(key) })
	defer release()
	return r.runRecognition(ctx, req, release)
}

// Start claims the file and recognizes it in the background under ctx, and returns false when
// another run has locked the file. An empty vault is the vault the host shows when Start is called,
// and the run reads and writes only that vault. The channel is closed when the run ends.
func (r *Recognition) Start(ctx context.Context, req Request) (<-chan struct{}, bool, error) {
	req, key, err := r.claim(ctx, req)
	if errors.Is(err, ErrAlreadyRecognising) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if err := r.host.Queue.ReportTask(ctx, newTask(req)); err != nil {
		r.release(key)
		return nil, false, err
	}
	runEnd := make(chan struct{})
	go func() {
		defer close(runEnd)
		release := sync.OnceFunc(func() { r.release(key) })
		defer release()
		// runRecognition reports a failure as the error of the task.
		_, err := r.runRecognition(ctx, req, release)
		// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_NEGATION,INVERT_LOGICAL reason="Stderr logging of background recognition failure does not affect recognition state."
		if err != nil && ctx.Err() == nil {
			r.reportError("recognize "+req.Path, err)
		}
	}()
	return runEnd, true, nil
}

func (r *Recognition) reportError(what string, err error) {
	if r.host.Report != nil && err != nil {
		r.host.Report(what, err)
	}
}

func (r *Recognition) claim(ctx context.Context, req Request) (Request, string, error) {
	vault, err := r.GetVaultID(ctx, req.Vault)
	if err != nil {
		return Request{}, "", err
	}
	req.Vault = vault
	key := getRunKey(req.Vault, req.Path)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running[key] {
		return Request{}, "", fmt.Errorf("%s: %w", req.Path, ErrAlreadyRecognising)
	}
	r.running[key] = true
	return req, key, nil
}

// IsRunning reports whether a run has locked the file of the vault. An empty vault ID means the vault
// the host shows.
func (r *Recognition) IsRunning(ctx context.Context, vault domain.VaultID, filePath string) (bool, error) {
	vault, err := r.GetVaultID(ctx, vault)
	if err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running[getRunKey(vault, filePath)], nil
}

// GetVaultID returns the vault, or the vault the host shows where it is empty.
func (r *Recognition) GetVaultID(ctx context.Context, vault domain.VaultID) (domain.VaultID, error) {
	if vault != "" {
		return vault, nil
	}
	shown, err := r.host.Vaults("").GetVault(ctx)
	if err != nil {
		return "", fmt.Errorf("get the shown vault: %w", err)
	}
	return shown.ID, nil
}

func getRunKey(vault domain.VaultID, filePath string) string {
	return string(vault) + "\x00" + filePath
}

// FindLiveDerivedVersion returns the live version of the file of the vault when its source chain
// contains the version the named plugin stores under VersionKey: that version itself, or a version
// made from it. A recognition writes VersionKey only once it is complete. The host keeps live only a
// version whose chain matches the file as it is now. An empty vault ID means the vault the host shows.
func (r *Recognition) FindLiveDerivedVersion(ctx context.Context, vault domain.VaultID, filePath, pluginID string) (domain.TextVersion, bool, error) {
	listing, err := r.host.Texts(vault).ListVersions(ctx, filePath)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	byID := make(map[domain.TextVersionID]domain.TextVersion, len(listing.Versions))
	for _, version := range listing.Versions {
		byID[version.GetID()] = version
	}
	live, isLive := byID[listing.Live]
	seenIDs := map[domain.TextVersionID]bool{}
	for version, isFound := live, isLive; isFound && !seenIDs[version.GetID()]; version, isFound = byID[version.Source] {
		if version.Key == VersionKey && version.PluginID == pluginID {
			return live, true, nil
		}
		seenIDs[version.GetID()] = true
	}
	return domain.TextVersion{}, false, nil
}

func (r *Recognition) release(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, key)
}

func newTask(req Request) host.Task {
	return host.Task{
		ID: GetTaskID(req.Vault, req.Path), Title: "Recognizing " + path.Base(req.Path), Detail: req.Path,
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "", IsUserStarted: true,
	}
}

// runRecognition calls release before it reports how the task ended, so that whoever sees the task
// end finds the file no longer running.
func (r *Recognition) runRecognition(ctx context.Context, req Request, release func()) (Result, error) {
	on := reach{vault: r.host.Vaults(req.Vault), store: r.host.Store(req.Vault), texts: r.host.Texts(req.Vault)}
	task := newTask(req)
	res, err := r.recognize(ctx, on, req, &task)
	release()
	if err != nil && ctx.Err() == nil && !isMissing(ctx, on.vault, req.Path) {
		task.FailureReason = err.Error()
		if reportErr := r.host.Queue.ReportTask(context.WithoutCancel(ctx), task); reportErr != nil {
			return res, errors.Join(err, reportErr)
		}
		return res, err
	}
	if err == nil && res.IsEmpty {
		task.FailureReason = fmt.Sprintf("%s says nothing that could be read, and nothing was kept", req.Path)
		return res, r.host.Queue.ReportTask(context.WithoutCancel(ctx), task)
	}
	if finishErr := r.host.Queue.FinishTask(context.WithoutCancel(ctx), task.ID); err == nil {
		return res, finishErr
	}
	return res, err
}

// GetTaskID returns the id of the recognition of a file of a vault in the list of tasks.
func GetTaskID(vault domain.VaultID, filePath string) string {
	return "recognize:" + string(vault) + ":" + filePath
}

// StatFile returns the metadata of a file of a vault. An empty vault ID means the vault the host shows.
func (r *Recognition) StatFile(ctx context.Context, vault domain.VaultID, filePath string) (domain.FileStat, error) {
	return r.host.Vaults(vault).StatFile(ctx, filePath)
}

func isMissing(ctx context.Context, vault host.FileReader, filePath string) bool {
	_, err := vault.StatFile(ctx, filePath)
	return errors.Is(err, host.ErrNotExist)
}

// reach is the vault a recognition reads, its store and its text versions, all of one vault.
type reach struct {
	vault host.FileReader
	store Folder
	texts host.TextVersions
}

func (r *Recognition) recognize(ctx context.Context, on reach, req Request, task *host.Task) (res Result, err error) {
	raw, _, err := on.vault.Read(ctx, req.Path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", req.Path, err)
	}
	inputHash := domain.HashContent(raw)
	rec, err := readRecord(ctx, on.store, req.Path)
	if err != nil {
		return Result{}, err
	}
	turn, err := r.host.Queue.EnqueueWork(ctx, host.Work{IsUserStarted: true, OnWaiting: func() {
		queuedTask := *task
		if rec != nil {
			queuedTask.CompletedCount, queuedTask.TotalCount = int64(len(rec.Pages)), int64(rec.PageCount)
		}
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="Stderr logging of waiting task failure does not affect recognition state."
		if err := r.host.Queue.ReportTask(ctx, queuedTask); err != nil {
			r.reportError("show the task "+queuedTask.ID, err)
		}
	}})
	if err != nil {
		return Result{}, fmt.Errorf("wait for a turn to recognize %s: %w", req.Path, err)
	}
	defer func() { err = errors.Join(err, turn.Release()) }()
	doc, err := r.documents.Open(req.Path, raw)
	if err != nil {
		return Result{}, err
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Document implementations opened by Reader return nil on Close."
	defer func() { err = errors.Join(err, doc.Close()) }()
	recogniser, err := r.open(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("open the recogniser: %w", err)
	}
	model := recogniser.GetRecognitionModel()
	rec = keepOrRestart(rec, req.Path, inputHash, model, doc.GetPageCount())

	if err := r.readPages(ctx, pageReading{doc: doc, recogniser: recogniser, dpi: model.DPI, store: on.store, task: task}, rec); err != nil {
		return Result{}, err
	}
	return putVersion(ctx, on, rec)
}

func keepOrRestart(rec *record, path, inputHash string, model Model, pageCount int) *record {
	recipe := fmt.Sprintf("%s|%s|%s|%d", model.From, model.Layout, model.Recogniser, model.DPI)
	if rec.isFor(inputHash, recipe) {
		return rec
	}
	return &record{Path: path, InputHash: inputHash, Recipe: recipe, PageCount: pageCount, Pages: nil}
}

// pageReading is what the pages of one file are read with.
type pageReading struct {
	doc        document.Document
	recogniser Recogniser
	dpi        int
	store      Folder
	task       *host.Task
}

func (r *Recognition) readPages(ctx context.Context, on pageReading, rec *record) error {
	on.task.TotalCount, on.task.CompletedCount = int64(rec.PageCount), int64(len(rec.Pages))
	if err := r.host.Queue.ReportTask(ctx, *on.task); err != nil {
		return err
	}
	for index := len(rec.Pages); index < rec.PageCount; index++ {
		page, err := on.doc.RenderPage(index, on.dpi)
		if err != nil {
			return err
		}
		blocks, err := on.recogniser.Recognize(ctx, page)
		if err != nil {
			return fmt.Errorf("recognize page %d of %s: %w", index+1, rec.Path, err)
		}
		rec.Pages = append(rec.Pages, newPage(page.Bounds(), blocks))
		if (index+1)%batch == 0 {
			if err := writeRecord(ctx, on.store, rec); err != nil {
				return err
			}
		}
		on.task.CompletedCount = int64(index + 1)
		if err := r.host.Queue.ReportTask(ctx, *on.task); err != nil {
			return err
		}
	}
	return nil
}

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
		return Result{Pages: rec.PageCount, IsEmpty: true}, nil
	}
	file := getVersionName(rec.Path)
	if err := host.WriteText(ctx, store, file, text.GetSegments()); err != nil {
		return Result{}, err
	}
	_, err = on.texts.PutVersion(ctx, host.TextPut{
		Path: rec.Path, Key: VersionKey, Source: "", Kind: textKind, InputHash: rec.InputHash, File: file,
	})
	if err != nil {
		return Result{}, fmt.Errorf("report the recognition of %s: %w", rec.Path, err)
	}
	if err := store.Delete(ctx, getRecordName(rec.Path)); err != nil {
		return Result{}, err
	}
	return Result{Pages: rec.PageCount, IsEmpty: false}, nil
}

package proofread

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/settings"
)

var (
	// errNoText is a file no live version gives words to.
	errNoText = errors.New("the file has no text to proofread")
	// errFormat is a version of a kind the plugin does not proofread.
	errFormat = errors.New("the plugin does not proofread this text kind")
)

// Request is one text to proofread. Source is the version read, and empty for the live one.
type Request struct {
	Vault  domain.VaultID
	Path   string
	Source domain.TextVersionID
}

// Result is what one proofreading did: the lines it read, the lines it put right, and the batches
// whose reply was no answer.
type Result struct {
	Lines       int
	Corrected   int
	Uncorrected int
	Version     domain.TextVersion
}

// Proofreading puts the lines of a text version right with the model and reports the corrected text
// as a version made from the one it read.
type Proofreading struct {
	host      Host
	kinds     map[domain.TextKind]bool
	ask       AskModel
	getConfig func() settings.Config
}

// New returns a proofreading of the kinds given, which asks ask and reads its sizes from getConfig on
// every run.
func New(h Host, kinds map[domain.TextKind]bool, ask AskModel, getConfig func() settings.Config) *Proofreading {
	return &Proofreading{host: h, kinds: kinds, ask: ask, getConfig: getConfig}
}

// Proofread reads the version, puts its lines right and puts the version made from it. A failure
// is shown as a task until the file is proofread again.
func (p *Proofreading) Proofread(ctx context.Context, req Request) (Result, error) {
	task := host.Task{
		ID: "proofread:" + req.Path, Title: "Proofreading " + path.Base(req.Path), Detail: req.Path,
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "", IsUserStarted: true,
	}
	res, err := p.proofread(ctx, req, &task)
	if err != nil && ctx.Err() == nil {
		task.FailureReason = err.Error()
		if reportErr := p.host.Queue.ReportTask(context.WithoutCancel(ctx), task); reportErr != nil {
			return res, errors.Join(err, reportErr)
		}
		return res, err
	}
	if finishErr := p.host.Queue.FinishTask(context.WithoutCancel(ctx), task.ID); finishErr != nil && err == nil {
		return res, finishErr
	}
	return res, err
}

func (p *Proofreading) proofread(ctx context.Context, req Request, task *host.Task) (result Result, err error) {
	store, texts := p.host.Store(req.Vault), p.host.Texts(req.Vault)
	source, err := p.findSource(ctx, texts, req.Path, req.Source)
	if err != nil {
		return Result{}, err
	}
	parts, err := ReadParts(ctx, texts, source)
	if err != nil {
		return Result{}, fmt.Errorf("read %s of %s: %w", source.GetID(), req.Path, err)
	}

	turn, err := p.host.Queue.EnqueueWork(ctx, host.Work{IsUserStarted: true, OnWaiting: nil})
	if err != nil {
		return Result{}, fmt.Errorf("wait for a turn to proofread %s: %w", req.Path, err)
	}
	defer func() { err = errors.Join(err, turn.Release()) }()

	lines := numbered.ReadLines(parts)
	task.TotalCount, task.CompletedCount = int64(len(lines)), 0
	if err := p.host.Queue.ReportTask(ctx, *task); err != nil {
		return Result{}, err
	}
	corrections, uncorrected, err := p.correct(ctx, parts, lines, task)
	if err != nil {
		return Result{}, err
	}
	corrected := numbered.ApplyCorrections(parts, corrections)

	key := getVersionKey(source.GetID())
	if err := host.WriteText(ctx, store, getFileName(key), corrected); err != nil {
		return Result{}, err
	}
	version, err := texts.PutVersion(ctx, host.TextPut{
		Path: req.Path, Key: key, Source: source.GetID(), Kind: ProofreadingKind,
		InputHash: source.TextHash, File: getFileName(key),
	})
	if err != nil {
		return Result{}, fmt.Errorf("report the proofreading of %s: %w", req.Path, err)
	}
	return Result{Lines: len(lines), Corrected: len(corrections), Uncorrected: uncorrected, Version: version}, nil
}

func (p *Proofreading) findSource(ctx context.Context, texts Texts, filePath string, id domain.TextVersionID) (domain.TextVersion, error) {
	listing, err := texts.ListVersions(ctx, filePath)
	if err != nil {
		return domain.TextVersion{}, err
	}
	if id == "" {
		id = listing.Live
	}
	for _, version := range listing.Versions {
		if id == "" || version.GetID() != id {
			continue
		}
		if !p.kinds[version.Kind] && !p.kinds["*"] {
			return domain.TextVersion{}, fmt.Errorf("%s is %s text: %w", id, version.Kind, errFormat)
		}
		return version, nil
	}
	if id == "" {
		return domain.TextVersion{}, fmt.Errorf("%s: %w", filePath, errNoText)
	}
	return domain.TextVersion{}, fmt.Errorf("%s of %s: %w", id, filePath, host.ErrNotExist)
}

// ReadParts reads every segment of the version, page by page.
func ReadParts(ctx context.Context, texts Texts, version domain.TextVersion) ([]domain.TextSegment, error) {
	var parts []domain.TextSegment
	for cursor := ""; ; {
		page, err := texts.ReadText(ctx, version.Path, version.GetID(), cursor)
		if err != nil {
			return nil, err
		}
		parts = append(parts, page.Segments...)
		if page.NextCursor == "" || page.NextCursor == cursor {
			return parts, nil
		}
		cursor = page.NextCursor
	}
}

func (p *Proofreading) correct(
	ctx context.Context, parts []domain.TextSegment, lines []numbered.Line, task *host.Task,
) ([]numbered.Line, int, error) {
	if numbered.IsTimed(parts) {
		return p.correctSpeech(ctx, lines, task)
	}
	return p.correctScan(ctx, numbered.ReadPages(parts, lines), task)
}

func (p *Proofreading) correctSpeech(ctx context.Context, lines []numbered.Line, task *host.Task) ([]numbered.Line, int, error) {
	cfg := p.getConfig()
	var out []numbered.Line
	uncorrected := 0
	for from := 1; ; {
		result, err := proofreadSpeech(ctx, p.ask, lines, batchShape{size: cfg.BatchSize, overlap: cfg.Overlap}, from, cfg.InFlight*cfg.BatchSize)
		if err != nil {
			return nil, 0, err
		}
		uncorrected += result.uncorrected
		next := result.next
		for _, line := range result.corrections {
			if result.isDone || line.Number < result.next {
				out = append(out, line)
				next = max(next, line.GetLastNumber()+1)
			}
		}
		task.CompletedCount = int64(min(next-1, len(lines)))
		if err := p.host.Queue.ReportTask(ctx, *task); err != nil {
			return nil, 0, err
		}
		if result.isDone || next <= from {
			return out, uncorrected, nil
		}
		from = next
	}
}

func (p *Proofreading) correctScan(ctx context.Context, pages []numbered.Batch, task *host.Task) ([]numbered.Line, int, error) {
	cfg := p.getConfig()
	var out []numbered.Line
	uncorrected := 0
	for _, page := range pages {
		result, err := proofreadScan(ctx, p.ask, []numbered.Batch{page}, cfg.MaxEditDistance)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, result.corrections...)
		uncorrected += result.uncorrected
		task.CompletedCount = int64(page.Last())
		if err := p.host.Queue.ReportTask(ctx, *task); err != nil {
			return nil, 0, err
		}
	}
	return out, uncorrected, nil
}

// keyPrefix starts the key of every version the plugin makes.
const keyPrefix = "proofread-"

func getVersionKey(source domain.TextVersionID) string {
	sum := sha256.Sum256([]byte(source))
	return keyPrefix + hex.EncodeToString(sum[:8])
}

func getFileName(key string) string { return key + ".jsonl" }

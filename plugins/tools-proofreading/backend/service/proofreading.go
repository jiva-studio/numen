package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

var (
	// errNoText is a file no live version gives words to.
	errNoText = errors.New("the file has no text to proofread")
	// errFormat is a version of a kind the manifest does not list.
	errFormat = errors.New("the plugin proofreads the text kinds its manifest lists")
)

// Request is one text to proofread. Source is the version read, and empty for the live one.
// IsUserStarted is set when a user requested it.
type Request struct {
	Vault         domain.VaultID
	Path          string
	Source        domain.TextVersionID
	IsUserStarted bool
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
	ask       askModel
	getConfig func() proofreadingConfig
}

func newProofreading(h Host, kinds map[domain.TextKind]bool, ask askModel, getConfig func() proofreadingConfig) *Proofreading {
	return &Proofreading{host: h, kinds: kinds, ask: ask, getConfig: getConfig}
}

// Proofread reads the version, puts its lines right and puts the version made from it. A failure
// is shown as a task until the file is proofread again.
func (p *Proofreading) Proofread(ctx context.Context, req Request) (Result, error) {
	task := host.Task{
		ID: "proofread:" + req.Path, Title: "Proofreading " + path.Base(req.Path), About: req.Path,
		Unit: host.UnitThings, IsUserStarted: req.IsUserStarted,
	}
	res, err := p.proofread(ctx, req, &task)
	if err != nil && ctx.Err() == nil {
		task.Error = err.Error()
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

func (p *Proofreading) proofread(ctx context.Context, req Request, task *host.Task) (Result, error) {
	store, texts := p.host.Store, p.host.Texts
	if req.Vault != "" {
		store, texts = store.With(req.Vault), texts.With(req.Vault)
	}
	source, err := p.findSource(ctx, texts, req.Path, req.Source)
	if err != nil {
		return Result{}, err
	}
	parts, err := readParts(ctx, texts, source)
	if err != nil {
		return Result{}, fmt.Errorf("read %s of %s: %w", source.GetID(), req.Path, err)
	}

	turn, err := p.host.Queue.EnqueueWork(ctx, host.Work{IsUserStarted: req.IsUserStarted})
	if err != nil {
		return Result{}, fmt.Errorf("wait for a turn to proofread %s: %w", req.Path, err)
	}
	defer turn.Release()

	lines := readLines(parts)
	task.Total, task.Done = int64(len(lines)), 0
	if err := p.host.Queue.ReportTask(ctx, *task); err != nil {
		return Result{}, err
	}
	corrections, uncorrected, err := p.correct(ctx, parts, lines, task)
	if err != nil {
		return Result{}, err
	}
	corrected := applyCorrections(parts, corrections)

	key := getVersionKey(source.GetID())
	if err := host.WriteText(ctx, store, getFileName(key), corrected); err != nil {
		return Result{}, err
	}
	version, err := texts.PutVersion(ctx, host.TextPut{
		Path: req.Path, Key: key, Source: source.GetID(), Kind: source.Kind,
		SourceHash: source.Hash, File: getFileName(key),
	})
	if err != nil {
		return Result{}, fmt.Errorf("report the proofreading of %s: %w", req.Path, err)
	}
	return Result{Lines: len(lines), Corrected: len(corrections), Uncorrected: uncorrected, Version: version}, nil
}

// findSource is the version of the file the id names, or its live version where the id is empty.
func (p *Proofreading) findSource(ctx context.Context, texts host.Texts, filePath string, id domain.TextVersionID) (domain.TextVersion, error) {
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

// readParts is every page of the version's text, in order.
func readParts(ctx context.Context, texts host.Texts, version domain.TextVersion) ([]domain.TextSegment, error) {
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

// correct queries the model about the lines as their selectors place them: timed cues as speech, one
// window at a time, and the lines of each page as a scan.
func (p *Proofreading) correct(
	ctx context.Context, parts []domain.TextSegment, lines []Line, task *host.Task,
) ([]Line, int, error) {
	if isTimed(parts) {
		return p.correctSpeech(ctx, lines, task)
	}
	return p.correctScan(ctx, readPages(parts, lines), task)
}

// correctSpeech queries the model about the lines one window at a time. A correction is kept from a window if it
// begins before the next window's first line, and the next window starts after the last line kept.
func (p *Proofreading) correctSpeech(ctx context.Context, lines []Line, task *host.Task) ([]Line, int, error) {
	cfg := p.getConfig()
	var out []Line
	uncorrected := 0
	for from := 1; ; {
		result, err := proofreadSpeech(ctx, p.ask, lines, cfg.batchSize, cfg.overlap, from, cfg.inFlight*cfg.batchSize)
		if err != nil {
			return nil, 0, err
		}
		uncorrected += result.uncorrected
		next := result.next
		for _, line := range result.corrections {
			if result.isDone || line.Number < result.next {
				out = append(out, line)
				next = max(next, getLastLine(line)+1)
			}
		}
		task.Done = int64(min(next-1, len(lines)))
		if err := p.host.Queue.ReportTask(ctx, *task); err != nil {
			return nil, 0, err
		}
		if result.isDone || next <= from {
			return out, uncorrected, nil
		}
		from = next
	}
}

// correctScan queries the model about the lines of each page in turn.
func (p *Proofreading) correctScan(ctx context.Context, pages []Batch, task *host.Task) ([]Line, int, error) {
	cfg := p.getConfig()
	var out []Line
	uncorrected := 0
	for _, page := range pages {
		result, err := proofreadScan(ctx, p.ask, []Batch{page}, cfg.maxEditDistance)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, result.corrections...)
		uncorrected += result.uncorrected
		task.Done = int64(page.Last())
		if err := p.host.Queue.ReportTask(ctx, *task); err != nil {
			return nil, 0, err
		}
	}
	return out, uncorrected, nil
}

// getLastLine is the last line a correction answers for.
func getLastLine(line Line) int {
	if line.IsJoined() {
		return line.Last
	}
	return line.Number
}

// readLines is the parts as numbered lines, one to a part and counted from one, with the space
// around each part's words left out.
func readLines(parts []domain.TextSegment) []Line {
	lines := make([]Line, len(parts))
	for i, part := range parts {
		lines[i] = Line{Number: i + 1, Text: strings.TrimSpace(part.Text)}
	}
	return lines
}

// readPages is the lines in a batch for each run of lines on one page, numbered by the page. The
// lines are numbered as readLines numbers the parts.
func readPages(parts []domain.TextSegment, lines []Line) []Batch {
	var out []Batch
	for _, line := range lines {
		selector, _ := host.ParseSelector(parts[line.Number-1].Selector)
		page := selector.Page
		if len(out) == 0 || out[len(out)-1].Number != page {
			out = append(out, Batch{Number: page})
		}
		out[len(out)-1].Lines = append(out[len(out)-1].Lines, line)
	}
	return out
}

// isTimed reports whether the first segment is placed by a time range of a recording.
func isTimed(parts []domain.TextSegment) bool {
	if len(parts) == 0 {
		return false
	}
	selector, err := host.ParseSelector(parts[0].Selector)
	return err == nil && len(selector.T) == 2
}

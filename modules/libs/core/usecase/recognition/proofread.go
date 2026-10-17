package recognition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// Proofread puts a document's recognition right, where a person configured
// something to proofread it with.
//
// What the recogniser produced stays on disk under its own name and the
// corrections go beside it, so a proofreading that went wrong is a file that can
// be deleted and a person can always ask what the machine read.
type Proofread struct {
	readers port.VaultReaders
	derived port.DerivedStores
	by      port.Proofreader

	// Queue is where the pages are left for the proofreader to answer about
	// later. Where there is one, a run leaves a batch and comes back for it,
	// and nothing waits.
	Queue port.ProofreadQueue

	// Area is the store the recognition is kept in. Empty means the default.
	Area string

	// Hash is the known fingerprint of the recognition's source file. When set,
	// Execute reuses it without reading the source file from disk.
	Hash string

	// Pages is how many pages are asked about at once. Zero takes the default.
	Pages int

	// MaxEditDistance is how far a correction may stand from the line as read
	// and still be a correction. Zero takes what was measured.
	MaxEditDistance float64

	// Cut is optional. It makes a source's chunks, and is called as corrections
	// are written down, so a book answers about the pages already put right
	// while the rest is still being asked about. Where nothing cuts, the chunks
	// stand on the text as it was read until the next scan.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	OnProgress func(ProofreadResult)
}

// NewProofread is what a recognition is put right through: the vault it is
// read out of, the store the recognition and its corrections are kept in, and the
// proofreader that answers about a page.
//
// An installation that configured no proofreader is refused here.
func NewProofread(
	readers port.VaultReaders, derived port.DerivedStores, by port.Proofreader,
) (Proofread, error) {
	if err := run.CheckProofread(readers, derived, by); err != nil {
		return Proofread{}, err
	}
	return Proofread{readers: readers, derived: derived, by: by}, nil
}

// CreateProofread is what puts a document's recognition right, at the sizes
// this installation proofreads at and against the profile it names for a scan.
//
// held is false with no error where the installation named no profile, and the
// reading is then used exactly as it was read. A profile that is named and
// could not be opened is an error instead.
//
// Cut and OnProgress are the caller's: what a run says about itself while it
// goes belongs to whoever asked for it.
func CreateProofread(
	c run.ProofreadingConfig, readers port.VaultReaders, derived port.DerivedStores,
) (right Proofread, held bool, err error) {
	by, err := run.OpenProofreader(c.By, proofread.ScanInstruction)
	if err != nil {
		return Proofread{}, false, fmt.Errorf("nothing to proofread with: %w", err)
	}
	if by == nil {
		return Proofread{}, false, nil
	}

	// A proofreader with a queue is left the pages and answers later, and the
	// batch is collected by whatever comes back for it.
	var queue port.ProofreadQueue
	if c.Queue != nil {
		if queue, err = c.Queue(proofread.ScanInstruction); err != nil {
			return Proofread{}, false, fmt.Errorf("nothing to leave the pages with: %w", err)
		}
	}

	right, err = NewProofread(readers, derived, by)
	if err != nil {
		return Proofread{}, false, err
	}
	right.Queue = queue
	right.Pages = c.Batch
	right.MaxEditDistance = c.MaxEditDistance
	return right, true, nil
}

// GetProofreaderName is what puts the words right, by the name it goes under.
func (u Proofread) GetProofreaderName() string { return u.by.GetName() }

// ProofreadResult reports what proofreading a recognition did.
type ProofreadResult struct {
	Path             string // the document being proofread
	Pages            int    // how many pages the recognition has
	Read             int    // how many have been asked about, this run and before it
	Resumed          int    // how many a run before this one had already asked about
	Fixed            int    // lines put right
	UncorrectedPages int    // pages replied to without a correction, left as they were read
	IsNone           bool   // there is no recognition to proofread, and nothing was done
	IsWaiting        bool   // a batch is out and what comes back is not there yet
	IsBusy           bool   // somebody else is proofreading this recognition
}

// DefaultPages is how many pages one request carries.
const DefaultPages = 40

// checkpoint is what says who put a recognition right and how far they got.
type checkpoint struct {
	By    string `json:"by"`
	Pages int    `json:"pages"`
	// Batch is the name what was left for the proofreader is collected under,
	// and Left is how many pages beyond the count it covers. A recognition with no
	// batch out names neither.
	Batch string `json:"batch,omitempty"`
	Left  int    `json:"left,omitempty"`
}

// Execute proofreads one document's recognition.
func (u Proofread) Execute(ctx context.Context, v domain.Vault, path string) (ProofreadResult, error) {
	res := ProofreadResult{Path: path}
	hash, err := run.GetHash(ctx, u.readers, v, path, u.Hash)
	if err != nil {
		return res, err
	}
	store, err := u.derived.Open(v)
	if err != nil {
		return res, err
	}

	area := run.GetArea(u.Area, "ocr")
	corrections := domain.NewArtifactName(domain.ArtifactCorrections, area, hash)
	progressName := domain.NewArtifactName(domain.ArtifactProofread, area, hash)

	// One run to a recognition. The name is the one the corrections are kept under,
	// and it is held for as long as the proofreading takes.
	release, err := store.ClaimArtifact(ctx, corrections)
	if errors.Is(err, port.ErrClaimed) {
		res.IsBusy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	pages, err := u.lines(ctx, store, area, hash)
	if err != nil {
		return res, err
	}
	if len(pages) == 0 {
		res.IsNone = true
		return res, nil
	}
	res.Pages = len(pages)

	stood, err := u.readCheckpoint(ctx, store, corrections, progressName)
	if err != nil {
		return res, err
	}
	done := min(stood.Pages, len(pages))
	stood.Pages = done
	if err := dropCorrections(ctx, store, corrections, pages, done); err != nil {
		return res, err
	}
	res.Resumed, res.Read = done, done
	// Progress is reported once there is a page to ask about, so a recognition
	// nothing is left to be asked about is never work anybody is shown.
	if done < len(pages) {
		run.Report(u.OnProgress, res)
	}

	job := proofreadRecognitionJob{
		v: v, store: store, path: path, pages: pages, corrections: corrections, progressName: progressName,
	}
	if u.Queue != nil {
		err = u.await(ctx, job, stood, &res)
	} else {
		err = u.askPages(ctx, job, done, &res)
	}
	return res, err
}

// proofreadRecognitionJob is the state of proofreading one recognition: where its pages and their corrections
// are kept.
type proofreadRecognitionJob struct {
	v            domain.Vault
	store        port.DerivedStore
	path         string
	pages        []proofread.Batch
	corrections  domain.ArtifactName
	progressName domain.ArtifactName
}

// askPages asks the proofreader about the pages from at onwards, a batch at a
// time.
func (u Proofread) askPages(
	ctx context.Context,
	p proofreadRecognitionJob,
	at int,
	res *ProofreadResult,
) error {
	batch := run.GetSize(u.Pages, DefaultPages)
	for ; at < len(p.pages); at += batch {
		if err := ctx.Err(); err != nil {
			// What came back is on disk already, and the next run begins at the
			// page this one stopped on.
			return err
		}
		end := min(at+batch, len(p.pages))
		asked := p.pages[at:end]

		replies, err := u.by.Proofread(ctx, ask(asked))
		if err != nil {
			return fmt.Errorf("proofread %s: %w", p.path, err)
		}
		if err := u.writeCorrections(ctx, p, asked, replies, end, res); err != nil {
			return err
		}
	}
	return nil
}

// writeCorrections writes the corrections a batch came back with, cuts the source, and
// writes the count after both: a batch no count claims is one the next run asks
// about again, and it trims the corrections back to the count first.
func (u Proofread) writeCorrections(
	ctx context.Context,
	p proofreadRecognitionJob,
	asked []proofread.Batch,
	replies map[int]string,
	end int,
	res *ProofreadResult,
) error {
	put, uncorrected := u.getCorrections(asked, replies)
	res.UncorrectedPages += uncorrected
	if len(put) > 0 {
		if err := p.store.AppendArtifact(ctx, p.corrections, domain.PackCorrections(put)); err != nil {
			return err
		}
		res.Fixed += len(put)
	}
	res.Read = end
	if err := run.Cut(ctx, u.Cut, p.v, p.path); err != nil {
		return err
	}
	if err := u.writeCheckpoint(ctx, p.store, p.progressName, checkpoint{Pages: end}); err != nil {
		return err
	}
	run.Report(u.OnProgress, *res)
	return nil
}

// lines are the printed lines of a recognition, by the page they were read from. A
// recognition whose text or coordinates are not there is a recognition nothing can be
// asked about.
func (u Proofread) lines(
	ctx context.Context,
	store port.DerivedStore,
	area, hash string,
) ([]proofread.Batch, error) {
	artifact, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactText, area, hash))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	boxes, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactBoxes, area, hash))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	prose, _ := ocr.Read(artifact)
	return proofread.GetScanBatches(prose, highlight.Unpack(boxes)), nil
}

// getCorrections is what a run of pages had put right, and how many of them
// answered with something that was no answer.
//
// A page nothing came back about and a page whose reply the gates refused are
// the same outcome: the page is left as it was read.
func (u Proofread) getCorrections(
	asked []proofread.Batch,
	replies map[int]string,
) (put []domain.CorrectionLine, uncorrected int) {
	for _, page := range asked {
		reply, answered := replies[page.Number]
		if !answered {
			continue
		}
		lines, _, ok := proofread.GetFixedLines(page, reply, run.GetSize(u.MaxEditDistance, proofread.MaxEditDistance))
		if !ok {
			uncorrected++
			continue
		}
		for _, line := range lines {
			put = append(put, domain.CorrectionLine{Number: line.Number, Text: line.Text})
		}
	}
	return put, uncorrected
}

// readCheckpoint is how many pages a run before this one asked about, with the
// corrections past that count cut away.
//
// A count stands after the corrections it claims, so what follows the last one
// is a batch that did not land whole. Corrections another proofreader made are
// not this one's, and the recognition is taken up from the first page.
func (u Proofread) readCheckpoint(
	ctx context.Context,
	store port.DerivedStore,
	corrections, progressName domain.ArtifactName,
) (checkpoint, error) {
	raw, err := store.ReadArtifact(ctx, progressName)
	if errors.Is(err, fs.ErrNotExist) {
		return checkpoint{}, nil
	}
	if err != nil {
		return checkpoint{}, err
	}
	var stood checkpoint
	if err := json.Unmarshal(raw, &stood); err != nil || stood.By != u.by.GetName() {
		if err := store.RemoveArtifact(ctx, corrections); err != nil {
			return checkpoint{}, err
		}
		return checkpoint{}, store.RemoveArtifact(ctx, progressName)
	}
	return stood, nil
}

// await collects what the proofreader has answered about, and leaves the pages
// after it. One run collects one batch and leaves one, so a book is put right
// over as many runs as it has batches.
func (u Proofread) await(
	ctx context.Context,
	p proofreadRecognitionJob,
	stood checkpoint,
	res *ProofreadResult,
) error {
	if stood.Batch != "" {
		next, ready, err := u.fetchBatch(ctx, p, stood, res)
		if err != nil {
			return err
		}
		if !ready {
			res.IsWaiting = true
			return nil
		}
		stood = next
	}
	if stood.Pages >= len(p.pages) {
		return nil
	}
	return u.submitBatch(ctx, p, stood.Pages, res)
}

// fetchBatch collects the batch left for the proofreader and lands its
// corrections. It returns the checkpoint the recognition now stands at, and whether
// the batch was ready.
func (u Proofread) fetchBatch(
	ctx context.Context,
	p proofreadRecognitionJob,
	stood checkpoint,
	res *ProofreadResult,
) (checkpoint, bool, error) {
	replies, ready, err := u.Queue.GetReplies(ctx, stood.Batch)
	if err != nil {
		// The batch is forgotten, and the pages it covered are left again by the
		// next run.
		stood.Batch, stood.Left = "", 0
		if put := u.writeCheckpoint(ctx, p.store, p.progressName, stood); put != nil {
			return stood, false, put
		}
		return stood, false, fmt.Errorf("collect the proofreading of %s: %w", p.path, err)
	}
	if !ready {
		return stood, false, nil
	}
	end := min(stood.Pages+stood.Left, len(p.pages))
	if err := u.writeCorrections(ctx, p, p.pages[stood.Pages:end], replies, end, res); err != nil {
		return stood, false, err
	}
	return checkpoint{Pages: end}, true, nil
}

// submitBatch leaves the batch of pages from at for the proofreader, and writes
// down the name it is collected under.
func (u Proofread) submitBatch(
	ctx context.Context,
	p proofreadRecognitionJob,
	at int,
	res *ProofreadResult,
) error {
	end := min(at+run.GetSize(u.Pages, DefaultPages), len(p.pages))
	name, err := u.Queue.Submit(ctx, ask(p.pages[at:end]))
	if err != nil {
		return fmt.Errorf("leave the pages of %s: %w", p.path, err)
	}
	res.IsWaiting = true
	return u.writeCheckpoint(ctx, p.store, p.progressName, checkpoint{Pages: at, Batch: name, Left: end - at})
}

// ask is the batches with their prompt built for the proofreader.
func ask(pages []proofread.Batch) []domain.ProofreadBatch {
	out := make([]domain.ProofreadBatch, len(pages))
	for i, page := range pages {
		out[i] = domain.ProofreadBatch{
			Number: page.Number,
			Prompt: proofread.Ask(page),
		}
	}
	return out
}

// dropCorrections drops the corrections of pages no count claims. A line is
// numbered by where its run stands in the recognition, so the pages beyond the
// count are the lines from the first of them onwards.
func dropCorrections(
	ctx context.Context,
	store port.DerivedStore,
	corrections domain.ArtifactName,
	pages []proofread.Batch,
	done int,
) error {
	beyond, ok := opening(pages, done)
	if !ok {
		return nil
	}
	raw, err := store.ReadArtifact(ctx, corrections)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	held := domain.UnpackCorrections(raw)
	kept := make([]domain.CorrectionLine, 0, len(held))
	for _, line := range held {
		if line.Number < beyond {
			kept = append(kept, line)
		}
	}
	if len(kept) == len(held) {
		return nil
	}
	// Written back whatever was dropped. An append that did not land whole leaves
	// bytes that are not a record, and every record appended after them is read
	// at a shifted offset.
	return store.WriteArtifact(ctx, corrections, domain.PackCorrections(kept))
}

// opening is the number of the first line no count claims.
func opening(pages []proofread.Batch, done int) (int, bool) {
	for _, page := range pages[min(done, len(pages)):] {
		if len(page.Lines) > 0 {
			return page.Lines[0].Number, true
		}
	}
	return 0, false
}

// writeCheckpoint writes down who put this recognition right and how far they got.
// It stands last and is what makes the batch before it count.
func (u Proofread) writeCheckpoint(
	ctx context.Context, store port.DerivedStore, progressName domain.ArtifactName, stood checkpoint,
) error {
	stood.By = u.by.GetName()
	raw, err := json.MarshalIndent(stood, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteArtifact(ctx, progressName, append(raw, '\n'))
}

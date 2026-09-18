package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// errNothingProofreads is a proofreading with nothing to proofread with. An
// installation that named no profile has no proofreader at all, so nil is a
// value the settings produce and not a caller's slip.
var errNothingProofreads = errors.New("nothing to proofread with: none is configured")

// ProofreadReading puts a document's reading right, where a person configured
// something to proofread it with.
//
// What the recogniser produced stays on disk under its own name and the
// corrections go beside it, so a proofreading that went wrong is a file that can
// be deleted and a person can always ask what the machine read.
type ProofreadReading struct {
	Readers port.VaultReaders
	Derived port.DerivedStores
	// By is what puts the words right. A run given none answers that nothing
	// proofreads, and the reading is left as it was read.
	By port.Proofreader

	// Queue is where the pages are left for the proofreader to answer about
	// later. Where there is one, a run leaves a batch and comes back for it,
	// and nothing waits.
	Queue port.ProofreadQueue

	// Area is the store the reading is kept in. Empty means the default.
	Area string

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

	OnProgress func(ProofreadReadingResult)
}

// NewProofreadReading is what a reading is put right through: the vault it is
// read out of, the store the reading and its corrections are kept in, and the
// proofreader that answers about a page.
func NewProofreadReading(
	readers port.VaultReaders, derived port.DerivedStores, by port.Proofreader,
) ProofreadReading {
	return ProofreadReading{Readers: readers, Derived: derived, By: by}
}

// ProofreadReadingResult reports what proofreading a reading did.
type ProofreadReadingResult struct {
	Path             string // the document being proofread
	Pages            int    // how many pages the reading has
	Read             int    // how many have been asked about, this run and before it
	Resumed          int    // how many a run before this one had already asked about
	Fixed            int    // lines put right
	UncorrectedPages int    // pages replied to without a correction, left as they were read
	IsNone           bool   // there is no reading to proofread, and nothing was done
	IsWaiting        bool   // a batch is out and what comes back is not there yet
	IsBusy           bool   // somebody else is proofreading this reading
}

// DefaultPages is how many pages one request carries.
const DefaultPages = 40

// checkpoint is what says who put a reading right and how far they got.
type checkpoint struct {
	By    string `json:"by"`
	Pages int    `json:"pages"`
	// Batch is the name what was left for the proofreader is collected under,
	// and Left is how many pages beyond the count it covers. A reading with no
	// batch out names neither.
	Batch string `json:"batch,omitempty"`
	Left  int    `json:"left,omitempty"`
}

// Execute proofreads one document's reading.
func (u ProofreadReading) Execute(ctx context.Context, v domain.Vault, path string) (ProofreadReadingResult, error) {
	res := ProofreadReadingResult{Path: path}
	if u.By == nil {
		return res, errNothingProofreads
	}
	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return res, err
	}

	hash := text.Fingerprint(raw)
	area := u.area()
	corrections, far := text.Corrections(area, hash), text.Proofread(area, hash)

	// One run to a reading. The name is the one the corrections are kept under,
	// and it is held for as long as the proofreading takes.
	release, err := store.Claim(ctx, corrections)
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

	stood, err := u.readCheckpoint(ctx, store, corrections, far)
	if err != nil {
		return res, err
	}
	done := min(stood.Pages, len(pages))
	stood.Pages = done
	if err := dropCorrections(ctx, store, corrections, pages, done); err != nil {
		return res, err
	}
	res.Resumed, res.Read = done, done
	// Progress is reported once there is a page to ask about, so a reading
	// nothing is left to be asked about is never work anybody is shown.
	if done < len(pages) {
		u.progress(res)
	}

	if u.Queue != nil {
		return u.await(ctx, v, store, path, pages, stood, corrections, far, res)
	}

	for at := done; at < len(pages); at += u.batch() {
		if err := ctx.Err(); err != nil {
			// What came back is on disk already, and the next run begins at the
			// page this one stopped on.
			return res, err
		}
		end := min(at+u.batch(), len(pages))
		asked := pages[at:end]

		replies, err := u.By.Proofread(ctx, asked)
		if err != nil {
			return res, fmt.Errorf("proofread %s: %w", path, err)
		}
		put, uncorrected := u.getCorrections(asked, replies)
		res.UncorrectedPages += uncorrected

		// The corrections are written, the source is cut, and the count stands
		// after both: a batch no count claims is one the next run asks about
		// again, and it trims the corrections back to the count first.
		if len(put) > 0 {
			if err := store.Append(ctx, corrections, correction.Pack(put)); err != nil {
				return res, err
			}
			res.Fixed += len(put)
		}
		res.Read = end
		if err := u.cut(ctx, v, path); err != nil {
			return res, err
		}
		if err := u.writeCheckpoint(ctx, store, far, checkpoint{Pages: end}); err != nil {
			return res, err
		}
		u.progress(res)
	}
	return res, nil
}

// lines are the printed lines of a reading, by the page they were read from. A
// reading whose text or coordinates are not there is a reading nothing can be
// asked about.
func (u ProofreadReading) lines(
	ctx context.Context,
	store port.DerivedStore,
	area, hash string,
) ([]proofread.Batch, error) {
	artifact, err := store.Read(ctx, text.Artifact(area, hash))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	boxes, err := store.Read(ctx, text.Boxes(area, hash))
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
func (u ProofreadReading) getCorrections(
	asked []proofread.Batch,
	replies map[int]string,
) (put []correction.Line, uncorrected int) {
	for _, page := range asked {
		reply, answered := replies[page.Number]
		if !answered {
			continue
		}
		lines, _, ok := proofread.GetFixedLines(page, reply, u.distance())
		if !ok {
			uncorrected++
			continue
		}
		for _, line := range lines {
			put = append(put, correction.Line{Number: line.Number, Text: line.Text})
		}
	}
	return put, uncorrected
}

// readCheckpoint is how many pages a run before this one asked about, with the
// corrections past that count cut away.
//
// A count stands after the corrections it claims, so what follows the last one
// is a batch that did not land whole. Corrections another proofreader made are
// not this one's, and the reading is taken up from the first page.
func (u ProofreadReading) readCheckpoint(
	ctx context.Context,
	store port.DerivedStore,
	corrections, far string,
) (checkpoint, error) {
	raw, err := store.Read(ctx, far)
	if errors.Is(err, fs.ErrNotExist) {
		return checkpoint{}, nil
	}
	if err != nil {
		return checkpoint{}, err
	}
	var stood checkpoint
	if err := json.Unmarshal(raw, &stood); err != nil || stood.By != u.By.GetName() {
		if err := store.Remove(ctx, corrections); err != nil {
			return checkpoint{}, err
		}
		return checkpoint{}, store.Remove(ctx, far)
	}
	return stood, nil
}

// await collects what the proofreader has answered about, and leaves the pages
// after it. One run collects one batch and leaves one, so a book is put right
// over as many runs as it has batches.
func (u ProofreadReading) await(
	ctx context.Context,
	v domain.Vault,
	store port.DerivedStore,
	path string,
	pages []proofread.Batch,
	stood checkpoint,
	corrections, far string,
	res ProofreadReadingResult,
) (ProofreadReadingResult, error) {
	if stood.Batch != "" {
		replies, ready, err := u.Queue.Collect(ctx, stood.Batch)
		if err != nil {
			// The batch is forgotten, and the pages it covered are left again
			// by the next run.
			stood.Batch, stood.Left = "", 0
			if put := u.writeCheckpoint(ctx, store, far, stood); put != nil {
				return res, put
			}
			return res, fmt.Errorf("collect the proofreading of %s: %w", path, err)
		}
		if !ready {
			res.IsWaiting = true
			return res, nil
		}
		end := min(stood.Pages+stood.Left, len(pages))
		put, uncorrected := u.getCorrections(pages[stood.Pages:end], replies)
		res.UncorrectedPages += uncorrected
		if len(put) > 0 {
			if err := store.Append(ctx, corrections, correction.Pack(put)); err != nil {
				return res, err
			}
			res.Fixed += len(put)
		}
		stood = checkpoint{Pages: end}
		res.Read = end
		if err := u.cut(ctx, v, path); err != nil {
			return res, err
		}
		if err := u.writeCheckpoint(ctx, store, far, stood); err != nil {
			return res, err
		}
		u.progress(res)
	}
	if stood.Pages >= len(pages) {
		return res, nil
	}

	end := min(stood.Pages+u.batch(), len(pages))
	name, err := u.Queue.Leave(ctx, pages[stood.Pages:end])
	if err != nil {
		return res, fmt.Errorf("leave the pages of %s: %w", path, err)
	}
	res.IsWaiting = true
	return res, u.writeCheckpoint(
		ctx, store, far, checkpoint{Pages: stood.Pages, Batch: name, Left: end - stood.Pages})
}

// dropCorrections drops the corrections of pages no count claims. A line is
// numbered by where its run stands in the reading, so the pages beyond the
// count are the lines from the first of them onwards.
func dropCorrections(
	ctx context.Context,
	store port.DerivedStore,
	corrections string,
	pages []proofread.Batch,
	done int,
) error {
	beyond, ok := opening(pages, done)
	if !ok {
		return nil
	}
	raw, err := store.Read(ctx, corrections)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	held := correction.Unpack(raw)
	kept := make([]correction.Line, 0, len(held))
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
	return store.Write(ctx, corrections, correction.Pack(kept))
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

// writeCheckpoint writes down who put this reading right and how far they got.
// It stands last and is what makes the batch before it count.
func (u ProofreadReading) writeCheckpoint(ctx context.Context, store port.DerivedStore, far string, stood checkpoint) error {
	stood.By = u.By.GetName()
	raw, err := json.MarshalIndent(stood, "", "  ")
	if err != nil {
		return err
	}
	return store.Write(ctx, far, append(raw, '\n'))
}

// cut makes this source's chunks from the reading as it now stands.
func (u ProofreadReading) cut(ctx context.Context, v domain.Vault, path string) error {
	if u.Cut == nil {
		return nil
	}
	return u.Cut(ctx, v, path)
}

func (u ProofreadReading) area() string {
	if u.Area == "" {
		return "ocr"
	}
	return u.Area
}

func (u ProofreadReading) batch() int {
	if u.Pages <= 0 {
		return DefaultPages
	}
	return u.Pages
}

func (u ProofreadReading) distance() float64 {
	if u.MaxEditDistance <= 0 {
		return proofread.MaxEditDistance
	}
	return u.MaxEditDistance
}

func (u ProofreadReading) progress(res ProofreadReadingResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}

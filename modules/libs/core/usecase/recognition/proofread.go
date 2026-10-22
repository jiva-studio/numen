package recognition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocrartifact"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// ErrJoinedCorrectionNotAllowed marks a correction that joins lines, which a scan does not admit.
var ErrJoinedCorrectionNotAllowed = errors.New("scan correction cannot join lines")

// ErrUnaskedCorrectionLine marks a correction whose line number was not asked about.
var ErrUnaskedCorrectionLine = errors.New("scan correction line was not asked")

// Proofread puts a document's recognition right, where a person configured
// something to proofread it with.
//
// What the recogniser produced stays on disk under its own name and the
// corrections go beside it, so a proofreading that went wrong is a file that can
// be deleted and a person can always ask what the machine read.
type Proofread struct {
	readers  port.VaultReaders
	derived  port.DerivedStores
	provider port.Proofreader

	// Area is the store the recognition is kept in. Empty means the default.
	Area string

	// Hash is the known fingerprint of the recognition's source file. When set,
	// Execute reuses it without reading the source file from disk.
	Hash string

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
func NewProofread(
	readers port.VaultReaders, derived port.DerivedStores, proofreader port.Proofreader,
) (Proofread, error) {
	if err := run.CheckProofread(readers, derived, proofreader); err != nil {
		return Proofread{}, err
	}
	return Proofread{readers: readers, derived: derived, provider: proofreader}, nil
}

// CreateProofread is what puts a document's recognition right, with the proofreader
// it names for a scan.
//
// held is false with no error where the installation named no profile, and the
// reading is then used exactly as it was read. A profile that is named and
// could not be opened is an error instead.
//
// Cut and OnProgress are the caller's: what a run says about itself while it
// goes belongs to whoever asked for it.
func CreateProofread(
	c run.ProofreadingConfig, readers port.VaultReaders, derived port.DerivedStores,
) (right Proofread, item bool, err error) {
	delta, err := run.OpenProofreader(c.By)
	if err != nil {
		return Proofread{}, false, fmt.Errorf("nothing to proofread with: %w", err)
	}
	if delta == nil {
		return Proofread{}, false, nil
	}

	right, err = NewProofread(readers, derived, delta)
	if err != nil {
		return Proofread{}, false, err
	}
	return right, true, nil
}

// GetProofreaderName is what puts the words right, by the name it goes under.
func (u Proofread) GetProofreaderName() string { return u.provider.GetName() }

// ProofreadResult reports what proofreading a recognition did.
type ProofreadResult struct {
	Path             string // the document being proofread
	Pages            int    // how many pages the recognition has
	Read             int    // how many have been asked about, this run and before it
	Resumed          int    // how many a run before this one had already asked about
	Fixed            int    // lines put right
	UncorrectedPages int    // pages replied to without a correction, unchanged as read
	IsNone           bool   // there is no recognition to proofread, and nothing was done
	IsBusy           bool   // somebody else is proofreading this recognition
}

// checkpoint is what says who put a recognition right and how far they got.
type checkpoint struct {
	By    string `json:"by"`
	Pages int    `json:"pages"`
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

	status, err := u.readCheckpoint(ctx, store, corrections, progressName)
	if err != nil {
		return res, err
	}
	isDone := min(status.Pages, len(pages))
	status.Pages = isDone
	if err := dropCorrections(ctx, store, corrections, pages, isDone); err != nil {
		return res, err
	}
	res.Resumed, res.Read = isDone, isDone
	// Progress is reported once there is a page to ask about, so a recognition
	// nothing is left to be asked about is never work anybody is shown.
	if isDone < len(pages) {
		run.Report(u.OnProgress, res)
	}

	job := proofreadRecognitionJob{
		v: v, store: store, path: path, pages: pages, corrections: corrections, progressName: progressName,
	}
	err = u.askPages(ctx, job, isDone, &res)
	return res, err
}

// proofreadRecognitionJob is the state of proofreading one recognition: where its pages and their corrections
// are kept.
type proofreadRecognitionJob struct {
	v            domain.Vault
	store        port.DerivedStore
	path         string
	pages        []domain.ProofreadPage
	corrections  domain.ArtifactName
	progressName domain.ArtifactName
}

// askPages asks the proofreader about the pages from at onwards.
func (u Proofread) askPages(
	ctx context.Context,
	p proofreadRecognitionJob,
	pos int,
	res *ProofreadResult,
) error {
	if pos >= len(p.pages) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	request := p.pages[pos:]
	answer, err := u.provider.ProofreadScan(ctx, request)
	if err != nil {
		return fmt.Errorf("proofread %s: %w", p.path, err)
	}
	return u.writeCorrections(ctx, p, request, answer, len(p.pages), res)
}

// writeCorrections writes the corrections a batch came back with, cuts the source, and
// writes the count after both: a batch no count claims is one the next run asks
// about again, and it trims the corrections back to the count first.
func (u Proofread) writeCorrections(
	ctx context.Context,
	p proofreadRecognitionJob,
	requested []domain.ProofreadPage,
	answer domain.ScanProofreading,
	end int,
	res *ProofreadResult,
) error {
	placed, err := getCorrectionLines(requested, answer.Corrections)
	if err != nil {
		return fmt.Errorf("proofread %s: %w", p.path, err)
	}
	res.UncorrectedPages += answer.Uncorrected
	if len(placed) > 0 {
		if err := p.store.AppendArtifact(ctx, p.corrections, correction.Pack(placed)); err != nil {
			return err
		}
		res.Fixed += len(placed)
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
) ([]domain.ProofreadPage, error) {
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
	prose, _ := ocrartifact.Read(artifact)
	return getProofreadPages(prose, highlight.Unpack(boxes)), nil
}

// getCorrectionLines returns the corrections as lines of the recognition, rejecting
// any correction whose line number was not asked about or that joins lines.
func getCorrectionLines(requested []domain.ProofreadPage, made []domain.ProofreadCorrection) ([]domain.CorrectionLine, error) {
	out := make([]domain.CorrectionLine, len(made))
	for i, one := range made {
		if one.IsJoined() {
			return nil, fmt.Errorf("%w: scan correction %d-%d cannot join lines", ErrJoinedCorrectionNotAllowed, one.Number, one.Last)
		}
		out[i] = domain.CorrectionLine{Number: one.Number, Text: one.Text}
	}
	var lines []domain.ProofreadLine
	for _, page := range requested {
		lines = append(lines, page.Lines...)
	}
	if one, ok := domain.FindUnaskedCorrection(lines, made); ok {
		return nil, fmt.Errorf("%w: scan correction line %d was not asked", ErrUnaskedCorrectionLine, one.Number)
	}
	return out, nil
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
	var status checkpoint
	if err := json.Unmarshal(raw, &status); err != nil || status.By != u.provider.GetName() {
		if err := store.RemoveArtifact(ctx, corrections); err != nil {
			return checkpoint{}, err
		}
		return checkpoint{}, store.RemoveArtifact(ctx, progressName)
	}
	return status, nil
}

// dropCorrections drops the corrections of pages no count claims. A line is
// count are the lines from the first of them onwards.
func dropCorrections(
	ctx context.Context,
	store port.DerivedStore,
	corrections domain.ArtifactName,
	pages []domain.ProofreadPage,
	finished int,
) error {
	val, ok := opening(pages, finished)
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
	target := correction.Unpack(raw)
	keptLines := make([]domain.CorrectionLine, 0, len(target))
	for _, line := range target {
		if line.Number < val {
			keptLines = append(keptLines, line)
		}
	}
	if len(keptLines) == len(target) {
		return nil
	}
	// Written back whatever was dropped. An append that did not land whole leaves
	// bytes that are not a record, and every record appended after them is read
	// at a shifted offset.
	return store.WriteArtifact(ctx, corrections, correction.Pack(keptLines))
}

// opening is the number of the first line no count claims.
func opening(pages []domain.ProofreadPage, completed int) (int, bool) {
	for _, page := range pages[min(completed, len(pages)):] {
		if len(page.Lines) > 0 {
			return page.Lines[0].Number, true
		}
	}
	return 0, false
}

// writeCheckpoint writes down who put this recognition right and how far they got.
// It stands last and is what makes the batch before it count.
func (u Proofread) writeCheckpoint(
	ctx context.Context, store port.DerivedStore, progressName domain.ArtifactName, current checkpoint,
) error {
	current.By = u.provider.GetName()
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteArtifact(ctx, progressName, append(raw, '\n'))
}

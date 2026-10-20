package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Proofread puts a recording's transcript right, where a person
// configured something to proofread it with.
//
// What the model heard stays on disk under its own name and the words as they
// now stand go beside it, so a proofreading that went wrong is a file that can
// be deleted and a person can always ask what the machine heard.
//
// A correction changes words. The proofreader is given line numbers and text,
// and every cue keeps the moments it was spoken between.
type Proofread struct {
	readers port.VaultReaders
	derived port.DerivedStores
	by      port.Proofreader

	// Area is the store the transcript is kept in. Empty means the default.
	Area string

	// Hash is the known fingerprint of the transcript's source file. When set,
	// Execute reuses it without reading the source file from disk.
	Hash string

	// BatchSize is how many lines of the transcript one request to the model
	// holds. Zero takes the default.
	BatchSize int

	// Overlap is how many lines a request holds over from the one before it.
	// Speech runs on past the cut, and a line two requests answer about is
	// settled between them. Zero takes the default.
	Overlap int

	// InFlight is how many requests one call to the proofreader answers. Zero
	// takes the default.
	InFlight int

	// Cut is optional. It makes a source's chunks, and is called as the words
	// are written down, so a recording answers about the speech already put
	// right while the rest is still being asked about. Where nothing cuts, the
	// chunks stand on the words as heard until the next scan.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	OnProgress func(ProofreadResult)
}

// NewProofread is what a transcript is put right through: the vault
// the recording is read out of, the store the transcript and the words as they
// now stand are kept in, and the proofreader that answers about the lines.
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

// CreateProofread is what puts a recording's transcript right, against the
// profile this installation names for speech.
//
// held is false with no error where the installation named no profile. A
// transcript is put right in requests that hold lines over from the one before,
// because speech runs on past a cut, and several are answered in one call.
func CreateProofread(
	c run.ProofreadingConfig, readers port.VaultReaders, derived port.DerivedStores,
) (right Proofread, held bool, err error) {
	by, err := run.OpenProofreader(c.By)
	if err != nil {
		return Proofread{}, false, fmt.Errorf("nothing to proofread with: %w", err)
	}
	if by == nil {
		return Proofread{}, false, nil
	}

	right, err = NewProofread(readers, derived, by)
	if err != nil {
		return Proofread{}, false, err
	}
	right.BatchSize = c.Batch
	right.Overlap = c.Overlap
	right.InFlight = c.InFlight
	return right, true, nil
}

// GetProofreaderName is what puts the words right, by the name it goes under.
func (u Proofread) GetProofreaderName() string { return u.by.GetName() }

// ProofreadResult reports what proofreading a transcript did.
type ProofreadResult struct {
	Path               string // the recording whose transcript is being put right
	Lines              int    // how many lines the transcript has
	Read               int    // how many have been asked about, this run and before it
	Resumed            int    // how many a run before this one had already asked about
	Fixed              int    // lines put right
	Left               int    // lines asked about that stand as they were heard
	UncorrectedBatches int    // requests replied to without a correction, left as heard
	IsNone             bool   // there is no transcript to put right, and nothing was done
	IsEdited           bool   // somebody else wrote the corrections, and they are left as they left them
	IsBusy             bool   // the recording is held by another run
	IsAlready          bool   // this proofreader has been over every line, and nothing was asked
}

// How a transcript is cut up where nothing says otherwise: the lines to a
// request, the lines a request holds over from the one before it, and the
// requests one call answers.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// errNoProgress is a proofreader that asked about lines and said to continue
// from where it began.
var errNoProgress = errors.New("the proofreader did not move past the line it began at")

// errCorrectionOutside is a correction naming a line the transcript does not
// have.
var errCorrectionOutside = errors.New("the proofreader corrected a line the transcript does not have")

// errCorrectionUnasked is a correction naming a line the window did not ask
// about.
var errCorrectionUnasked = errors.New("the proofreader corrected a line it was not asked about")

// putting is what says who put a transcript right and how far they got.
type putting struct {
	By string `json:"by"`
	// At is the millisecond past which nothing has been asked about. It counts
	// in time, which putting a broken sentence back together does not move.
	At int `json:"at"`
}

// Execute puts one recording's transcript right.
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

	area := run.GetArea(u.Area, domain.ASR)
	correctionsName := domain.NewArtifactName(domain.ArtifactCorrections, area, hash)
	progressName := domain.NewArtifactName(domain.ArtifactProofread, area, hash)

	// One run to a recording. The name a transcription holds is the name held
	// here, so a run listening to a recording and a run putting its transcript
	// right are never under way at once.
	release, err := store.ClaimArtifact(ctx, domain.NewArtifactName(domain.ArtifactPartial, area, hash))
	if errors.Is(err, port.ErrClaimed) {
		res.IsBusy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	whole, beside, err := u.current(ctx, store, area, hash)
	if err != nil {
		return res, err
	}
	_, cues := transcript.Parse(whole)
	if len(cues) == 0 {
		res.IsNone = true
		return res, nil
	}

	stood, err := u.readCheckpoint(ctx, store, progressName)
	if err != nil {
		return res, err
	}
	if beside && (transcript.IsWrittenByHand(whole) || stood.By != u.by.GetName()) {
		// The words as they stand are somebody's own, and a model does not
		// correct them. Deleting the file beside the artifact gives back what
		// was heard.
		res.IsEdited = true
		return res, nil
	}
	if !beside {
		stood = putting{}
	}

	job := newTranscriptJob(v, store, path, correctionsName, progressName, cues, stood)
	res.Lines = linesBefore(cues, len(cues))
	res.Resumed = linesBefore(cues, job.from)
	res.Read, res.Left = res.Resumed, 0

	if !job.hasLinesFrom(job.from) {
		// This proofreader has been over every line.
		res.IsAlready = true
		return res, nil
	}
	// Progress is reported once there is a line to ask about, so a transcript
	// nothing is left to be asked about is never work anybody is shown.
	run.Report(u.OnProgress, res)
	// Who is putting this transcript right stands before the first words do,
	// and the count of what they have asked about stands after the lines it
	// claims.
	if err := u.writeCheckpoint(ctx, store, progressName, stood); err != nil {
		return res, err
	}
	err = u.askWindows(ctx, job, &res)
	return res, err
}

// transcriptProofreadJob is the state of proofreading one transcript: the lines
// it asks about, and which lines it has asked about and put right so far.
type transcriptProofreadJob struct {
	v     domain.Vault
	store port.DerivedStore
	path  string
	// correctionsName is where the corrected words are kept, and progressName where the
	// count of what has been asked about is kept.
	correctionsName, progressName domain.ArtifactName

	cues []domain.TranscriptCue
	// original is the transcript as this run found it, so the line a moment
	// falls in does not move as sentences are put back together.
	original []domain.TranscriptCue
	// lines are the cues of original that say something, known by the index of
	// the cue.
	lines []domain.ProofreadLine
	from  int

	asked, fixed map[int]bool
	// merged is every cue a run of lines has been put together into, and every
	// cue such a run swallowed.
	merged map[int]bool
}

// newTranscriptJob lays out what a transcript is asked about: the lines that
// say something, from the first line no run has asked about.
func newTranscriptJob(
	v domain.Vault,
	store port.DerivedStore,
	path string,
	correctionsName, progressName domain.ArtifactName,
	cues []domain.TranscriptCue,
	stood putting,
) *transcriptProofreadJob {
	original := slices.Clone(cues)
	var lines []domain.ProofreadLine
	for at, cue := range original {
		if cue.Text != "" {
			lines = append(lines, domain.ProofreadLine{Number: at, Text: cue.Text})
		}
	}
	return &transcriptProofreadJob{
		v: v, store: store, path: path, correctionsName: correctionsName, progressName: progressName,
		cues: cues, original: original, lines: lines, from: getFirstUnasked(cues, stood.At),
		asked: map[int]bool{}, fixed: map[int]bool{}, merged: map[int]bool{},
	}
}

// hasLinesFrom says whether any line that says something stands at or after the
// number.
func (p *transcriptProofreadJob) hasLinesFrom(number int) bool {
	return len(p.lines) > 0 && p.lines[len(p.lines)-1].Number >= number
}

// askWindows asks the proofreader about the transcript a window at a time, from
// the first line no run has asked about to the end.
func (u Proofread) askWindows(ctx context.Context, p *transcriptProofreadJob, res *ProofreadResult) error {
	size := run.GetSize(u.BatchSize, DefaultBatchSize)
	window := domain.SpeechWindow{
		Size:    size,
		Overlap: run.GetSize(u.Overlap, DefaultOverlap),
		From:    p.from,
		Limit:   run.GetSize(u.InFlight, DefaultInFlight) * size,
	}
	for {
		if err := ctx.Err(); err != nil {
			// What came back is on disk already, and the next run begins at the
			// line this one stopped on.
			return err
		}
		got, err := u.askWindow(ctx, p, window, res)
		if err != nil {
			return err
		}
		if got.IsDone {
			return nil
		}
		window.From = got.Next
	}
}

// askWindow asks about one window, puts what came back into the cues, and saves
// it. A proofreader that continues from where it began is refused.
func (u Proofread) askWindow(
	ctx context.Context,
	p *transcriptProofreadJob,
	window domain.SpeechWindow,
	res *ProofreadResult,
) (domain.SpeechProofreading, error) {
	got, err := u.by.ProofreadSpeech(ctx, p.lines, window)
	if err != nil {
		return got, fmt.Errorf("proofread %s: %w", p.path, err)
	}
	if !got.IsDone && got.Next <= window.From {
		return got, fmt.Errorf("proofread %s: %w", p.path, errNoProgress)
	}
	end := len(p.cues)
	if !got.IsDone {
		end = got.Next
	}
	wrote, err := p.applyCorrections(window, got.Corrections)
	if err != nil {
		return got, fmt.Errorf("proofread %s: %w", p.path, err)
	}
	res.UncorrectedBatches += got.Uncorrected
	p.markAskedLines(end)
	res.Fixed, res.Left = len(p.fixed), countUnchanged(p.asked, p.fixed)
	res.Read = res.Resumed + len(p.asked)

	if err := u.saveWindow(ctx, p, wrote, end); err != nil {
		return got, err
	}
	run.Report(u.OnProgress, *res)
	return got, nil
}

// saveWindow writes the words, cuts the source, and writes the count after
// both: a window no count claims is one the next run asks about again.
func (u Proofread) saveWindow(ctx context.Context, p *transcriptProofreadJob, wrote bool, end int) error {
	if wrote {
		if err := p.store.WriteArtifact(ctx, p.correctionsName, transcript.Marshal(p.cues)); err != nil {
			return err
		}
	}
	if err := run.Cut(ctx, u.Cut, p.v, p.path); err != nil {
		return err
	}
	return u.writeCheckpoint(ctx, p.store, p.progressName, getProgress(p.original, end))
}

// markAskedLines records the lines of the window from where this run began up
// to the line numbered end.
func (p *transcriptProofreadJob) markAskedLines(end int) {
	for _, line := range p.lines {
		if line.Number >= p.from && line.Number < end {
			p.asked[line.Number] = true
		}
	}
}

// applyCorrections puts the window's corrections into the cues, and says whether
// any cue changed. The corrections go in by the line, so what a transcript ends
// as does not turn on the order they arrive in. A window holding a correction
// of a line it did not ask about puts nothing in.
func (p *transcriptProofreadJob) applyCorrections(
	window domain.SpeechWindow, corrections []domain.ProofreadCorrection,
) (bool, error) {
	for _, one := range corrections {
		if one.Number < 0 || one.Number >= len(p.cues) || one.Last >= len(p.cues) {
			return false, fmt.Errorf("%w: %d-%d", errCorrectionOutside, one.Number, one.Last)
		}
	}
	if one, ok := domain.FindUnaskedCorrection(window.GetLines(p.lines), corrections); ok {
		return false, fmt.Errorf("%w: %d", errCorrectionUnasked, one.Number)
	}
	sorted := slices.Clone(corrections)
	slices.SortStableFunc(sorted, func(a, b domain.ProofreadCorrection) int { return a.Number - b.Number })
	wrote := false
	for _, one := range sorted {
		if p.putLine(one) {
			wrote = true
		}
	}
	return wrote, nil
}

// putLine puts one correction into the cues, and says whether it did. Words
// answered about a line already put together with another no longer stand on
// their own.
func (p *transcriptProofreadJob) putLine(one domain.ProofreadCorrection) bool {
	last := max(one.Last, one.Number)
	if isJoined(p.merged, one.Number, last) {
		return false
	}
	p.cues[one.Number].Text = one.Text
	p.fixed[one.Number] = true
	if last > one.Number {
		p.joinCues(one.Number, last)
	}
	return true
}

// joinCues makes a sentence put back together one cue, from the first moment of
// the run to the last. The cues it swallowed say nothing, and nothing is what a
// transcript writes them as.
func (p *transcriptProofreadJob) joinCues(line, last int) {
	p.merged[line] = true
	for gone := line + 1; gone <= last; gone++ {
		p.cues[line].To = max(p.cues[line].To, p.cues[gone].To)
		p.cues[gone].Text = ""
		p.merged[gone] = true
		p.fixed[gone] = true
	}
}

// current is the transcript as it now stands, and whether it is the file that
// stands beside the artifact. A recording nothing has listened to is nothing to
// put right.
func (u Proofread) current(
	ctx context.Context,
	store port.DerivedStore,
	area, hash string,
) ([]byte, bool, error) {
	raw, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactCorrections, area, hash))
	if err == nil {
		return raw, true, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, false, err
	}
	raw, err = store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactText, area, hash))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return raw, false, nil
}

// readCheckpoint is who put this transcript right and how far they got.
func (u Proofread) readCheckpoint(
	ctx context.Context, store port.DerivedStore, progressName domain.ArtifactName,
) (putting, error) {
	raw, err := store.ReadArtifact(ctx, progressName)
	if errors.Is(err, fs.ErrNotExist) {
		return putting{}, nil
	}
	if err != nil {
		return putting{}, err
	}
	var stood putting
	if err := json.Unmarshal(raw, &stood); err != nil {
		return putting{}, fmt.Errorf("read checkpoint %s: %w", progressName, err)
	}
	return stood, nil
}

// writeCheckpoint writes down who is putting this transcript right and how far
// they have got.
func (u Proofread) writeCheckpoint(
	ctx context.Context, store port.DerivedStore, progressName domain.ArtifactName, stood putting,
) error {
	stood.By = u.by.GetName()
	raw, err := json.MarshalIndent(stood, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteArtifact(ctx, progressName, append(raw, '\n'))
}

// getFirstUnasked is the first line no run has asked about. A line ending at
// the moment a run reached is one that run asked about.
func getFirstUnasked(cues []domain.TranscriptCue, ms int) int {
	for at, cue := range cues {
		if cue.To > ms {
			return at
		}
	}
	return len(cues)
}

// isJoined says whether a correction of the lines from first to last answers
// about a line already put together with another. Those words no longer stand
// on their own.
func isJoined(merged map[int]bool, first, last int) bool {
	for at := first; at <= last; at++ {
		if merged[at] {
			return true
		}
	}
	return false
}

// getProgress is where a run stands once the lines before the one numbered end
// have been answered: the moment the last of them ended.
func getProgress(original []domain.TranscriptCue, end int) putting {
	if end <= 0 {
		return putting{}
	}
	return putting{At: original[min(end, len(original))-1].To}
}

// countUnchanged reports how many of the asked lines remain unmodified.
func countUnchanged(asked, fixed map[int]bool) int {
	out := 0
	for line := range asked {
		if !fixed[line] {
			out++
		}
	}
	return out
}

// linesBefore is how many of the cues before one carry a line. A cue saying
// nothing is no line.
func linesBefore(cues []domain.TranscriptCue, at int) int {
	out := 0
	for _, cue := range cues[:min(at, len(cues))] {
		if cue.Text != "" {
			out++
		}
	}
	return out
}

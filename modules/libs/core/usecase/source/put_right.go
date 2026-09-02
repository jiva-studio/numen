package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// PutRight puts a transcript right, where a person configured something to
// proofread it with.
//
// What the model heard stays on disk under its own name and the words as they
// now stand go beside it, so a proofreading that went wrong is a file that can
// be deleted and a person can always ask what the machine heard.
//
// A correction changes words. The proofreader is given line numbers and text,
// and every cue keeps the moments it was spoken between.
type PutRight struct {
	Readers port.VaultReaders
	Derived port.DerivedStores
	By      port.Proofreader

	// Area is the store the transcript is kept in. Empty means the default.
	Area string

	// BatchSize is how many lines of the transcript one batch holds. Zero takes
	// the default.
	BatchSize int

	// Overlap is how many lines a batch holds over from the one before it.
	// Speech runs on past the cut, and a line two batches answer about is
	// settled between them. Zero takes the default.
	Overlap int

	// InFlight is how many batches a proofreader is asked about at once. Zero
	// takes the default.
	InFlight int

	// Cut makes a source's chunks. It is called as the words are written down,
	// so a recording answers about the speech already put right while the rest
	// is still being asked about.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	OnProgress func(PutRightResult)
}

// PutRightResult reports what putting a transcript right did.
type PutRightResult struct {
	Path    string // the recording whose transcript is being put right
	Lines   int    // how many lines the transcript has
	Read    int    // how many have been asked about, this run and before it
	Resumed int    // how many a run before this one had already asked about
	Fixed   int    // lines put right
	Left    int    // lines asked about that stand as they were heard
	Refused int    // batches whose reply was no answer, and were left as heard
	None    bool   // there is no transcript to put right, and nothing was done
	Edited  bool   // somebody else wrote what stands, and it is left as they left it
	Busy    bool   // the recording is held by another run
}

// How a transcript is cut up where nothing says otherwise: the lines to a
// batch, the lines a batch holds over from the one before it, and the batches
// to a request.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// putting is what says who put a transcript right and how far they got.
type putting struct {
	By string `json:"by"`
	// At is the millisecond past which nothing has been asked about. It counts
	// in time, which putting a broken sentence back together does not move.
	At int `json:"at"`
	// Seam is the millisecond past which no seam has been asked about. It
	// stands still until At stands at the end of the transcript.
	Seam int `json:"seam"`
}

// Execute puts one recording's transcript right.
func (u PutRight) Execute(ctx context.Context, v domain.Vault, path string) (PutRightResult, error) {
	res := PutRightResult{Path: path}
	if u.By == nil {
		return res, errors.New("no proofreader: none is configured")
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
	stands, far := text.Corrected(area, hash), text.Proofread(area, hash)

	// One run to a recording. The name a transcription holds is the name held
	// here, so a run listening to a recording and a run putting its transcript
	// right are never under way at once.
	release, err := store.Claim(ctx, text.Partial(area, hash))
	if errors.Is(err, port.ErrClaimed) {
		res.Busy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	whole, beside, err := u.standing(ctx, store, area, hash)
	if err != nil {
		return res, err
	}
	_, cues := transcript.Parse(whole)
	if len(cues) == 0 {
		res.None = true
		return res, nil
	}

	stood, err := u.taken(ctx, store, far)
	if err != nil {
		return res, err
	}
	if beside && (transcript.Written(whole) || stood.By != u.By.Name()) {
		// The words as they stand are somebody's own, and a model does not
		// correct them. Deleting the file beside the artifact gives back what
		// was heard.
		res.Edited = true
		return res, nil
	}
	if !beside {
		stood = putting{}
	}

	// The transcript is asked about batch by batch, and then once more around
	// the cuts a sentence was answered for past the end of.
	spoken := proofread.Spoken(cues, u.batchSize(), u.overlap())
	batches := spoken
	// The seams are cut from the transcript as this run found it, so the batch
	// a line falls in does not move as sentences are put back together.
	asHeard := slices.Clone(cues)
	// A batch reaches back over the lines it shares with the one before it, so
	// what this run counts as read begins where the run before it stopped.
	from := unasked(cues, stood.At)
	res.Lines = linesBefore(cues, len(cues))
	res.Resumed = linesBefore(cues, from)
	res.Read, res.Left = res.Resumed, 0

	at := after(spoken, cues, stood.At)
	if at >= len(spoken) {
		// A run taking up after the first pass holds no reply saying which cuts
		// a sentence was answered for past the end of, and asks about every
		// seam standing past the count.
		batches = slices.Concat(spoken, proofread.Seams(asHeard, u.batchSize(), u.overlap(), everyCut(len(spoken))))
		at = len(spoken) + after(batches[len(spoken):], cues, stood.Seam)
	}
	if at >= len(batches) {
		return res, nil
	}
	// Progress is reported once there is a batch to ask about, so a transcript
	// nothing is left to be asked about is never work anybody is shown.
	u.progress(res)
	// Who is putting this transcript right stands before the first words do,
	// and the count of what they have asked about stands after the lines it
	// claims.
	if err := u.counted(ctx, store, far, stood); err != nil {
		return res, err
	}

	// together is every cue a run of lines has been put together into, and every
	// cue such a run swallowed.
	asked, fixed, together := map[int]bool{}, map[int]bool{}, map[int]bool{}
	// cuts is every cut a sentence was answered for past the end of.
	var cuts []int
	for ; at < len(batches); at += u.inFlight() {
		if err := ctx.Err(); err != nil {
			// What came back is on disk already, and the next run begins at the
			// line this one stopped on.
			return res, err
		}
		end := min(at+u.inFlight(), len(batches))
		group := batches[at:end]

		replies, err := u.By.Read(ctx, group)
		if err != nil {
			return res, fmt.Errorf("proofread %s: %w", path, err)
		}
		res.Refused += refused(group, replies, unbounded)
		for _, batch := range group {
			for _, line := range batch.Lines {
				if line.At >= from {
					asked[line.At] = true
				}
			}
		}

		wrote := false
		put, past := proofread.Gathered(group, replies, unbounded)
		// A seam is asked about once, so a run past the end of one names no
		// further cut.
		for _, batch := range past {
			if batch < len(spoken) {
				cuts = append(cuts, batch)
			}
		}
		// The corrections go in by the line, so what a transcript ends as does
		// not turn on the order a map hands them back in.
		for _, line := range slices.Sorted(maps.Keys(put)) {
			said := put[line]
			if joined(together, said) {
				continue
			}
			cues[line].Text = said.Text
			wrote = true
			fixed[line] = true
			if !said.Joins() {
				continue
			}
			// A sentence put back together is one cue, from the first moment of
			// the run to the last. The cues it swallowed say nothing, and
			// nothing is what a transcript writes them as.
			together[line] = true
			for gone := line + 1; gone <= said.Through; gone++ {
				cues[line].To = max(cues[line].To, cues[gone].To)
				cues[gone].Text = ""
				together[gone] = true
				fixed[gone] = true
			}
		}
		res.Fixed, res.Left = len(fixed), left(asked, fixed)

		// The words are written, the source is cut, and the count stands after
		// both: a batch no count claims is one the next run asks about again.
		if wrote {
			if err := store.Write(ctx, stands, transcript.Marshal(cues)); err != nil {
				return res, err
			}
		}
		res.Read = res.Resumed + len(asked)
		if err := u.cut(ctx, v, path); err != nil {
			return res, err
		}
		if end == len(spoken) {
			batches = slices.Concat(spoken, proofread.Seams(asHeard, u.batchSize(), u.overlap(), cuts))
		}
		if err := u.counted(ctx, store, far, reached(batches, len(spoken), end, cues)); err != nil {
			return res, err
		}
		u.progress(res)
	}
	return res, nil
}

// standing is the transcript as it now stands, and whether it is the file that
// stands beside the artifact. A recording nothing has listened to is nothing to
// put right.
func (u PutRight) standing(
	ctx context.Context,
	store port.DerivedStore,
	area, hash string,
) ([]byte, bool, error) {
	raw, err := store.Read(ctx, text.Corrected(area, hash))
	if err == nil {
		return raw, true, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, false, err
	}
	raw, err = store.Read(ctx, text.Artifact(area, hash))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return raw, false, nil
}

// taken is who put this transcript right and how far they got. A record nothing
// here can read names nobody.
func (u PutRight) taken(ctx context.Context, store port.DerivedStore, far string) (putting, error) {
	raw, err := store.Read(ctx, far)
	if errors.Is(err, fs.ErrNotExist) {
		return putting{}, nil
	}
	if err != nil {
		return putting{}, err
	}
	var stood putting
	if err := json.Unmarshal(raw, &stood); err != nil {
		return putting{}, nil
	}
	return stood, nil
}

// counted writes down who is putting this transcript right and how far they
// have got.
func (u PutRight) counted(ctx context.Context, store port.DerivedStore, far string, stood putting) error {
	stood.By = u.By.Name()
	raw, err := json.MarshalIndent(stood, "", "  ")
	if err != nil {
		return err
	}
	return store.Write(ctx, far, append(raw, '\n'))
}

// refused is how many of a run's batches answered with what is no answer. Such
// a batch is one nothing was learned from, and its lines stand as they were
// heard.
func refused(asked []proofread.Batch, replies map[int]string, apart float64) int {
	out := 0
	for _, batch := range asked {
		reply, answered := replies[batch.At]
		if !answered {
			continue
		}
		if _, _, ok := proofread.Fixed(batch, reply, apart); !ok {
			out++
		}
	}
	return out
}

// after is the first batch holding a line no run has asked about.
func after(batches []proofread.Batch, cues []transcript.Cue, ms int) int {
	from := unasked(cues, ms)
	at := 0
	for at < len(batches) && last(batches[at]) < from {
		at++
	}
	return at
}

// unasked is the first line no run has asked about. A line ending at the moment
// a run reached is one that run asked about.
func unasked(cues []transcript.Cue, ms int) int {
	for at, cue := range cues {
		if cue.To > ms {
			return at
		}
	}
	return len(cues)
}

// joined says whether a correction answers about a line already put together
// with another. Those words no longer stand on their own.
func joined(together map[int]bool, said proofread.Line) bool {
	for at := said.At; at <= said.Through; at++ {
		if together[at] {
			return true
		}
	}
	return false
}

// reached is where a run stands once the first end batches have been answered.
// The first spoken of them are the pass over the whole transcript and the rest
// the pass over its seams, and each batch of a pass reaches further into the
// transcript than the one before it.
func reached(batches []proofread.Batch, spoken, end int, cues []transcript.Cue) putting {
	var stood putting
	if done := min(end, spoken); done > 0 {
		stood.At = cues[last(batches[done-1])].To
	}
	// Every batch is answered and no seam is left, so both counts stand at the
	// end of the transcript.
	if end == len(batches) {
		stood.Seam = stood.At
		return stood
	}
	if end > spoken {
		stood.Seam = cues[last(batches[end-1])].To
	}
	return stood
}

// everyCut is the cut after each of a run of batches but the last.
func everyCut(batches int) []int {
	out := make([]int, max(batches-1, 0))
	for at := range out {
		out[at] = at
	}
	return out
}

// left is how many of the lines a run asked about stand as they were heard.
func left(asked, fixed map[int]bool) int {
	out := 0
	for line := range asked {
		if !fixed[line] {
			out++
		}
	}
	return out
}

// last is the number of the final line a batch holds.
func last(batch proofread.Batch) int {
	return batch.Lines[len(batch.Lines)-1].At
}

// linesBefore is how many of the cues before one carry a line. A cue saying
// nothing is no line.
func linesBefore(cues []transcript.Cue, at int) int {
	out := 0
	for _, cue := range cues[:min(at, len(cues))] {
		if cue.Text != "" {
			out++
		}
	}
	return out
}

// cut makes this source's chunks from the transcript as it now stands.
func (u PutRight) cut(ctx context.Context, v domain.Vault, path string) error {
	if u.Cut == nil {
		return nil
	}
	return u.Cut(ctx, v, path)
}

func (u PutRight) area() string {
	if u.Area == "" {
		return text.ASR
	}
	return u.Area
}

func (u PutRight) batchSize() int {
	if u.BatchSize <= 0 {
		return DefaultBatchSize
	}
	return u.BatchSize
}

func (u PutRight) overlap() int {
	if u.Overlap <= 0 {
		return DefaultOverlap
	}
	return u.Overlap
}

func (u PutRight) inFlight() int {
	if u.InFlight <= 0 {
		return DefaultInFlight
	}
	return u.InFlight
}

// unbounded holds a correction to speech to no distance from what was heard.
const unbounded = 0.0

func (u PutRight) progress(res PutRightResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}

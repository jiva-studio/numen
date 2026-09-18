package source

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// ProofreadingConfig is what a text is put right with: what answers, where
// batches are left for it to answer about later, and how much of a text goes
// over at a time.
//
// Naming no proofreader is a text used exactly as it was made.
type ProofreadingConfig struct {
	// IsNamed says whether a profile is named for this kind of text. A person is
	// offered the run where one is.
	IsNamed bool
	// IsAutomatic says whether a text is put right without anybody asking.
	IsAutomatic bool
	// By opens what answers about a batch, and Queue where batches are left for
	// it to answer about later. Each is told what it is proofreading. Both
	// answer nothing where no profile is named, and the reason where one cannot
	// be opened.
	By    func(instruction string) (port.Proofreader, error)
	Queue func(instruction string) (port.ProofreadQueue, error)
	// Batch is how much of a text one request carries, Overlap how much of it
	// the request before also carried, and InFlight how many stand out at once.
	Batch, Overlap, InFlight int
	// MaxEditDistance is how far a reply may stand from the text before it is
	// refused as an answer about something else.
	MaxEditDistance float64
}

// Reading is what puts a document's reading right, at the sizes this
// installation proofreads at and against the profile it names for a scan.
//
// held is false with no error where the installation named no profile, and the
// reading is then used exactly as it was read. A profile that is named and
// could not be opened is an error instead.
//
// Cut and OnProgress are the caller's: what a run says about itself while it
// goes belongs to whoever asked for it.
func (c ProofreadingConfig) Reading(
	readers port.VaultReaders, derived port.DerivedStores,
) (right ProofreadReading, held bool, err error) {
	by, err := openProofreader(c.By, proofread.ScanInstruction)
	if err != nil {
		return ProofreadReading{}, false, fmt.Errorf("nothing to proofread with: %w", err)
	}
	if by == nil {
		return ProofreadReading{}, false, nil
	}

	// A proofreader with a queue is left the pages and answers later, and the
	// batch is collected by whatever comes back for it.
	var queue port.ProofreadQueue
	if c.Queue != nil {
		if queue, err = c.Queue(proofread.ScanInstruction); err != nil {
			return ProofreadReading{}, false,
				fmt.Errorf("nothing to leave the pages with: %w", err)
		}
	}

	right, err = NewProofreadReading(readers, derived, by)
	if err != nil {
		return ProofreadReading{}, false, err
	}
	right.Queue = queue
	right.Pages = c.Batch
	right.MaxEditDistance = c.MaxEditDistance
	return right, true, nil
}

// Transcript is the same for what a model heard, against the profile this
// installation names for speech.
//
// A transcript is put right in batches that hold lines over from the one
// before, because speech runs on past a cut, and several stand out at once.
func (c ProofreadingConfig) Transcript(
	readers port.VaultReaders, derived port.DerivedStores,
) (right ProofreadTranscript, held bool, err error) {
	by, err := openProofreader(c.By, proofread.SpeechInstruction)
	if err != nil {
		return ProofreadTranscript{}, false, fmt.Errorf("nothing to proofread with: %w", err)
	}
	if by == nil {
		return ProofreadTranscript{}, false, nil
	}

	right, err = NewProofreadTranscript(readers, derived, by)
	if err != nil {
		return ProofreadTranscript{}, false, err
	}
	right.BatchSize = c.Batch
	right.Overlap = c.Overlap
	right.InFlight = c.InFlight
	return right, true, nil
}

// openProofreader is what answers about a batch, and nothing where an
// installation placed nothing to open.
func openProofreader(
	open func(string) (port.Proofreader, error), instruction string,
) (port.Proofreader, error) {
	if open == nil {
		return nil, nil
	}
	return open(instruction)
}

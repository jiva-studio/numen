package source

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// Reading is what puts a document's reading right, at the sizes this
// installation proofreads at and against the profile it names for a scan.
//
// held is false where no proofreader answers, and a reading is then used
// exactly as it was read: an installation that named no profile, and one whose
// profile could not be opened, both arrive here.
//
// Cut and OnProgress are the caller's: what a run says about itself while it
// goes belongs to whoever asked for it.
func (c ProofreadingConfig) Reading(
	readers port.VaultReaders, derived port.DerivedStores,
) (right ProofreadReading, held bool, err error) {
	by, err := opened(c.By, proofread.ScanInstruction)
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

	right = NewProofreadReading(readers, derived, by)
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
	by, err := opened(c.By, proofread.SpeechInstruction)
	if err != nil {
		return ProofreadTranscript{}, false, fmt.Errorf("nothing to proofread with: %w", err)
	}
	if by == nil {
		return ProofreadTranscript{}, false, nil
	}

	right = NewProofreadTranscript(readers, derived, by)
	right.BatchSize = c.Batch
	right.Overlap = c.Overlap
	right.InFlight = c.InFlight
	return right, true, nil
}

// opened is what answers about a batch, and nothing where an installation
// placed nothing to open.
func opened(
	open func(string) (port.Proofreader, error), instruction string,
) (port.Proofreader, error) {
	if open == nil {
		return nil, nil
	}
	return open(instruction)
}

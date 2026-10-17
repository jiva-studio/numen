package run

import (
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/port"
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
	// By opens what answers about a request, and Queue where requests are left
	// for it to answer about later. Both answer nothing where no profile is named, and the reason where one cannot
	// be opened.
	By    func() (port.Proofreader, error)
	Queue func() (port.ProofreadQueue, error)
	// Batch is how much of a text one request carries, Overlap how much of it
	// the request before also carried, and InFlight how many stand out at once.
	Batch, Overlap, InFlight int
	// MaxEditDistance is how far a reply may stand from the text before it is
	// refused as an answer about something else.
	MaxEditDistance float64
}

// The faults a proofreading is refused with. A profile that has no proofreader
// at all is a value the settings produce and not a caller's slip; the other two
// are.
var (
	ErrNothingProofreads  = errors.New("nothing to proofread with: none is configured")
	ErrNoVaultToProofread = errors.New("no vault to proofread out of")
	ErrNoStoreToProofread = errors.New("no store to keep a proofreading in")
)

// CheckProofread says why a proofreading cannot be made out of these, and
// nothing where it can.
func CheckProofread(readers port.VaultReaders, derived port.DerivedStores, by port.Proofreader) error {
	switch {
	case readers == nil:
		return ErrNoVaultToProofread
	case derived == nil:
		return ErrNoStoreToProofread
	case by == nil:
		return ErrNothingProofreads
	}
	return nil
}

// OpenProofreader is what answers about a request, and nothing where an
// installation placed nothing to open.
func OpenProofreader(open func() (port.Proofreader, error)) (port.Proofreader, error) {
	if open == nil {
		return nil, nil
	}
	return open()
}

// GetProofreadID is what putting one file's text right is called, wherever it
// is shown. One file is one line, and it replaces itself as the text is put
// right.
func GetProofreadID(path string) string { return "proofreading-" + path }

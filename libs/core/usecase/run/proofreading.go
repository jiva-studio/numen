package run

import (
	"errors"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ProofreadingConfig is what a text is put right with: what answers, and how
// much of a text goes over at a time.
//
// Naming no proofreader is a text used exactly as it was made.
type ProofreadingConfig struct {
	// IsNamed says whether a profile is named for this kind of text. A person is
	// offered the run where one is.
	IsNamed bool
	// IsAutomatic says whether a text is put right without anybody asking.
	IsAutomatic bool
	// By opens what answers about a text. It answers nothing where no profile is
	// named, and the reason where one cannot be opened.
	By func() (recognitionport.Proofreader, error)
}

// at all is a value the settings produce and not a caller's slip; the other two
// are.
var (
	ErrNothingProofreads  = errors.New("nothing to proofread with: none is configured")
	ErrNoVaultToProofread = errors.New("no vault to proofread out of")
	ErrNoStoreToProofread = errors.New("no store to keep a proofreading in")
)

// CheckProofread says why a proofreading cannot be made out of these, and
// nothing where it can.
func CheckProofread(readers vaultport.VaultReaders, derived vaultport.DerivedStores, proofreader recognitionport.Proofreader) error {
	switch {
	case readers == nil:
		return ErrNoVaultToProofread
	case derived == nil:
		return ErrNoStoreToProofread
	case proofreader == nil:
		return ErrNothingProofreads
	}
	return nil
}

// OpenProofreader is what answers about a text, and nothing where an
// installation placed nothing to open.
func OpenProofreader(open func() (recognitionport.Proofreader, error)) (recognitionport.Proofreader, error) {
	if open == nil {
		return nil, nil
	}
	return open()
}

// GetProofreadID is what putting one file's text right is called, wherever it
// is shown. One file is one line, and it replaces itself as the text is put
// right.
func GetProofreadID(path string) string { return "proofreading-" + path }

// GetProofreadError is why a proofreading stopped, in the words the list of
// what is being done shows. A proofreader the person has not let ask a model is
// told by what allows it.
func GetProofreadError(err error) string {
	if errors.Is(err, completionport.ErrNoConsent) {
		return completionport.ErrNoConsent.Error()
	}
	return err.Error()
}

package run

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

type answering struct{}

func (answering) GetName() string { return "answering" }

func (answering) ProofreadScan(context.Context, []domain.ProofreadPage, float64) (domain.ScanProofreading, error) {
	return domain.ScanProofreading{}, nil
}

func (answering) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, domain.SpeechWindow,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, nil
}

type readers struct{}

func (readers) Open(domain.Vault) (port.VaultReader, error) { return nil, nil }

type stores struct{}

func (stores) Open(domain.Vault) (port.DerivedStore, error) { return nil, nil }

// A proofreading is made only out of a vault, a store and a proofreader, and
// each missing one is a fault of its own.
func TestAProofreadingNeedsAVaultAStoreAndAProofreader(t *testing.T) {
	for _, tc := range []struct {
		name    string
		readers port.VaultReaders
		derived port.DerivedStores
		by      port.Proofreader
		want    error
	}{
		{"no vault", nil, stores{}, answering{}, ErrNoVaultToProofread},
		{"no store", readers{}, nil, answering{}, ErrNoStoreToProofread},
		{"no proofreader", readers{}, stores{}, nil, ErrNothingProofreads},
		{"everything", readers{}, stores{}, answering{}, nil},
	} {
		if got := CheckProofread(tc.readers, tc.derived, tc.by); !errors.Is(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Nothing is opened where an installation placed nothing to open.
func TestNothingIsOpenedWhereNothingWasPlaced(t *testing.T) {
	if by, err := OpenProofreader(nil); by != nil || err != nil {
		t.Errorf("got %v, %v", by, err)
	}
}

// What the installation opens is what answers.
func TestWhatTheInstallationOpensAnswers(t *testing.T) {
	by, err := OpenProofreader(func() (port.Proofreader, error) { return answering{}, nil })
	if err != nil || by == nil {
		t.Errorf("got %v, %v", by, err)
	}
}

// A proofreading stopped for want of consent says where to allow it; any other
// stop says what stopped it.
func TestAProofreadErrorSaysWhereToAllowAModel(t *testing.T) {
	refused := fmt.Errorf("batch 1: unavailable: %w", port.ErrNoConsent)
	if got := GetProofreadError(refused); got != "AI access not allowed — allow it in Settings → AI" {
		t.Errorf("said %q", got)
	}
	if got := GetProofreadError(errors.New("the model broke")); got != "the model broke" {
		t.Errorf("said %q", got)
	}
}

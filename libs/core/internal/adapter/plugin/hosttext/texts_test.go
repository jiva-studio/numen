package hosttext_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// openFailure fails to open every vault, and records the vaults it received open requests for.
type openFailure struct {
	opened *[]domain.VaultID
}

func (o openFailure) Open(v domain.Vault) (vaultport.VaultReader, error) {
	*o.opened = append(*o.opened, v.ID)
	return nil, errors.New("vault is gone")
}

func TestListVersionsReturnsTheReaderFailure(t *testing.T) {
	var opened []domain.VaultID
	texts := hosttext.NewTexts(nil, openFailure{opened: &opened})
	if _, err := texts.ListVersions(t.Context(), domain.Vault{ID: "v1"}, "talk.mp3"); err == nil || errors.Is(err, textport.ErrNoText) {
		t.Fatalf("got %v, want the reader's failure", err)
	}
}

func TestListVersionsReadsTheVaultItIsGiven(t *testing.T) {
	var opened []domain.VaultID
	texts := hosttext.NewTexts(nil, openFailure{opened: &opened})
	if _, err := texts.ListVersions(t.Context(), domain.Vault{ID: "v2"}, "talk.mp3"); err == nil {
		t.Error("expected the reader's failure")
	}
	if len(opened) != 1 || opened[0] != "v2" {
		t.Errorf("opened %v, want v2", opened)
	}
}

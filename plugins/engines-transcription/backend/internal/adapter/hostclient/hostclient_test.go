package hostclient

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func newFiles(lectures *hosttest.Vault) Files {
	return NewFiles(func(id domain.VaultID) VaultReader { return lectures.With(id) })
}

func TestReadFileReadsTheVaultNamed(t *testing.T) {
	lectures := hosttest.New("Lectures")
	chemistry := hosttest.New("Chemistry")
	lectures.Beside(chemistry)
	chemistry.File("assets/only-chemistry.wav", "chem bytes")

	raw, err := newFiles(lectures).ReadFile(t.Context(), "Chemistry", "assets/only-chemistry.wav")

	if err != nil || string(raw) != "chem bytes" {
		t.Fatalf("ReadFile in Chemistry = %q, %v", raw, err)
	}
}

func TestGetActiveVaultIDIsTheVaultTheHostShows(t *testing.T) {
	lectures := hosttest.New("Lectures")

	id, err := newFiles(lectures).GetActiveVaultID(t.Context())

	if err != nil || id != lectures.Vault().ID {
		t.Fatalf("GetActiveVaultID = %q, %v, want %q", id, err, lectures.Vault().ID)
	}
}

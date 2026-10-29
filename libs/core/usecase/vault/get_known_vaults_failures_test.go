package vault_test

import (
	"errors"
	"testing"

	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A list that cannot be read is an error, and no vault is answered with.
func TestAListThatCannotBeReadIsAnError(t *testing.T) {
	t.Parallel()
	list, there := twoRows()
	list.fails = errors.New("the list cannot be read")
	list.rows = nil

	recognized, err := vaults.NewGetKnownVaults(list, there).Execute("")
	if !errors.Is(err, list.fails) {
		t.Fatalf("got %v", err)
	}
	if recognized != nil {
		t.Errorf("the vaults came back with the error: %v", recognized)
	}
}

// forgetful is a list that holds its rows and cannot say which one was opened
// last.
type forgetful struct {
	*theList
	err error
}

func (f forgetful) GetLastVault() (domain.Vault, bool, error) { return domain.Vault{}, false, f.err }

// Which vault was opened last that cannot be read is an error, and no vault is
// answered with.
func TestTheLastVaultThatCannotBeReadIsAnError(t *testing.T) {
	t.Parallel()
	list, there := twoRows()
	cannot := errors.New("the last vault cannot be read")

	recognized, err := vaults.NewGetKnownVaults(forgetful{theList: list, err: cannot}, there).Execute("")
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if recognized != nil {
		t.Errorf("the vaults came back with the error: %v", recognized)
	}
}

// Where no vault is named as in use and none was opened last, no vault is the
// current one, whatever identity it carries.
func TestNoVaultIsCurrentWhereNoneIsNamedAndNoneWasOpened(t *testing.T) {
	t.Parallel()
	list := &theList{rows: []domain.Vault{{ID: "", Name: "Nameless", Path: "/notes/nameless"}}}

	recognized, err := vaults.NewGetKnownVaults(list, folders{"": true}).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 1 {
		t.Fatalf("%d vaults of a list of one", len(recognized))
	}
	if recognized[0].IsCurrent {
		t.Error("a vault was called current when unnamed one")
	}
}

package vault_test

import (
	"errors"
	"io/fs"
	"slices"
	"testing"

	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// theList is the vaults an installation holds, in memory: the rows, the
// last-opened vault, and how often the last-opened vault was queried.
type theList struct {
	rows      []domain.Vault
	last      domain.Vault
	isKept    bool
	requested int
	fails     error
}

func (l *theList) List() ([]domain.Vault, error) { return l.rows, l.fails }

func (l *theList) GetLastVault() (domain.Vault, bool, error) {
	l.requested++
	return l.last, l.isKept, l.fails
}

func (l *theList) Save(domain.Vault) error                      { return nil }
func (l *theList) Remove(domain.VaultID) error                  { return nil }
func (l *theList) RecordOpened(domain.VaultID) error            { return nil }
func (l *theList) FindVault(string) (domain.Vault, bool, error) { return domain.Vault{}, false, nil }

// folders are the vaults whose folder can be opened. Every other path answers
// the way a folder that has gone does.
type folders map[domain.VaultID]bool

func (f folders) Open(v domain.Vault) (vaultport.VaultReader, error) {
	if f[v.ID] {
		return nil, nil
	}
	return nil, fs.ErrNotExist
}

// two vaults, both of whose folders exist where the list says.
func twoRows() (*theList, folders) {
	return &theList{rows: []domain.Vault{
		{ID: "wren", Name: "Wren", Path: "/notes/wren"},
		{ID: "teal", Name: "Teal", Path: "/notes/teal"},
	}}, folders{"wren": true, "teal": true}
}

// Every vault on the list is answered with, in the order the list gives them.
func TestEveryVaultOnTheListIsKnown(t *testing.T) {
	t.Parallel()
	list, there := twoRows()

	recognized, err := vaults.NewGetKnownVaults(list, there).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 2 {
		t.Fatalf("%d vaults of a list of two", len(recognized))
	}
	if recognized[0].Vault.ID != "wren" || recognized[1].Vault.ID != "teal" {
		t.Errorf("the list came back as %s then %s", recognized[0].Vault.ID, recognized[1].Vault.ID)
	}
	for _, one := range recognized {
		if one.IsMissing {
			t.Errorf("%s is where the list says it is and was called missing", one.Vault.ID)
		}
	}
}

// When no vault is specified as being worked, the last-opened vault is the
// current one: it is the vault the next window opens.
func TestTheVaultOpenedLastIsTheCurrentOne(t *testing.T) {
	t.Parallel()
	list, there := twoRows()
	list.last, list.isKept = list.rows[1], true

	recognized, err := vaults.NewGetKnownVaults(list, there).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if recognized[1].IsCurrent {
		t.Error("a vault that was not opened last was called the current one")
	}
	if !recognized[0].IsCurrent {
		t.Error("the vault opened last is not the current one")
	}
}

// The current vault comes first and the rest follow as the list holds them.
func TestTheCurrentVaultReachesFirst(t *testing.T) {
	t.Parallel()
	all := []domain.Vault{{ID: "one"}, {ID: "two"}, {ID: "three"}}

	for name, c := range map[string]struct {
		last domain.VaultID
		want []string
	}{
		"one in the middle":    {"two", []string{"two", "one", "three"}},
		"the last of them":     {"three", []string{"three", "one", "two"}},
		"the first of them":    {"one", []string{"one", "two", "three"}},
		"none opened yet":      {"", []string{"one", "two", "three"}},
		"one the list dropped": {"four", []string{"one", "two", "three"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			list := &theList{rows: all, last: domain.Vault{ID: c.last}, isKept: c.last != ""}
			recognized, err := vaults.NewGetKnownVaults(list, folders{}).Execute("")
			if err != nil {
				t.Fatal(err)
			}

			got := make([]string, 0, len(all))
			for _, one := range recognized {
				got = append(got, string(one.Vault.ID))
			}
			if !slices.Equal(got, c.want) {
				t.Fatalf("listed %v, want %v", got, c.want)
			}
		})
	}
}

// A caller working in a specific vault names it, and that takes precedence over what the list remembers.
func TestTheVaultBeingWorkedExistsOverWhatTheListRemembers(t *testing.T) {
	t.Parallel()
	list, there := twoRows()
	list.last, list.isKept = list.rows[1], true

	recognized, err := vaults.NewGetKnownVaults(list, there).Execute("wren")
	if err != nil {
		t.Fatal(err)
	}
	if !recognized[0].IsCurrent || recognized[1].IsCurrent {
		t.Error("the vault being worked is not the current one")
	}
	if list.requested != 0 {
		t.Errorf("the list was asked what was opened last %d times", list.requested)
	}
}

// Nothing is current when the list has never been opened and no vault is
// specified as being worked.
func TestAListNobodyHasOpenedHasNoCurrentVault(t *testing.T) {
	t.Parallel()
	list, there := twoRows()

	recognized, err := vaults.NewGetKnownVaults(list, there).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range recognized {
		if one.IsCurrent {
			t.Errorf("%s is current on a list nobody has opened", one.Vault.ID)
		}
	}
}

// A folder that cannot be read is marked missing, and the vault stays on the
// list: it is removed only by an explicit request.
func TestAVaultWhoseFolderHasMissingIsMarkedAndStays(t *testing.T) {
	t.Parallel()
	list, _ := twoRows()

	recognized, err := vaults.NewGetKnownVaults(list, folders{"wren": true}).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 2 {
		t.Fatalf("%d vaults after one folder went", len(recognized))
	}
	if recognized[0].IsMissing {
		t.Error("a folder that is there was called missing")
	}
	if !recognized[1].IsMissing {
		t.Error("a folder that has gone was not called missing")
	}
}

// A build with no way to open a vault marks no folder missing, since it never
// checked, and a vault wrongly marked missing tells the user their notes have
// vanished.
func TestABuildThatCannotOpenAVaultMarksNoFolderMissing(t *testing.T) {
	t.Parallel()
	list, _ := twoRows()

	recognized, err := vaults.NewGetKnownVaults(list, nil).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range recognized {
		if one.IsMissing {
			t.Errorf("%s was called missing by a build that cannot open a vault", one.Vault.ID)
		}
	}
}

// An empty list contains no vaults, and the last-opened vault is not queried:
// there is nothing for the result to describe.
func TestAnEmptyListIsNoVaults(t *testing.T) {
	t.Parallel()
	list := &theList{}

	recognized, err := vaults.NewGetKnownVaults(list, folders{}).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 0 {
		t.Errorf("%d vaults on an empty list", len(recognized))
	}
	if list.requested != 0 {
		t.Errorf("the list was asked what was opened last %d times", list.requested)
	}
}

// A list that cannot be read is the answer, and no half of one is given.
func TestAListThatCannotBeReadIsTheAnswer(t *testing.T) {
	t.Parallel()
	broken := errors.New("the list is not readable")
	list := &theList{rows: []domain.Vault{{ID: "wren"}}, fails: broken}

	recognized, err := vaults.NewGetKnownVaults(list, folders{}).Execute("")
	if !errors.Is(err, broken) {
		t.Fatalf("got %v", err)
	}
	if recognized != nil {
		t.Errorf("%d vaults came back with the failure", len(recognized))
	}
}

package vault_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// theList is the vaults an installation holds, held in memory: the rows, which
// of them was opened last, and how often that was asked.
type theList struct {
	rows   []domain.Vault
	last   domain.Vault
	isKept bool
	asked  int
	fails  error
}

func (l *theList) List() ([]domain.Vault, error) { return l.rows, l.fails }

func (l *theList) Last() (domain.Vault, bool, error) {
	l.asked++
	return l.last, l.isKept, l.fails
}

func (l *theList) Save(domain.Vault) error                 { return nil }
func (l *theList) Remove(domain.VaultID) error             { return nil }
func (l *theList) RecordOpened(domain.VaultID) error       { return nil }
func (l *theList) Find(string) (domain.Vault, bool, error) { return domain.Vault{}, false, nil }

// folders are the vaults whose folder can be opened. Every other path answers
// the way a folder that has gone does.
type folders map[domain.VaultID]bool

func (f folders) Open(v domain.Vault) (port.VaultReader, error) {
	if f[v.ID] {
		return nil, nil
	}
	return nil, fs.ErrNotExist
}

// two vaults, both of whose folders are where the list says they are.
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

	known, err := vaults.NewKnownVaults(list, there).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 2 {
		t.Fatalf("%d vaults of a list of two", len(known))
	}
	if known[0].Vault.ID != "wren" || known[1].Vault.ID != "teal" {
		t.Errorf("the list came back as %s then %s", known[0].Vault.ID, known[1].Vault.ID)
	}
	for _, one := range known {
		if one.IsMissing {
			t.Errorf("%s is where the list says it is and was called missing", one.Vault.ID)
		}
	}
}

// Where nobody says which vault is being worked, the one opened last is the
// current one: that is the vault the next window opens.
func TestTheVaultOpenedLastIsTheCurrentOne(t *testing.T) {
	t.Parallel()
	list, there := twoRows()
	list.last, list.isKept = list.rows[1], true

	known, err := vaults.NewKnownVaults(list, there).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if known[0].IsCurrent {
		t.Error("a vault that was not opened last was called the current one")
	}
	if !known[1].IsCurrent {
		t.Error("the vault opened last is not the current one")
	}
}

// A caller sitting in front of a vault says which, and what the list remembers
// is not asked: the window is showing one vault whatever was opened last.
func TestTheVaultBeingWorkedStandsOverWhatTheListRemembers(t *testing.T) {
	t.Parallel()
	list, there := twoRows()
	list.last, list.isKept = list.rows[1], true

	known, err := vaults.NewKnownVaults(list, there).Execute("wren")
	if err != nil {
		t.Fatal(err)
	}
	if !known[0].IsCurrent || known[1].IsCurrent {
		t.Error("the vault being worked is not the current one")
	}
	if list.asked != 0 {
		t.Errorf("the list was asked what was opened last %d times", list.asked)
	}
}

// Nothing is current where the list has never been opened and nobody says which
// vault is being worked.
func TestAListNobodyHasOpenedHasNoCurrentVault(t *testing.T) {
	t.Parallel()
	list, there := twoRows()

	known, err := vaults.NewKnownVaults(list, there).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range known {
		if one.IsCurrent {
			t.Errorf("%s is current on a list nobody has opened", one.Vault.ID)
		}
	}
}

// A folder that is not there to be read is marked, and the vault stays on the
// list: it is forgotten when somebody says so and not before.
func TestAVaultWhoseFolderHasGoneIsMarkedAndStays(t *testing.T) {
	t.Parallel()
	list, _ := twoRows()

	known, err := vaults.NewKnownVaults(list, folders{"wren": true}).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 2 {
		t.Fatalf("%d vaults after one folder went", len(known))
	}
	if known[0].IsMissing {
		t.Error("a folder that is there was called missing")
	}
	if !known[1].IsMissing {
		t.Error("a folder that has gone was not called missing")
	}
}

// A build with nothing to open a vault through marks no folder missing. It
// never looked, and every vault marked gone is a person told their notes have
// vanished.
func TestABuildThatCannotOpenAVaultMarksNoFolderMissing(t *testing.T) {
	t.Parallel()
	list, _ := twoRows()

	known, err := vaults.NewKnownVaults(list, nil).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range known {
		if one.IsMissing {
			t.Errorf("%s was called missing by a build that cannot open a vault", one.Vault.ID)
		}
	}
}

// An empty list is no vaults, and what was opened last is not asked: there is
// nothing for the answer to be about.
func TestAnEmptyListIsNoVaults(t *testing.T) {
	t.Parallel()
	list := &theList{}

	known, err := vaults.NewKnownVaults(list, folders{}).Execute("")
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 0 {
		t.Errorf("%d vaults on an empty list", len(known))
	}
	if list.asked != 0 {
		t.Errorf("the list was asked what was opened last %d times", list.asked)
	}
}

// A list that cannot be read is the answer, and no half of one is given.
func TestAListThatCannotBeReadIsTheAnswer(t *testing.T) {
	t.Parallel()
	broken := errors.New("the list is not readable")
	list := &theList{rows: []domain.Vault{{ID: "wren"}}, fails: broken}

	known, err := vaults.NewKnownVaults(list, folders{}).Execute("")
	if !errors.Is(err, broken) {
		t.Fatalf("got %v", err)
	}
	if known != nil {
		t.Errorf("%d vaults came back with the failure", len(known))
	}
}

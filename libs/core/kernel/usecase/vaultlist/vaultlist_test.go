package vaultlist_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vaultlist"
	"github.com/jiva-studio/numen/libs/note/domain"
)

var errRefused = errors.New("refused")

func newHarbor() domain.Vault {
	return domain.Vault{ID: "v-harbor", Name: "harbor", Path: "/shelf/harbor"}
}
func newOrchard() domain.Vault {
	return domain.Vault{ID: "v-orchard", Name: "orchard", Path: "/shelf/orchard"}
}

type vaults []domain.Vault

func (v vaults) Execute() ([]domain.Vault, error) { return v, nil }

type find map[string]domain.Vault

func (f find) Execute(id string) (domain.Vault, error) {
	if v, ok := f[id]; ok {
		return v, nil
	}
	return domain.Vault{}, errRefused
}

type gone map[domain.VaultID]bool

func (g gone) Execute(v domain.Vault) bool { return g[v.ID] }

type adds struct{ fail bool }

func (a adds) Execute(path, name string) (domain.Vault, error) {
	if a.fail {
		return domain.Vault{}, errRefused
	}
	return domain.Vault{ID: "v-new", Name: name, Path: path}, nil
}

type renames struct{}

func (renames) Execute(_ context.Context, v domain.Vault, name string) (domain.Vault, error) {
	v.Name = name
	return v, nil
}

type screen struct {
	calls []domain.Vault
	fail  error
}

func (s *screen) ShowVault(_ context.Context, v domain.Vault) error {
	s.calls = append(s.calls, v)
	return s.fail
}

func TestTheListReportsWhichVaultFoldersAreMissing(t *testing.T) {
	got, err := vaultlist.NewListVaults(vaults{newHarbor(), newOrchard()}, gone{newOrchard().ID: true}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].IsMissing || !got[1].IsMissing || got[1].Name != "orchard" {
		t.Errorf("listed %+v, want only the orchard marked missing", got)
	}
}

func TestAVaultAddedIsAnsweredWithTheStateOfItsFolder(t *testing.T) {
	got, err := vaultlist.NewAddVault(adds{}, gone{"v-new": true}).Execute("/shelf/new", "new")
	if err != nil || got.Path != "/shelf/new" || !got.IsMissing {
		t.Errorf("added %+v, %v", got, err)
	}
	if _, err := vaultlist.NewAddVault(adds{fail: true}, gone{}).Execute("/shelf/new", "new"); !errors.Is(err, errRefused) {
		t.Errorf("a refused add answered %v", err)
	}
}

func TestARenameFindsTheVaultThenNamesIt(t *testing.T) {
	rename := vaultlist.NewRenameVault(find{string(newHarbor().ID): newHarbor()}, renames{}, gone{})
	got, err := rename.Execute(t.Context(), string(newHarbor().ID), "port")
	if err != nil || got.Name != "port" || got.Path != newHarbor().Path {
		t.Errorf("renamed %+v, %v", got, err)
	}
	if _, err := rename.Execute(t.Context(), "nobody", "port"); !errors.Is(err, errRefused) {
		t.Errorf("an unknown vault answered %v", err)
	}
}

func TestOpeningShowsTheVaultFoundAndNothingForAnUnknownOne(t *testing.T) {
	display := &screen{}
	open := vaultlist.NewOpenVault(find{string(newOrchard().ID): newOrchard()}, display)
	if err := open.Execute(t.Context(), string(newOrchard().ID)); err != nil || len(display.calls) != 1 || display.calls[0].ID != newOrchard().ID {
		t.Errorf("showed %v, %v", display.calls, err)
	}
	if err := open.Execute(t.Context(), "nobody"); !errors.Is(err, errRefused) || len(display.calls) != 1 {
		t.Errorf("an unknown vault answered %v and showed %v", err, display.calls)
	}
	display.fail = errRefused
	if err := open.Execute(t.Context(), string(newOrchard().ID)); !errors.Is(err, errRefused) {
		t.Errorf("a window that refused answered %v", err)
	}
}

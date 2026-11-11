package registry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
)

type shelf struct {
	vaults   []domain.Vault
	recorded []domain.VaultID
	added    []string
	failure  error
}

func (s *shelf) GetName(root string) string {
	if root == "/link/orchard" {
		return "/real/orchard"
	}
	return root
}

func (s *shelf) CheckReadable(string) error { return nil }

func (s *shelf) EnsureIdentity(string, time.Time) (domain.VaultID, error) { return "", nil }

func (s *shelf) GetVaultID(string) (domain.VaultID, bool, error) { return "", false, nil }

func (s *shelf) FindVault(path string) (domain.Vault, bool, error) {
	for _, v := range s.vaults {
		if v.Path == path {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, s.failure
}

func (s *shelf) Execute(root, name string) (domain.Vault, error) {
	s.added = append(s.added, root+"|"+name)
	v := domain.Vault{ID: "V1", Name: name, Path: root}
	s.vaults = append(s.vaults, v)
	return v, nil
}

func (s *shelf) RecordOpened(id domain.VaultID) error {
	s.recorded = append(s.recorded, id)
	return nil
}

func (s *shelf) ensure() registry.EnsureRegistered {
	return registry.NewEnsureRegistered(s, s, s, s)
}

func TestAFolderOnTheListIsReturnedWithoutAddingOrRecordingIt(t *testing.T) {
	list := &shelf{vaults: []domain.Vault{{ID: "V0", Name: "Orchard", Path: "/real/orchard"}}}

	got, err := list.ensure().Execute(registry.EnsureRequest{Root: "/link/orchard", Name: "other", IsRecordedOpened: true})

	if err != nil || got.ID != "V0" || len(list.added) != 0 || len(list.recorded) != 0 {
		t.Fatalf("got %+v (%v), added %v, recorded %v", got, err, list.added, list.recorded)
	}
}

func TestAFolderNotOnTheListIsAddedUnderItsResolvedPath(t *testing.T) {
	list := &shelf{}

	got, err := list.ensure().Execute(registry.EnsureRequest{Root: "/link/orchard", Name: "numen"})

	if err != nil || got.Path != "/real/orchard" || len(list.added) != 1 || list.added[0] != "/real/orchard|numen" {
		t.Fatalf("got %+v (%v), added %v", got, err, list.added)
	}
	if len(list.recorded) != 0 {
		t.Errorf("recorded %v, want none", list.recorded)
	}
}

func TestAnAddedVaultIsRecordedAsOpenedWhenAsked(t *testing.T) {
	list := &shelf{}

	if _, err := list.ensure().Execute(registry.EnsureRequest{Root: "/vaults/pond", IsRecordedOpened: true}); err != nil {
		t.Fatal(err)
	}

	if len(list.recorded) != 1 || list.recorded[0] != "V1" {
		t.Errorf("recorded %v", list.recorded)
	}
}

func TestAFailedLookupAddsNothing(t *testing.T) {
	list := &shelf{failure: errors.New("the list cannot be read")}

	_, err := list.ensure().Execute(registry.EnsureRequest{Root: "/vaults/pond"})

	if err == nil || len(list.added) != 0 {
		t.Errorf("said %v, added %v", err, list.added)
	}
}

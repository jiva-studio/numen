package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func vaultsAt(paths ...string) domain.Vaults {
	vs := domain.Vaults{}
	for i, p := range paths {
		vs = append(vs, domain.Vault{ID: domain.VaultID(string(rune('a' + i))), Name: "Vault " + p, Path: p})
	}
	return vs
}

func TestCheckOverlap_FolderInsideVaultFails(t *testing.T) {
	err := vaultsAt("/notes").CheckOverlap("/notes/deep")

	if !errors.Is(err, domain.ErrOverlaps) {
		t.Fatalf("CheckOverlap error = %v, want ErrOverlaps", err)
	}
	if got := err.Error(); !strings.Contains(got, "Vault /notes") || !strings.Contains(got, "is inside") {
		t.Errorf("CheckOverlap error = %q, want it to name the vault it lies inside", got)
	}
}

func TestCheckOverlap_FolderContainingVaultFails(t *testing.T) {
	err := vaultsAt("/notes/deep").CheckOverlap("/notes")

	if !errors.Is(err, domain.ErrOverlaps) {
		t.Fatalf("CheckOverlap error = %v, want ErrOverlaps", err)
	}
	if got := err.Error(); !strings.Contains(got, "Vault /notes/deep") || !strings.Contains(got, "holds") {
		t.Errorf("CheckOverlap error = %q, want it to name the vault it holds", got)
	}
}

func TestCheckOverlap_SiblingFolderPasses(t *testing.T) {
	vs := vaultsAt("/notes", "/notes-two", "/other/notes")

	if err := vs.CheckOverlap("/books"); err != nil {
		t.Errorf("CheckOverlap(/books) = %v, want room", err)
	}
	if err := vs.CheckOverlap("/notes-three"); err != nil {
		t.Errorf("CheckOverlap(/notes-three) = %v, want room", err)
	}
}

func TestAVaultsOwnFolderIsNotInItsWay(t *testing.T) {
	if err := vaultsAt("/notes").CheckOverlap("/notes"); err != nil {
		t.Errorf("CheckOverlap(/notes) = %v, want room", err)
	}
}

func TestAnOverlapIsFoundPastTheVaultsBeforeIt(t *testing.T) {
	err := vaultsAt("/a", "/b", "/c").CheckOverlap("/c/inner")

	if !errors.Is(err, domain.ErrOverlaps) {
		t.Fatalf("CheckOverlap error = %v, want ErrOverlaps", err)
	}
}

func TestAFolderWhosePathIsNotRelatedToAVaultIsNotBelowIt(t *testing.T) {
	if err := vaultsAt("/notes").CheckOverlap("relative/dir"); err != nil {
		t.Errorf("CheckOverlap(relative/dir) = %v, want room", err)
	}
	if err := vaultsAt("/notes").CheckOverlap("/notes/..hidden"); !errors.Is(err, domain.ErrOverlaps) {
		t.Errorf("CheckOverlap(/notes/..hidden) = %v, want ErrOverlaps: a name starting with two dots is inside", err)
	}
}

func TestAVaultIsFoundByNameWhateverTheCase(t *testing.T) {
	vs := domain.Vaults{
		{ID: "one", Name: "Journal", Path: "/j"},
		{ID: "two", Name: "Work", Path: "/w"},
	}

	got, ok := vs.FindByName("WORK", "")
	if !ok || got.ID != "two" {
		t.Errorf("FindByName(WORK) = %+v, %v; want vault two", got, ok)
	}
	if got, ok := vs.FindByName("Play", ""); ok || got != (domain.Vault{}) {
		t.Errorf("FindByName(Play) = %+v, %v; want nothing", got, ok)
	}
}

func TestTheVaultBeingNamedIsNotAnAnswer(t *testing.T) {
	vs := domain.Vaults{
		{ID: "one", Name: "Journal", Path: "/j"},
		{ID: "two", Name: "journal", Path: "/k"},
	}

	if got, ok := vs.FindByName("Journal", "one"); !ok || got.ID != "two" {
		t.Errorf("FindByName(Journal, one) = %+v, %v; want vault two", got, ok)
	}
	if got, ok := vs.FindByName("Journal", "two"); !ok || got.ID != "one" {
		t.Errorf("FindByName(Journal, two) = %+v, %v; want vault one", got, ok)
	}
	if got, ok := (domain.Vaults{vs[0]}).FindByName("Journal", "one"); ok {
		t.Errorf("FindByName(Journal, one) = %+v, true; want nothing", got)
	}
}

func TestAFreeNameIsCachedAndATakenOneIsNumberedFromTwo(t *testing.T) {
	vs := domain.Vaults{
		{ID: "one", Name: "Notes", Path: "/n"},
		{ID: "two", Name: "Notes 2", Path: "/n2"},
		{ID: "three", Name: "Notes 4", Path: "/n4"},
	}

	if got := vs.FreeName("Books", ""); got != "Books" {
		t.Errorf("FreeName(Books) = %q, want Books", got)
	}
	if got := vs.FreeName("notes", ""); got != "notes 3" {
		t.Errorf("FreeName(notes) = %q, want notes 3", got)
	}
	if got := (domain.Vaults{vs[0]}).FreeName("Notes", ""); got != "Notes 2" {
		t.Errorf("FreeName(Notes) = %q, want Notes 2", got)
	}
	if got := vs.FreeName("Notes", "one"); got != "Notes" {
		t.Errorf("FreeName(Notes, one) = %q, want Notes", got)
	}
}

func TestTheSameFolderSavedTwoWaysIsNotInsideItself(t *testing.T) {
	if err := vaultsAt("/notes/").CheckOverlap("/notes"); err != nil {
		t.Errorf("CheckOverlap(/notes) = %v, want room", err)
	}
}

func TestAnOverlapIsFoundAfterTheVaultAtTheSamePath(t *testing.T) {
	err := vaultsAt("/x/y", "/x").CheckOverlap("/x/y")

	if !errors.Is(err, domain.ErrOverlaps) {
		t.Fatalf("CheckOverlap error = %v, want ErrOverlaps", err)
	}
}

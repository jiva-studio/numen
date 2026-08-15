package vault_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// refreshing is a vault already scanned once, and the use case that brings named
// notes up to date afterwards.
func refreshing(t *testing.T, notes map[string]string) (usecase.Refresh, *container.Index, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db := openIndex(t)
	if _, err := scanner(filesystem.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return usecase.Refresh{Readers: filesystem.Readers{}, Notes: db.Notes()}, db, v
}

func titles(t *testing.T, db *container.Index, v domain.Vault, query string) []string {
	t.Helper()
	matches, err := db.Queries().Search(t.Context(), v.ID, query, 10)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.Title)
	}
	slices.Sort(out)
	return out
}

// TestARefreshedNoteIsWhatIsOnDisk.
func TestARefreshedNoteIsWhatIsOnDisk(t *testing.T) {
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	if err := os.WriteFile(filepath.Join(v.Path, "Note.md"),
		[]byte("---\ntitle: Renamed\n---\n\n# Renamed\n\nentropy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := refresh.Execute(t.Context(), v, []string{"Note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Indexed, []string{"Note.md"}) {
		t.Errorf("indexed %v", res.Indexed)
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Renamed"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestADeletedNoteLeavesTheIndex. The vault is authoritative: what is not on
// disk is not in the index, and a note that stays behind is still searchable
// and opens nothing.
func TestADeletedNoteLeavesTheIndex(t *testing.T) {
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md":  "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n\nentropy\n",
	})

	if err := os.Remove(filepath.Join(v.Path, "Note.md")); err != nil {
		t.Fatal(err)
	}
	res, err := refresh.Execute(t.Context(), v, []string{"Note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Removed, []string{"Note.md"}) {
		t.Errorf("removed %v", res.Removed)
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Other"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestANoteThatVanishesMidReadKeepsItsRow. A file that is briefly absent is
// what an editor saving through a temporary file looks like, and the save that
// follows arrives as its own event.
func TestANoteThatVanishesMidReadKeepsItsRow(t *testing.T) {
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	refresh.Readers = vanishingReaders{VaultReaders: filesystem.Readers{}, gone: "Note.md"}
	res, err := refresh.Execute(t.Context(), v, []string{"Note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 0 {
		t.Errorf("removed %v — it was there when it was looked at", res.Removed)
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Note"}) {
		t.Errorf("the index holds %v", got)
	}
}

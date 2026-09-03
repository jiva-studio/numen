package index

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A vault's row is what every other row points at, and registering one is how a
// walk makes sure it is there. What the vault is called and where it is are the
// list's, and Save is what writes them.
func TestRegisteringAVaultLeavesTheNameAndThePathItHas(t *testing.T) {
	db := opened(t)
	vaults := db.Vaults()

	renamed := domain.Vault{ID: first.ID, Name: "journal", Path: "/moved/journal"}
	if err := vaults.Save(t.Context(), renamed); err != nil {
		t.Fatal(err)
	}
	if err := vaults.Register(t.Context(),
		domain.Vault{ID: first.ID, Name: "first", Path: "/first"}); err != nil {
		t.Fatal(err)
	}

	name, path := calledAt(t, db, string(first.ID))
	if name != renamed.Name || path != renamed.Path {
		t.Errorf("the index holds %q at %s, want %q at %s", name, path, renamed.Name, renamed.Path)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM vaults WHERE identifier = ?`, string(first.ID)); got != 1 {
		t.Errorf("the vault has %d rows", got)
	}
}

// A vault the index has never seen is given a row by the walk that finds it.
func TestRegisteringAVaultTheIndexDoesNotHoldWritesIt(t *testing.T) {
	db := opened(t)
	fresh := domain.Vault{ID: "01THIRD", Name: "third", Path: "/third"}

	if err := db.Vaults().Register(t.Context(), fresh); err != nil {
		t.Fatal(err)
	}

	name, path := calledAt(t, db, string(fresh.ID))
	if name != fresh.Name || path != fresh.Path {
		t.Errorf("the index holds %q at %s, want %q at %s", name, path, fresh.Name, fresh.Path)
	}
}

// calledAt is what the index says a vault is called and where it is.
func calledAt(t *testing.T, db *DB, id string) (name, path string) {
	t.Helper()
	if err := db.read.QueryRowContext(t.Context(),
		`SELECT name, path FROM vaults WHERE identifier = ?`, id).Scan(&name, &path); err != nil {
		t.Fatal(err)
	}
	return name, path
}

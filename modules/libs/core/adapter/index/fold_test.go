package index

import (
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// The driver takes a name once for the whole process, so opening a second
// index must not try to take it again. Registering where the package is
// imported hides this: nothing else can ask for the registration at all.
func TestTheNameIsRegisteredOnceHoweverManyIndexesAreOpened(t *testing.T) {
	if err := foldsNames(); err != nil {
		t.Fatal(err)
	}
	if err := foldsNames(); err != nil {
		t.Fatalf("registering the name a second time: %v", err)
	}
}

// What SQL computes under the name is what the scan writes, and an index is
// what makes it available.
func TestSQLFoldsANameTheWayTheScanDoes(t *testing.T) {
	db, err := Open(t.Context(), filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	for _, name := range []string{"Ångström", "STRAßE", "İstanbul"} {
		var got string
		if err := db.read.QueryRowContext(t.Context(), "SELECT numen_fold(?)", name).Scan(&got); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if want := domain.FoldName(name); got != want {
			t.Errorf("SQL folded %q to %q, and the scan writes %q", name, got, want)
		}
	}
}

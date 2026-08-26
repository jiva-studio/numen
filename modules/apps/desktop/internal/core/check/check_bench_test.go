package check_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/check"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// BenchmarkRun is what opening the problems view costs.
//
// Two of these checks read rows a scan already stored and are a query each. Two
// work the whole vault out at the moment they are asked, and this is the number
// that says whether that stays affordable — a vault where every note answers to
// one of a handful of names is the shape that costs the most, because every
// link written by one of them is a candidate.
func BenchmarkRun(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			l, v := scanned(b, vaultOf(b, notes))
			b.ResetTimer()
			for range b.N {
				if _, err := l.Run(b.Context(), v); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRunDangling is the quiet check on its own: one query, no resolving.
func BenchmarkRunDangling(b *testing.B) {
	l, v := scanned(b, vaultOf(b, 10_000))
	b.ResetTimer()
	for range b.N {
		if _, err := l.Run(b.Context(), v, domain.CheckDangling); err != nil {
			b.Fatal(err)
		}
	}
}

// vaultOf is a vault where names collide and links are written by them, which
// is what the ambiguity check has to work through.
func vaultOf(b *testing.B, notes int) map[string]string {
	b.Helper()
	out := make(map[string]string, notes)
	for i := range notes {
		// Ten names for the whole vault, in a hundred folders: every name
		// answers for many notes, and every link written by one is ambiguous.
		name := fmt.Sprintf("Note-%d", i%10)
		out[fmt.Sprintf("f%02d/%s-%d.md", i%100, name, i)] = fmt.Sprintf(
			"---\nlinks:\n  - to: %s\n    role: parent\n---\n# %s\n\nsomething to index.\n",
			name, name)
	}
	return out
}

func scanned(b *testing.B, notes map[string]string) (check.Checks, domain.Vault) {
	b.Helper()
	v := testsupport.NewVault(b, notes)
	db, err := container.Config{
		IndexPath: filepath.Join(b.TempDir(), "index.db"),
	}.OpenIndex(b.Context())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })

	scan := usecase.Scan{
		Readers: filesystem.Readers{}, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.Queries(), Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(b.Context(), v); err != nil {
		b.Fatal(err)
	}
	return check.Standard(db.Problems()), v
}

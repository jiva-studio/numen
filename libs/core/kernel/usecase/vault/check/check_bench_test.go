package check_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// BenchmarkRun is what opening the problems view costs.
// Two checks read rows a scan already stored; two work out the whole vault when
// they run, and this number shows whether that stays affordable. A vault where
// every note has one of a handful of names is the costliest shape, because every
// link written by one of them is a candidate.
func BenchmarkRun(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			l, v := scanBenchVault(b, vaultOf(b, notes))
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
	l, v := scanBenchVault(b, vaultOf(b, 10_000))
	b.ResetTimer()
	for range b.N {
		if _, err := l.Run(b.Context(), v, domain.CheckDangling); err != nil {
			b.Fatal(err)
		}
	}
}

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

func scanBenchVault(b *testing.B, notes map[string]string) (check.Checks, domain.Vault) {
	b.Helper()
	v := vaultfixture.NewVault(b, notes)
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(b.TempDir(), "index.db")},
	}), b.Context())
	if err != nil {
		b.Fatal(err)
	}
	testsupport.CloseAfter(b, db)

	scan := vaults.NewScan(
		vaultreader.Readers{}, db.Vaults(), db.Notes(),
		db.NoteQueries(), db.Maintenance(), nil,
	)
	if _, err := scan.Execute(b.Context(), v); err != nil {
		b.Fatal(err)
	}
	return check.Standard(db.NoteProblems()), v
}

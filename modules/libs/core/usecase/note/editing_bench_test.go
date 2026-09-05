package note_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// unlevelled brings nothing level. What is measured here is the file being
// replaced; what the index then costs is a refresh, and is measured as one.
func unlevelled(context.Context, domain.Vault, []string) error { return nil }

// What these measure: what a person waits for between stopping typing and their
// note being on disk, and what a window pays per open tab every time the vault
// changes. The numbers live in docs/performance.md.
//
//	go test ./usecase/note/ -run XXX -bench . -benchtime 100x

// BenchmarkRead is one tab opening a note, and one tab reading it again after a
// change. A window pays this once per clean tab per change that names its path.
func BenchmarkRead(b *testing.B) {
	for _, notes := range []int{1_000, 100_000} {
		b.Run(fmt.Sprintf("%d notes", notes), func(b *testing.B) {
			v := testsupport.GenerateVault(b, notes)
			read := note.Read{Readers: filesystem.VaultReaders{}}
			path := "01/note-000001.md"

			b.ResetTimer()
			for range b.N {
				found, err := read.Execute(b.Context(), v, path)
				if err != nil {
					b.Fatal(err)
				}
				if found.Outcome != note.Ok {
					b.Fatalf("%s: %s", path, found.Outcome)
				}
			}
		})
	}
}

// BenchmarkSave is the write a quiet interval fires: the file read for its
// frontmatter, the body put on it, and the whole replaced under the vault's
// write lock.
func BenchmarkSave(b *testing.B) {
	for _, size := range []int{500, 5_000} {
		b.Run(fmt.Sprintf("%d words", size), func(b *testing.B) {
			v := testsupport.GenerateVault(b, 100)
			write := note.NewWrite(
				filesystem.VaultReaders{}, filesystem.VaultWriters{}, unlevelled, time.Now)
			path := "01/note-000001.md"

			body := ""
			for i := range size {
				body += fmt.Sprintf("word%d ", i)
			}

			b.ResetTimer()
			for i := range b.N {
				// Every save writes something different, which is what a person
				// typing does and what the index has to notice.
				if _, err := write.Save(b.Context(), v, path, fmt.Sprintf("%s\n\n%d\n", body, i), nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

package note_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

func indexNothing(context.Context, domain.Vault, []string) error { return nil }

// These benchmarks measure the delay between the end of typing and the note
// being written to disk, and the cost per open tab each time the vault changes.
// The numbers live in docs/performance.md.
//
//	go test ./usecase/note/ -run XXX -bench . -benchtime 100x

// BenchmarkRead is one tab opening a note, and one tab reading it again after a
// change. A window pays this once per clean tab per change that names its path.
func BenchmarkRead(b *testing.B) {
	for _, notes := range []int{1_000, 100_000} {
		b.Run(fmt.Sprintf("%d notes", notes), func(b *testing.B) {
			v := vaultfixture.GenerateVault(b, notes)
			read := note.NewRead(vaultreader.Readers{})
			path := "01/note-000001.md"

			b.ResetTimer()
			for range b.N {
				found, err := read.Execute(b.Context(), v, path)
				if err != nil {
					b.Fatal(err)
				}
				if found.Outcome != edit.Ok {
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
			v := vaultfixture.GenerateVault(b, 100)
			write := note.NewWrite(
				vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, indexNothing, testsupport.NewClock())
			path := "01/note-000001.md"

			body := ""
			for i := range size {
				body += fmt.Sprintf("word%d ", i)
			}

			b.ResetTimer()
			for i := range b.N {
				// Every save writes different content, as typing does, so the index
				// detects a change.
				if _, err := write.Save(b.Context(), v, path, fmt.Sprintf("%s\n\n%d\n", body, i), nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

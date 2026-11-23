package file_test

import (
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

// BenchmarkTabKinds compares the tab kinds of a batch of notes taken from the index with the same
// taken from the files, over a generated vault of 3431 notes.
//
//	go test ./usecase/file/ -run XXX -bench BenchmarkTabKinds -benchtime 200x
func BenchmarkTabKinds(b *testing.B) {
	db, err := index.Open(b.Context(), indexfile.GetPath(b), index.DisableSync())
	if err != nil {
		b.Fatal(err)
	}
	testsupport.CloseAfter(b, db)
	v := vaulttest.GenerateVault(b, 3431)
	scan := indexing.NewScan(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), testsupport.EmptySourceBacklog{})
	if _, err := scan.Execute(b.Context(), v); err != nil {
		b.Fatal(err)
	}
	choose := newShippedChoice()
	fromIndex := indexnote.NewTabKinds(db.NoteQueries(), choose)
	fromFiles := file.NewListFileKinds(file.NewGetFileKinds(vaultreader.Readers{}, db.NoteQueries()), choose)

	for _, size := range []int{13, 50} {
		paths := make([]string, size)
		for i := range paths {
			note := i * 53
			paths[i] = fmt.Sprintf("%02d/note-%06d.md", note%50, note)
		}
		b.Run(fmt.Sprintf("index/%d", size), func(b *testing.B) {
			for range b.N {
				if _, err := fromIndex.ListTabKinds(b.Context(), v, paths); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("files/%d", size), func(b *testing.B) {
			for range b.N {
				if _, err := fromFiles.Execute(b.Context(), v, paths); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

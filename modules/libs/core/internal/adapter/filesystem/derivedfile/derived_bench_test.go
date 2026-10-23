package derivedfile_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultconfig"
)

// opening is a store on a vault of one file, which is what the store operations
// below are measured on.
func opening(b *testing.B) *derivedfile.DerivedStore {
	b.Helper()
	root := b.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		b.Fatal(err)
	}
	derived, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, derivedfile.OCRDir)
	if err != nil {
		b.Fatal(err)
	}
	return derived
}

// What one name of the store costs. An answer written to the log is one append
// and a reading of the log is one read a file, so each of these is paid once
// per card answered and once per run file read.
func BenchmarkDerived(b *testing.B) {
	ctx := b.Context()
	line := []byte("one line of a run\n")

	b.Run("Append", func(b *testing.B) {
		derived := opening(b)
		for b.Loop() {
			if err := derived.Append(ctx, "ocr/run.txt", line); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Read", func(b *testing.B) {
		derived := opening(b)
		if err := derived.Write(ctx, "ocr/run.txt", line); err != nil {
			b.Fatal(err)
		}
		for b.Loop() {
			if _, err := derived.Read(ctx, "ocr/run.txt"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("List", func(b *testing.B) {
		derived := opening(b)
		if err := derived.Write(ctx, "ocr/run.txt", line); err != nil {
			b.Fatal(err)
		}
		for b.Loop() {
			if _, err := derived.List(ctx, derivedfile.OCRDir); err != nil {
				b.Fatal(err)
			}
		}
	})
}

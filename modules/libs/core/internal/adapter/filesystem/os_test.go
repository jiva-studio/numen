package filesystem_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
)

func makeBenchmarkVault(b *testing.B, depth, filesPerDir int) (string, *filesystem.VaultReader) {
	b.Helper()
	root := b.TempDir()
	var populate func(dir string, currentDepth int)
	populate = func(dir string, currentDepth int) {
		for i := 0; i < filesPerDir; i++ {
			name := filepath.Join(dir, fmt.Sprintf("note_%d_%d.md", currentDepth, i))
			if err := os.WriteFile(name, []byte("# Benchmark Note\nSome note content for testing.\n"), 0o644); err != nil {
				b.Fatal(err)
			}
		}
		if currentDepth < depth {
			for d := 0; d < 3; d++ {
				sub := filepath.Join(dir, fmt.Sprintf("folder_%d_%d", currentDepth, d))
				if err := os.Mkdir(sub, 0o755); err != nil {
					b.Fatal(err)
				}
				populate(sub, currentDepth+1)
			}
		}
	}
	populate(root, 1)

	reader, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		b.Fatal(err)
	}
	return root, reader
}

func BenchmarkWalk(b *testing.B) {
	_, reader := makeBenchmarkVault(b, 3, 20)
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var count int
		err := reader.Walk(ctx, func(_ domain.Fingerprint) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadFile(b *testing.B) {
	root := b.TempDir()
	content := []byte("# Benchmark Note\nThis is sample note content being read during filesystem benchmarks.\n")
	path := "sample.md"
	if err := os.WriteFile(filepath.Join(root, path), content, 0o644); err != nil {
		b.Fatal(err)
	}
	reader, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		b.Fatal(err)
	}
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		data, err := reader.Read(ctx, path)
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("empty data")
		}
	}
}

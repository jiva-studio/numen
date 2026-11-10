package indexport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// NoteScanner scans the notes of a vault.
type NoteScanner interface {
	ScanWithProgress(ctx context.Context, v domain.Vault, onProgress func(vaultreading.ScanResult)) (vaultreading.ScanResult, error)
}

// TextIndexer indexes the file texts of a vault.
type TextIndexer interface {
	IndexWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(vaultreading.IndexResult), isSkipped func(path string) bool,
	) (vaultreading.IndexResult, error)
	IndexOne(ctx context.Context, v domain.Vault, path string) (vaultreading.IndexResult, error)
}

// VectorEmbedder embeds the chunks of a vault.
type VectorEmbedder interface {
	EmbedWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(vaultreading.EmbedResult), isSkipped func(path, hash string) bool,
	) (vaultreading.EmbedResult, error)
}

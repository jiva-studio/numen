package vault

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
)

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the file texts cut, and the vectors made.
func NewReadWholeVault(notes vaultreading.NoteScanner, texts vaultreading.TextIndexer, vectors vaultreading.VectorEmbedder) vaultreading.ReadWholeVault {
	return vaultreading.NewReadWholeVault(notes, texts, vectors)
}

// ReadWholeVault is the three passes the composition root names: it may not import internal/vaultreading.
type ReadWholeVault = vaultreading.ReadWholeVault

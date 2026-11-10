package window

import "github.com/jiva-studio/numen/libs/note/domain"

// Change is what the window tells its clients about the vault it shows: the files that changed,
// the files that moved, or that everything must be read again.
type Change struct {
	Paths        []string
	ShouldReload bool
	Renamed      []domain.Move
	VersionPaths []string
}

// VaultState is what the window knows about the vault it shows: whether the first scan is done,
// what stopped it, and how far the vault has been read for meaning.
type VaultState struct {
	Vault         domain.Vault
	IsReady       bool
	Error         string
	WatchError    string
	IsEmbedding   bool
	ChunkCount    int64
	EmbeddedCount int64
}

// DetachedTab is a tab to open in a window of its own. The new window reads the tab by its id.
type DetachedTab struct {
	ID            string
	X, Y          int
	Width, Height int
}

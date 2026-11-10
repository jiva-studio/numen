package plugin

import "github.com/jiva-studio/numen/libs/note/domain"

// StoreFile is a file of a plugin's folder: its name, and how many bytes it holds.
type StoreFile struct {
	Name string
	Size int64
}

// IndexStatus is how far the index carries a vault.
type IndexStatus int

const (
	// Indexed is a vault the index carries.
	Indexed IndexStatus = iota + 1
	// Indexing is a vault being read into the index.
	Indexing
	// IndexFailed is a vault that could not be read into the index.
	IndexFailed
)

// IndexState is the status of a vault in the index, and the reason where it failed.
type IndexState struct {
	Status IndexStatus
	Reason string
}

// FileRange is a ranged read of a file: the text, its start in the file, and the file stat of
// the file it was read from.
type FileRange struct {
	Text  string
	Start int
	Stat  domain.FileStat
}

// LinkAddition is one link to write into the note at From, which the caller read as Baseline.
type LinkAddition struct {
	From     string
	Baseline domain.FileStat
	Link     domain.Link
}

// LinkAdded is the outcome of one addition: the note after the write, or the error of a link
// not written.
type LinkAdded struct {
	Stat domain.FileStat
	Err  error
}

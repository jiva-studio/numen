// Package ports declares what the use cases of the saved search require of the host.
package ports

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Search runs queries of the search language over the index of one vault.
type Search interface {
	StartSearch(ctx context.Context, start host.SearchStart) (string, error)
	GetSearch(ctx context.Context, id string, read host.SearchRead) (host.SearchState, error)
	DeleteSearch(ctx context.Context, id string) error
	ListSearchFunctions(ctx context.Context) (host.SearchFunctions, error)
}

// NoteLookup reads the titles and identities of the notes of one vault.
type NoteLookup interface {
	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
}

// Notes reads and writes the saved search notes of one vault.
type Notes interface {
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
	Scan(ctx context.Context, paths []string) error
	ListPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}

// Store is the plugin's folder of one vault.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	List(ctx context.Context) ([]host.StoreFile, error)
	Write(ctx context.Context, name string, content []byte) error
	Delete(ctx context.Context, name string) error
}

// Tasks reports the refresh to the host.
type Tasks interface {
	ReportTask(ctx context.Context, task host.Task) error
	FinishTask(ctx context.Context, id string) error
}

// Window names the vault the window shows.
type Window interface {
	GetVault(ctx context.Context) (domain.Vault, error)
}

// Clock is the time of the plugin.
type Clock interface {
	Now() time.Time
}

// Log records what a use case does in the background where no caller sees its error.
type Log interface {
	Debug(msg string, args ...any)
}

// Entropy fills the random bits of the identifiers a use case mints.
type Entropy interface {
	Fill(b []byte)
}

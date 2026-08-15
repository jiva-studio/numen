// Package port declares what the core needs from the world outside it.
//
// Every interface here is small and named after the need rather than after the
// technology that will satisfy it: the core asks for somewhere to read a vault
// from, not for a filesystem. Adapters are named after the technology, which is
// the only place it is allowed to appear.
package port

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// VaultReader is one vault as seen from outside: something that can be walked
// and read. The filesystem is one implementation; a fixture in memory is
// another.
type VaultReader interface {
	// Walk reports every file that is a candidate for indexing, in unspecified
	// order. The service folder is not reported.
	Walk(ctx context.Context, fn func(domain.FileRef) error) error
	// Read returns the bytes of one file, addressed by a path a walk reported.
	Read(ctx context.Context, path string) ([]byte, error)
}

// VaultReaders opens a reader for a given vault. A use case is handed this
// rather than a reader, because which vault it works on is decided while it
// runs, not when it is built.
type VaultReaders interface {
	Open(v domain.Vault) (VaultReader, error)
}

// VaultIdentity is what turns a folder into a vault: a folder can be looked at
// without being one, and becomes one when it is given an identity that stays
// with it.
type VaultIdentity interface {
	// Readable reports whether the folder can be read as a vault, writing
	// nothing. Looking and adding are different acts.
	Readable(root string) error
	// Ensure returns the identity the folder carries, creating one if it has
	// none. An existing identity is never replaced: it is what every row in the
	// index points at.
	Ensure(root string, at time.Time) (string, error)
}

// NoteRepository is where parsed notes are kept so they can be queried. Every
// method takes a vault: one database holds them all, and a query that forgets
// its vault silently returns another vault's notes.
type NoteRepository interface {
	RegisterVault(ctx context.Context, v domain.Vault) error
	// Known returns what is currently held about a vault, keyed by path, so a
	// scan can decide what to reparse without reading anything.
	Known(ctx context.Context, vaultID string) (map[string]domain.FileRef, error)
	Put(ctx context.Context, vaultID string, n domain.Note) error
	// Delete removes notes whose files are gone. The vault is authoritative:
	// what is not on disk is not in the index.
	Delete(ctx context.Context, vaultID string, paths []string) error
	Search(ctx context.Context, vaultID, query string, limit int) ([]domain.Hit, error)
}

// VaultRegistry is the list of vaults the user has added. It is application
// state: not derivable from any vault, and not stored in one.
type VaultRegistry interface {
	List() ([]domain.Vault, error)
	Add(v domain.Vault) error
	Find(nameOrPath string) (domain.Vault, bool, error)
}

package version

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Statements is every statement of the package by name, for the query plan test.
func Statements() map[string]string { return sqlfile.Load(files, "sql").All() }

// FindContentHash is the hash the sources table holds for the file at the path,
// with the stat it was taken at, and false where the table has no row of it.
func (r *Repository) FindContentHash(ctx context.Context, vaultID domain.VaultID, path string) (textport.ContentHash, bool, error) {
	return textport.ContentHash{}, false, errors.New("not implemented")
}

// SaveContentHash records the hash of a file on its row of the sources table,
// where the row holds the record's size and modification time or already holds
// its hash. A file with no row is not recorded.
func (r *Repository) SaveContentHash(ctx context.Context, vaultID domain.VaultID, record textport.ContentHash) error {
	return errors.New("not implemented")
}

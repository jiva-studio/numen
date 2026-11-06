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

// FindFileHash is the hash the sources table holds for the file at the path,
// with the stat it was taken at, and false where the table has no row of it.
func (r *Repository) FindFileHash(ctx context.Context, vaultID domain.VaultID, path string) (textport.FileHash, bool, error) {
	return textport.FileHash{}, false, errors.New("not implemented")
}

// SaveFileHash records the hash of a file on its row of the sources table,
// where the row holds the record's size and modification time or already holds
// its hash. A file with no row is not recorded.
func (r *Repository) SaveFileHash(ctx context.Context, vaultID domain.VaultID, record textport.FileHash) error {
	return errors.New("not implemented")
}

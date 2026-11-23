// Package indexfile hands a test an index with the migrations already run.
//
// Each index is built by opening one, so a migration added later is in it with
// nothing else to do.
package indexfile

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
)

func newMigratedTemplate() *vaulttest.Template {
	return vaulttest.NewTemplate(func(ctx context.Context, path string) error {
		db, err := index.Open(ctx, path, index.DisableSync())
		if err != nil {
			return err
		}
		return db.Close()
	})
}

// GetPath is a migrated index of the test's own, at a path nothing else uses.
func GetPath(tb testing.TB) string {
	tb.Helper()
	return newMigratedTemplate().GetPath(tb)
}

// Package indexfile hands a test an index with the migrations already run.
//
// Each index is built by opening one, so a migration added later is in it with
// nothing else to do.
package indexfile

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testonly"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
)

// SkipFlush is what a test opens an index with. Nothing a test binary writes
// outlives the run, so no index it opens waits for the disk.
func SkipFlush() index.Option { return index.SkipFlush(testonly.NewGrant()) }

func newMigrated() *vaultfixture.Template {
	return vaultfixture.NewTemplate(func(ctx context.Context, path string) error {
		db, err := index.Open(ctx, path, SkipFlush())
		if err != nil {
			return err
		}
		return db.Close()
	})
}

// Path is a migrated index of the test's own, at a path nothing else uses.
func Path(tb testing.TB) string {
	tb.Helper()
	return newMigrated().Path(tb)
}

// Package indexfile hands a test an index with the migrations already run.
//
// The migrations run once for the whole test binary, on a template every test
// is given its own copy of. The template is built by opening an index, so a
// migration added later is in it with nothing else to do.
package indexfile

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testonly"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// Unsynchronised is what a test opens an index with. Nothing a test binary
// writes outlives the run, so no index it opens waits for the disk.
func Unsynchronised() index.Option { return index.Unsynchronised(testonly.NewGrant()) }

var migrated = testsupport.NewTemplate(func(ctx context.Context, path string) error {
	db, err := index.Open(ctx, path, Unsynchronised())
	if err != nil {
		return err
	}
	return db.Close()
})

// Path is a migrated index of the test's own, at a path nothing else uses.
func Path(t testing.TB) string { return migrated.Path(t) }

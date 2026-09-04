package index

import (
	"database/sql/driver"
	"fmt"
	"sync"

	sqlite "modernc.org/sqlite"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// folded is the one registration, and folding what came of it.
var (
	folded  sync.Once
	folding error
)

// foldsNames gives SQL domain.FoldName under a name it can call, so a migration
// rewriting the stored names computes the same key the scan writes.
//
// The driver takes the name for the whole process and hands it to every
// connection opened afterwards, so this runs where a connection is opened and
// not where the package is imported: a program that never opens an index
// registers nothing.
func foldsNames() error {
	folded.Do(func() {
		folding = sqlite.RegisterDeterministicScalarFunction("numen_fold", 1,
			func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				name, ok := args[0].(string)
				if !ok {
					return nil, fmt.Errorf("numen_fold was given %T, which is not a name", args[0])
				}
				return domain.FoldName(name), nil
			})
	})
	return folding
}

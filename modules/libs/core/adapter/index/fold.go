package index

import (
	"database/sql/driver"
	"fmt"

	sqlite "modernc.org/sqlite"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// folding registers domain.FoldName under a name SQL can call, so a migration
// rewriting the stored names computes the same key the scan writes.
//
// The function reaches connections opened after it, which is every connection
// this package opens.
var folding = sqlite.RegisterDeterministicScalarFunction("numen_fold", 1,
	func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("numen_fold was given %T, which is not a name", args[0])
		}
		return domain.FoldName(name), nil
	})

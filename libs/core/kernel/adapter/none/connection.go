package none

import "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

// OperationTitles is the titles of a machine where no plugin titles an operation.
type OperationTitles struct{}

func (OperationTitles) ListOperationTitles() []manifest.Operation { return nil }

package none

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Detected is the detection of a machine where no provider detects connections.
type Detected struct{}

func (Detected) ListDetected() []connection.Connection { return nil }
func (Detected) IsPending(string) bool                 { return false }
func (Detected) Detect(context.Context, string)        {}
func (Detected) Refresh()                              {}

// OperationTitles is the titles of a machine where no plugin titles an operation.
type OperationTitles struct{}

func (OperationTitles) ListOperationTitles() []manifest.Operation { return nil }

package connection

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// ListRowOptions returns the values a row field of a connection can take, as its provider lists them for the
// options of the row and the operations of its setting.
type ListRowOptions func(ctx context.Context, connectionID, field string, options map[string]string, operations []string) ([]modelcall.Option, error)

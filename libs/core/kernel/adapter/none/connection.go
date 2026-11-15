package none

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// OperationTitles is the titles of a machine where no plugin titles an operation.
type OperationTitles struct{}

func (OperationTitles) ListOperationTitles() []manifest.Operation { return nil }

// RerankRows is the Reranking setting of an installation that runs no plugin: it has no row.
type RerankRows struct{}

func (RerankRows) ListRows(string, string) ([]modelcall.Row, error) { return nil, nil }

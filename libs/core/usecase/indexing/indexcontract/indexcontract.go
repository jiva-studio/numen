// Package indexcontract re-exports the index port contract.
package indexcontract

import "github.com/jiva-studio/numen/libs/core/port/indexport"

type (
	ExtensionCount   = indexport.ExtensionCount
	IndexCounts      = indexport.IndexCounts
	IndexMaintenance = indexport.IndexMaintenance
	IndexProgress    = indexport.IndexProgress
	IndexSizes       = indexport.IndexSizes
	IndexStats       = indexport.IndexStats
)

package searchquery

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// QuerySource is a search source that makes the vector of a query as it does for its own search, under
// its description.
type QuerySource interface {
	// Describe is the description every vector this source returns belongs to.
	Describe() searchsource.Description
	searchsourceport.QueryEmbedder
}

// Package index holds the counts and sizes the search index reports.
package index

// ExtensionCount is the sources of one file extension and the chunks that can carry a vector among
// the chunks they hold.
type ExtensionCount struct {
	Extension string
	Sources   int64
	Chunks    int64
}

// Counts is what the index holds for one vault under one recipe.
type Counts struct {
	Chunks     int64
	Vectors    int64
	Sources    int64
	Extensions []ExtensionCount
}

// Sizes is the space the index takes on disk, in bytes.
type Sizes struct {
	Database int64
	Vectors  int64
}

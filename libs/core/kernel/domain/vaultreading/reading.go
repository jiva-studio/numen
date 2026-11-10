// Package vaultreading holds what the three passes that make a vault searchable report: the scan of its notes,
// the indexing of its file texts and the embedding of its chunks.
package vaultreading

// ScanResult reports what a scan did, in the terms the user cares about.
type ScanResult struct {
	Notes      int
	Assets     int
	Indexed    int
	Unchanged  int
	Removed    int
	Vanished   int
	Unreadable int
}

// IndexResult reports what indexing did.
type IndexResult struct {
	Seen          int
	Recorded      int
	Unchanged     int
	Extracted     int
	Chunks        int
	Unreadable    int
	Removed       int
	Outdated      int
	Vanished      int
	CurrentSource string
}

// EmbedResult reports what embedding did.
type EmbedResult struct {
	Pending       int
	Embedded      int
	Reused        int
	Vanished      int
	Displaced     int
	Skipped       int
	CurrentSource string
	Active        []string
	IsBusy        bool
}

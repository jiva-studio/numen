package index

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

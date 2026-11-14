package books

// Baseline names the bytes a book was read from: their size and their modification time in
// nanoseconds. A time of zero names no time.
type Baseline struct {
	Size  int64
	Mtime int64
}

package plugin

// DataSize is how many files a plugin's folder holds, and their bytes.
type DataSize struct {
	FileCount int
	ByteCount int64
}

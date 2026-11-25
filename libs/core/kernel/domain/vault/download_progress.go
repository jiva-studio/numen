package vault

// DownloadProgress is how far a download has come. Without a running download it is not known and the
// sizes are zero.
type DownloadProgress struct {
	IsKnown    bool
	BytesDone  int64
	BytesTotal int64
}

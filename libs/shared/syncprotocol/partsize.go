package syncprotocol

const partSizeStep = 1 << 20

// GetPartSize returns the part size of a multipart upload of the stored size: DefaultPartSize while it gives at
// most MaxParts parts, else the least multiple of 1 MiB that does. A size the protocol never uploads in parts gets
// DefaultPartSize.
func GetPartSize(storedSize int64) int64 {
	if storedSize <= 0 || storedSize > MaxMultipartSize {
		return DefaultPartSize
	}
	if (storedSize+DefaultPartSize-1)/DefaultPartSize <= MaxParts {
		return DefaultPartSize
	}
	least := (storedSize + MaxParts - 1) / MaxParts
	return (least + partSizeStep - 1) / partSizeStep * partSizeStep
}

package note

const (
	KindText   = "text"
	KindBinary = "binary"

	BigFileLimit       = 64 << 20
	MassDeleteRatio    = 30
	MassDeleteMinFiles = 5
	BatchFiles         = 100
	MovedBatchFiles    = 5000
)

// IsSyncable reports whether a file at the path of the vault goes to the sync server.
func IsSyncable(filePath string) bool {
	return false
}

// Kind returns "markdown", "text" or "binary" for the extension of the path.
func Kind(filePath string) string {
	return ""
}

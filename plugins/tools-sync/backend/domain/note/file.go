package note

// FileRecord is what this device knows of a synced note: the file on the server, where it is, the hash both
// computers last agreed on and the revision that is its head.
type FileRecord struct {
	FileID   []byte `json:"fileId"`
	Path     string `json:"path"`
	BaseHash string `json:"baseHash"`
	HeadRev  []byte `json:"headRev"`
}

// Seen is a note as the last look at the vault found it.
type Seen struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"mtime"`
}

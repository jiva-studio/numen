package note

// FileRecord is what this device knows of a synced file: the file on the server, where it is, the hash both
// computers last agreed on, the revision that is its head and what the head's blob is, so that a move can be
// sent without reading the file. SealedKey is the revision key of the head, sealed with the vault key.
type FileRecord struct {
	FileID     []byte `json:"fileId"`
	Path       string `json:"path"`
	BaseHash   string `json:"baseHash"`
	HeadRev    []byte `json:"headRev"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"mtime"`
	Kind       string `json:"kind"`
	BlobID     []byte `json:"blobId"`
	BlobHash   []byte `json:"blobHash"`
	BlobSize   uint64 `json:"blobSize"`
	SealedKey  []byte `json:"revKey"`
}

// Seen is a file as the last look at the vault found it.
type Seen struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"mtime"`
	SourceID   int64  `json:"sourceId"`
}

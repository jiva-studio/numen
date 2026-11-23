package note

// Seen is a file as the last look at the vault found it.
type Seen struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"mtime"`
	SourceID   int64  `json:"sourceId"`
}

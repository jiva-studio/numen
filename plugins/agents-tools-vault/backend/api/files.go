package api

type FileReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault root"`
	Start  int    `json:"start,omitempty" jsonschema:"byte offset to start reading from"`
	Length int    `json:"length,omitempty" jsonschema:"how many bytes to read, at most 16000; 16000 when left out"`
}

type FileReadOutput struct {
	Path        string `json:"path"`
	Text        string `json:"text" jsonschema:"the bytes read as text"`
	Start       int    `json:"start" jsonschema:"starting byte offset of the returned slice"`
	Length      int    `json:"length" jsonschema:"number of bytes in the returned slice"`
	Size        int    `json:"size" jsonschema:"total size of the file in bytes"`
	Fingerprint string `json:"fingerprint" jsonschema:"file fingerprint"`
}

type FileCreateInput struct {
	Path string `json:"path" jsonschema:"the file's path relative to the vault root"`
	Text string `json:"text" jsonschema:"raw content to write to the new file"`
}

type FileCreateOutput struct {
	Path string `json:"path" jsonschema:"the path of the created file, relative to the vault root"`
	Size int    `json:"size" jsonschema:"the size of the created file in bytes"`
}

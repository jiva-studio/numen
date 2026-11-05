package service

// The key and bounds of the setting that names the longest chunk of a note, in tokens. They repeat
// the setting in manifest.json.
const (
	settingChunkTokens = "chunkTokens"
	defaultChunkTokens = 256
	minChunkTokens     = 16
	maxChunkTokens     = 8192
)

// ReadChunkTokens is the requested chunk size: the default for a value that is no number, the
// whole part of a fraction, and the nearest bound for a number outside the bounds.
func ReadChunkTokens(_ map[string]string) int {
	return 0
}

// ClampChunkTokens is the requested size limited to the model's own limit. A limit of zero or less
// is a model that named none, and holds nothing.
func ClampChunkTokens(_, _ int) int {
	return 0
}

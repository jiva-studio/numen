package note

import (
	"path"
	"slices"
	"strings"
)

const (
	KindText   = "text"
	KindBinary = "binary"

	BigFileLimit       = 64 << 20
	MassDeleteRatio    = 30
	MassDeleteMinFiles = 5
	BatchFiles         = 100
	MovedBatchFiles    = 5000

	maxPathBytes = 1000
)

var (
	skippedFolders = []string{".numen", ".git", ".trash"}
	skippedNames   = []string{".ds_store", "thumbs.db", "desktop.ini"}
	skippedEnds    = []string{".part", ".tmp", "~"}
	textExtensions = []string{".txt", ".json", ".csv", ".xml", ".html", ".svg", ".yaml", ".yml", ".toml", ".log"}
)

// IsSyncable reports whether a file at the path of the vault goes to the sync server.
func IsSyncable(filePath string) bool {
	if len(filePath) > maxPathBytes {
		return false
	}
	segments := strings.Split(filePath, "/")
	for _, folder := range segments[:len(segments)-1] {
		if slices.Contains(skippedFolders, folder) {
			return false
		}
	}
	name := segments[len(segments)-1]
	lower := strings.ToLower(name)
	if slices.Contains(skippedFolders, name) || slices.Contains(skippedNames, lower) || strings.HasPrefix(name, "~$") {
		return false
	}
	return !slices.ContainsFunc(skippedEnds, func(end string) bool { return strings.HasSuffix(lower, end) })
}

// Kind returns "markdown", "text" or "binary" for the extension of the path.
func Kind(filePath string) string {
	extension := strings.ToLower(path.Ext(filePath))
	switch {
	case extension == ".md":
		return KindMarkdown
	case slices.Contains(textExtensions, extension):
		return KindText
	}
	return KindBinary
}

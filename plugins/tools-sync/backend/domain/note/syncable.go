package note

import (
	"encoding/hex"
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

// IsSyncable reports whether a file at the path of the vault goes to the sync server.
func IsSyncable(filePath string) bool {
	if len(filePath) > maxPathBytes {
		return false
	}
	segments := strings.Split(filePath, "/")
	if slices.ContainsFunc(segments[:len(segments)-1], isSkippedFolder) {
		return false
	}
	name := segments[len(segments)-1]
	lower := strings.ToLower(name)
	if isSkippedFolder(name) || isSkippedName(lower) || strings.HasPrefix(name, "~$") {
		return false
	}
	return !hasSkippedEnd(lower)
}

// GetKind returns "markdown", "text" or "binary" for the extension of the path.
func GetKind(filePath string) string {
	extension := strings.ToLower(path.Ext(filePath))
	switch {
	case extension == ".md":
		return KindMarkdown
	case isTextExtension(extension):
		return KindText
	}
	return KindBinary
}

// SmallFileLimit is the size up to which a file is read whole into memory and sent or placed in one call.
const SmallFileLimit = 4 << 20

// RecordKeyContext is the context the revision key of a record is sealed under.
const RecordKeyContext = "numen-sync/record-key/v1"

// GetRecordKeyContext returns the context the revision key of the file is sealed under in files.json.
func GetRecordKeyContext(remoteID, fileID []byte) []byte {
	return slices.Concat([]byte(RecordKeyContext), remoteID, fileID)
}

// GetStoredName returns the name of the staged file of a download in the store folder.
func GetStoredName(fileID []byte) string { return hex.EncodeToString(fileID) + ".part" }

func isSkippedFolder(name string) bool {
	return slices.Contains([]string{".numen", ".git", ".trash"}, strings.ToLower(name))
}

func isSkippedName(lower string) bool {
	return slices.Contains([]string{".ds_store", "thumbs.db", "desktop.ini"}, lower)
}

func hasSkippedEnd(lower string) bool {
	return slices.ContainsFunc([]string{".part", ".tmp", "~"}, func(end string) bool { return strings.HasSuffix(lower, end) })
}

func isTextExtension(extension string) bool {
	return slices.Contains([]string{".txt", ".json", ".csv", ".xml", ".html", ".svg", ".yaml", ".yml", ".toml", ".log"}, extension)
}

const fetchPrefix = "fetch-"

// GetFetchPartName returns the name of the file in the store folder that keeps the bytes of a blob a fetch
// brought down so far.
func GetFetchPartName(blobID []byte) string {
	return fetchPrefix + hex.EncodeToString(blobID) + ".part"
}

// GetFetchPlainName returns the name of the file in the store folder that holds the opened bytes of a file a
// fetch is about to place.
func GetFetchPlainName(fileID []byte) string {
	return fetchPrefix + hex.EncodeToString(fileID) + ".bin"
}

// IsFetchName reports whether the name in the store folder is one of a fetch.
func IsFetchName(name string) bool { return strings.HasPrefix(name, fetchPrefix) }

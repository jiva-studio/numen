package domain

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"
)

// TextFormat is the shape of a text version's places. The core reads three, and
// every plugin that gives text writes one of them.
type TextFormat string

const (
	// TextFormatPlain is text with no places: an article.
	TextFormatPlain TextFormat = "plain"
	// TextFormatTimed is text whose parts stand at moments of a recording: a
	// transcript, subtitles.
	TextFormatTimed TextFormat = "timed"
	// TextFormatPositioned is text whose parts stand in rectangles on pages: a
	// recognition, the text layer of a PDF.
	TextFormatPositioned TextFormat = "positioned"
)

// TextFormats is every format the core reads.
func TextFormats() []TextFormat {
	return []TextFormat{TextFormatPlain, TextFormatTimed, TextFormatPositioned}
}

// IsValid reports whether the format is one the core reads.
func (f TextFormat) IsValid() bool { return slices.Contains(TextFormats(), f) }

// TextVersionID names a text version among the versions of one file. It is the
// plugin that made the version and the key that plugin gave it.
type TextVersionID string

// NewTextVersionID is the identity of the version a plugin keeps under a key.
func NewTextVersionID(pluginID, key string) TextVersionID {
	return TextVersionID(pluginID + "/" + key)
}

// A TextVersion is one text of a vault file: who made it, from what, in which
// format, and from which input. It is the core's record; the text itself is
// kept by the plugin that made it.
//
// Input is empty for a version made from the file itself, and names another
// version of the same file otherwise. InputHash is HashContent of the file for
// the first, and the Hash of the input version for the second, as both were
// when the version was made. Hash is what the version's own text is now.
type TextVersion struct {
	Path       string
	PluginID   string
	Key        string
	Input      TextVersionID
	Format     TextFormat
	InputHash  string
	Hash       string
	CreateTime time.Time
}

// GetID is the identity of the version among the versions of its file.
func (v TextVersion) GetID() TextVersionID { return NewTextVersionID(v.PluginID, v.Key) }

// IsFromFile reports whether the version was made from the file itself.
func (v TextVersion) IsFromFile() bool { return v.Input == "" }

// HashContent is the hash a file's input is compared by: SHA-256 of its bytes,
// in lowercase hex.
func HashContent(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// IsLiveVersion reports whether a version is live: every version of its chain,
// down to the file, was made by an enabled plugin, and every input is still
// what it was when the version above it was made.
//
// fileHash is HashContent of the file as it is now.
func IsLiveVersion(
	versions []TextVersion, id TextVersionID, fileHash string, isEnabled func(pluginID string) bool,
) bool {
	byID := make(map[TextVersionID]TextVersion, len(versions))
	for _, v := range versions {
		byID[v.GetID()] = v
	}
	seen := map[TextVersionID]bool{}
	for {
		v, ok := byID[id]
		if !ok || seen[id] || !isEnabled(v.PluginID) {
			return false
		}
		seen[id] = true
		if v.IsFromFile() {
			return v.InputHash == fileHash
		}
		input, ok := byID[v.Input]
		if !ok || input.Hash != v.InputHash {
			return false
		}
		id = v.Input
	}
}

// FindLiveVersion is the newest live version of a file, and false where none is
// live. Newer is a later CreateTime; of two made at one instant the later in
// the list is newer.
func FindLiveVersion(
	versions []TextVersion, fileHash string, isEnabled func(pluginID string) bool,
) (TextVersion, bool) {
	order := make([]int, len(versions))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		if c := versions[b].CreateTime.Compare(versions[a].CreateTime); c != 0 {
			return c
		}
		return cmp.Compare(b, a)
	})
	for _, i := range order {
		if IsLiveVersion(versions, versions[i].GetID(), fileHash, isEnabled) {
			return versions[i], true
		}
	}
	return TextVersion{}, false
}

// HasPluginInChain reports whether the plugin made the version or any version
// it was made from.
func HasPluginInChain(versions []TextVersion, id TextVersionID, pluginID string) bool {
	byID := make(map[TextVersionID]TextVersion, len(versions))
	for _, v := range versions {
		byID[v.GetID()] = v
	}
	seen := map[TextVersionID]bool{}
	for {
		v, ok := byID[id]
		if !ok || seen[id] {
			return false
		}
		if v.PluginID == pluginID {
			return true
		}
		seen[id] = true
		id = v.Input
	}
}

// FileInput is the files a plugin makes text from, by extension with its dot.
type FileInput []string

// IsFileAccepted reports whether a file of the path is one the input names.
// Extensions compare without regard to case.
func (in FileInput) IsFileAccepted(path string) bool {
	dot := strings.LastIndexByte(path, '.')
	if dot < 0 || strings.LastIndexByte(path, '/') > dot {
		return false
	}
	ext := strings.ToLower(path[dot:])
	return slices.ContainsFunc(in, func(one string) bool { return strings.ToLower(one) == ext })
}

package domain

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"slices"
	"strings"
	"time"
)

// TextKind is what a text version is, as the plugin that makes it names it.
// The core compares it as an opaque string.
type TextKind string

// TextVersionID names a text version among the versions of one file. It is the
// plugin that made the version and the key that plugin gave it.
type TextVersionID string

// NewTextVersionID is the identity of the version a plugin keeps under a key.
func NewTextVersionID(pluginID, key string) TextVersionID {
	return TextVersionID(pluginID + "/" + key)
}

// A TextVersion is the core's record of one text of a vault file: who made it,
// from what, of which kind, and the file in the maker's plugin folder that
// holds its segments.
//
// Source is empty for a version made from the file itself; for any other
// version it names another version of the same file. InputHash is HashContent of the file for
// the first, and the TextHash of the source version for the second, as both were
// when the version was made. File is the name of the version's file in the
// folder of PluginID; TextHash is HashContent of that file's bytes when the version
// was recorded.
type TextVersion struct {
	Path       string
	PluginID   string
	Key        string
	Source     TextVersionID
	Kind       TextKind
	File       string
	InputHash  string
	TextHash   string
	CreateTime time.Time
}

// GetID is the identity of the version among the versions of its file.
func (v TextVersion) GetID() TextVersionID { return NewTextVersionID(v.PluginID, v.Key) }

// IsFromFile reports whether the version was made from the file itself.
func (v TextVersion) IsFromFile() bool { return v.Source == "" }

// HashContent is the hash a file's content is compared by: SHA-256 of its
// bytes, in lowercase hex.
func HashContent(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// HashReader is HashContent of the bytes the reader gives, read as a stream.
func HashReader(r io.Reader) (string, error) {
	sum := sha256.New()
	if _, err := io.Copy(sum, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// IsChainValid reports whether every source in a version's chain, down to the
// file, is still what it was when the version above it was made.
//
// contentHash is HashContent of the file as it is now.
func IsChainValid(versions []TextVersion, id TextVersionID, contentHash string) bool {
	byID := make(map[TextVersionID]TextVersion, len(versions))
	for _, v := range versions {
		byID[v.GetID()] = v
	}
	visited := map[TextVersionID]bool{}
	for {
		v, ok := byID[id]
		if !ok || visited[id] {
			return false
		}
		visited[id] = true
		if v.IsFromFile() {
			return v.InputHash == contentHash
		}
		source, ok := byID[v.Source]
		// gomutants:disable-next-line EXPRESSION_REMOVE reason="a missing source either mismatches a non-empty InputHash here or fails the byID lookup on the next iteration"
		if !ok || source.TextHash != v.InputHash {
			return false
		}
		id = v.Source
	}
}

// FindLiveVersion is the newest version of a file whose chain is valid and
// whose own file isIntact reports unchanged, and false where none is. Newer is
// a later CreateTime; of two made at one instant the later in the list is
// newer. The plugin that made a version need not be enabled.
func FindLiveVersion(
	versions []TextVersion, contentHash string, isIntact func(TextVersion) bool,
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
		if IsChainValid(versions, versions[i].GetID(), contentHash) && isIntact(versions[i]) {
			return versions[i], true
		}
	}
	return TextVersion{}, false
}

// FileInput is the files a plugin makes text from, by extension with its dot.
type FileInput []string

// IsFileAccepted reports whether a file of the path is one the input names.
// Extensions compare without regard to case.
func (in FileInput) IsFileAccepted(path string) bool {
	dot := strings.LastIndexByte(path, '.')
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="a slash and a dot cannot occupy the same positive byte index"
	if dot < 0 || strings.LastIndexByte(path, '/') > dot {
		return false
	}
	ext := strings.ToLower(path[dot:])
	return slices.ContainsFunc(in, func(one string) bool { return strings.ToLower(one) == ext })
}

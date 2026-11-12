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

// TextSourceProperty is a frontmatter property of a note that a text version
// was made from: the key and its string value. The zero value is no property.
type TextSourceProperty struct {
	Key   string
	Value string
}

// A TextVersion is the core record of one text of a vault file: who made it,
// from what, of which kind, and the file in the maker folder.
// A version has one root: Source (InputHash is its TextHash), SourceProperty
// (InputHash empty), or neither (the file; InputHash is HashContent of the file).
// TextHash is HashContent of File bytes. Hashes are as of when the version was made.
type TextVersion struct {
	Path           string
	PluginID       string
	Key            string
	Source         TextVersionID
	SourceProperty TextSourceProperty
	Kind           TextKind
	File           string
	InputHash      string
	TextHash       string
	CreateTime     time.Time
}

// GetID is the identity of the version among the versions of its file.
func (v TextVersion) GetID() TextVersionID { return NewTextVersionID(v.PluginID, v.Key) }

// IsFromFile reports whether the version was made from the file itself.
func (v TextVersion) IsFromFile() bool { return v.Source == "" && !v.IsFromProperty() }

// IsFromProperty reports whether the version was made from a property of the note.
func (v TextVersion) IsFromProperty() bool { return v.SourceProperty.Key != "" }

func (p TextSourceProperty) isPropertyHeld(properties map[string]any) bool {
	value, isText := properties[p.Key].(string)
	return isText && value == p.Value
}

// HashContent is the hash a file's content is compared by: SHA-256 of its
// bytes, in lowercase hex.
func HashContent(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// HashReader returns HashContent of the bytes of the reader, read as a stream.
func HashReader(r io.Reader) (string, error) {
	sum := sha256.New()
	if _, err := io.Copy(sum, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// IsChainValid reports whether every source in a version's chain, down to its
// root, is still what it was when the version above it was made.
//
// contentHash is HashContent of the file as it is now, and properties is the
// note's frontmatter as it is now.
func IsChainValid(versions []TextVersion, id TextVersionID, contentHash string, properties map[string]any) bool {
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
		if v.IsFromProperty() {
			return v.SourceProperty.isPropertyHeld(properties)
		}
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
	versions []TextVersion, contentHash string, properties map[string]any, isIntact func(TextVersion) bool,
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
		if IsChainValid(versions, versions[i].GetID(), contentHash, properties) && isIntact(versions[i]) {
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

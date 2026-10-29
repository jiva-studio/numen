package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/textfile"
)

var (
	// ErrInvalidVersion is what a put gets for a version with no key, no kind or no file, a file that
	// is missing or is not JSON Lines of segments, or a source that is no other version of the file.
	ErrInvalidVersion = errors.New("host: the version is not valid")
	// ErrNotOwner is what a call gets for a version of another plugin, or from a plugin that declares
	// no text input.
	ErrNotOwner = errors.New("host: the version is another plugin's")
)

// TextPut is what a plugin reports of one of its text versions. The plugin writes the version's
// file into its folder first, with WriteText; File is that file's name. Source names the version the
// text was made from, and is empty for the file itself. SourceHash is HashContent of the
// file for the first, and the Hash of the source version for the second. Kind is what the text is,
// as the plugin names it ("transcript", "recognition"). The host records the hash of the file.
type TextPut struct {
	Path       string
	Key        string
	Source     TextVersionID
	Kind       TextKind
	SourceHash string
	File       string
}

// TextListing is the versions of a file and the one the host shows, which is empty where none is live.
type TextListing struct {
	Versions []TextVersion
	Live     TextVersionID
}

// TextPage is one page of a version's segments. An empty NextCursor is the last page.
type TextPage struct {
	Segments   []TextSegment
	NextCursor string
}

// Texts is the text versions of the vault the host shows, as a plugin works them. TextClient works
// them through HostTextService; tests work them through hosttest.Texts.
type Texts interface {
	// With is the text versions of the vault of the identity. A call on it returns ErrUnknownVault
	// when the installation knows no such vault.
	With(id VaultID) Texts

	// PutVersion records a version of the plugin from the file it wrote, or replaces the one it has
	// under the key, and reindexes the file.
	PutVersion(ctx context.Context, params TextPut) (TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, path, key string) error
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, path string) (TextListing, error)
	// ReadText is one page of any version's segments, read by the host from the version's file, or
	// ErrNotExist.
	ReadText(ctx context.Context, path string, id TextVersionID, cursor string) (TextPage, error)
}

// errRepeatedCursor is a text that returned a page with the cursor it was asked with.
var errRepeatedCursor = errors.New("host: the same page was returned twice")

// ReadWholeText is every page of a version's segments, joined.
func ReadWholeText(ctx context.Context, texts Texts, version TextVersion) (VersionText, error) {
	var segments []TextSegment
	cursor := ""
	for {
		page, err := texts.ReadText(ctx, version.Path, version.GetID(), cursor)
		if err != nil {
			return VersionText{}, err
		}
		segments = append(segments, page.Segments...)
		if page.NextCursor == "" {
			return NewVersionText(version, segments), nil
		}
		if page.NextCursor == cursor {
			return VersionText{}, fmt.Errorf("read %s of %s: %w", version.GetID(), version.Path, errRepeatedCursor)
		}
		cursor = page.NextCursor
	}
}

// DeleteText removes a text version the plugin of the id made: its record, then its file in the
// plugin's folder. A version that is not there is ErrNotExist, and one of another plugin is ErrNotOwner.
func DeleteText(ctx context.Context, texts Texts, store Store, pluginID, path string, id TextVersionID) error {
	listing, err := texts.ListVersions(ctx, path)
	if err != nil {
		return err
	}
	for _, version := range listing.Versions {
		if version.GetID() != id {
			continue
		}
		if version.PluginID != pluginID {
			return fmt.Errorf("delete %s of %s: %w", id, path, ErrNotOwner)
		}
		if err := texts.RemoveVersion(ctx, path, version.Key); err != nil {
			return err
		}
		return store.Delete(ctx, version.File)
	}
	return fmt.Errorf("delete %s of %s: %w", id, path, ErrNotExist)
}

// WriteText writes segments as a version file of the plugin's folder, replacing the file
// atomically. The plugin puts the version after the write.
func WriteText(ctx context.Context, store Store, name string, segments []TextSegment) error {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return err
	}
	return store.Write(ctx, name, raw)
}

// AppendText adds segments to the end of a version file of the plugin's folder, making the file
// where it is not there.
func AppendText(ctx context.Context, store Store, name string, segments []TextSegment) error {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return err
	}
	return store.Append(ctx, name, raw)
}

// ReadTextFile reads the segments of a version file of the plugin's folder.
func ReadTextFile(ctx context.Context, store Store, name string) ([]TextSegment, error) {
	raw, err := store.Read(ctx, name)
	if err != nil {
		return nil, err
	}
	return textfile.Decode(raw)
}

// Selector is the keys of a segment's selector that the SDK documents. A selector is a JSON object
// (W3C Web Annotation selector) that locates a segment in the file its text was made from. The
// core stores it as is and returns it with search hits; the view that opens the file reads it.
// Each key and its unit come from a standard:
//
//   - t: [start, end] in seconds from the start of a recording (W3C Media Fragments).
//   - page: a page of a document, counted from 1 (RFC 8118).
//   - cfi: an EPUB Canonical Fragment Identifier.
//   - boxes: rectangles on the page, each [minX, minY, maxX, maxY] in fractions of the page.
//
// A plugin may add keys of its own, named with its plugin id as a prefix.
type Selector struct {
	T     []float64    `json:"t,omitempty"`
	Page  int          `json:"page,omitempty"`
	CFI   string       `json:"cfi,omitempty"`
	Boxes [][4]float32 `json:"boxes,omitempty"`
}

// Encode is the selector as the JSON a segment carries.
func (s Selector) Encode() RawSelector {
	// A struct of numbers, strings and slices of them always marshals.
	raw, _ := json.Marshal(s)
	return RawSelector(raw)
}

// ParseSelector reads the documented keys of a segment's selector. An empty selector has none.
func ParseSelector(selector RawSelector) (Selector, error) {
	var out Selector
	if selector == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(selector), &out); err != nil {
		return Selector{}, fmt.Errorf("host: read selector: %w", err)
	}
	return out, nil
}

// GetBoxes returns the rectangles of the segments a run of a version's text overlaps, in the order
// of the segments. Each box carries the page its selector names, counted from zero, and the run of
// the segment it covers. A segment whose selector names no page has no boxes.
func GetBoxes(text VersionText, run ByteRange) []HighlightBox {
	if run.Empty() {
		return nil
	}
	var boxes []HighlightBox
	from := 0
	for _, segment := range text.GetSegments() {
		to := from + len(segment.Text)
		if to > run.Start && from < run.End {
			if selector, err := ParseSelector(segment.Selector); err == nil && selector.Page > 0 {
				for _, box := range selector.Boxes {
					boxes = append(boxes, HighlightBox{
						Page: selector.Page - 1, ByteRange: ByteRange{Start: from, End: to},
						Rect: Rect{MinX: box[0], MinY: box[1], MaxX: box[2], MaxY: box[3]},
					})
				}
			}
		}
		from = to
	}
	return boxes
}

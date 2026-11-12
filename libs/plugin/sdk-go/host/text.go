package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/textfile"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var (
	// ErrInvalidVersion is what a put gets for a version with no key, no kind or no file, a file that
	// is missing or is not JSON Lines of segments, a source that is no other version of the file, or a
	// source property given with a source or a source hash.
	ErrInvalidVersion = errors.New("host: the version is not valid")
	// ErrNotOwner is what a call gets for a version of another plugin, or from a plugin that is not
	// enabled.
	ErrNotOwner = errors.New("host: the version is not the plugin's")
)

// TextPut is what a plugin reports of one of its text versions.
// The plugin writes the file into its folder first, with WriteText; File is its name.
// A version has one root: Source (InputHash is its TextHash), SourceProperty (no
// Source or InputHash; live while the note holds the property's key as that string),
// or neither (InputHash is HashContent of the file). The host records the file's hash.
type TextPut struct {
	Path           string
	Key            string
	Source         domain.TextVersionID
	SourceProperty domain.TextSourceProperty
	Kind           domain.TextKind
	InputHash      string
	File           string
}

// TextListing is the versions of a file and the one the host shows, which is empty where none is live.
type TextListing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// TextRange is a ranged read of a version: the segments that cover the range, each truncated to it, where
// the first begins in the whole text, and the bytes of the whole text.
type TextRange struct {
	Segments []domain.TextSegment
	Start    int
	Size     int
}

// TextPage is one page of a version's segments. An empty NextCursor is the last page.
type TextPage struct {
	Segments   []domain.TextSegment
	NextCursor string
}

// TextVersions records the text versions of a plugin in the vault the host shows. TextClient records
// them through HostTextService; tests record them through hosttest.Texts. A plugin that works another
// vault than the shown one calls With on TextClient and keeps the narrow interface of the calls it
// needs on the result.
type TextVersions interface {
	// PutVersion records a version of the plugin from the file it wrote, or replaces the one it has
	// under the key, and reindexes the file.
	PutVersion(ctx context.Context, params TextPut) (domain.TextVersion, error)
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, path string) (TextListing, error)
}

// TextReader reads the segments of the text versions of the vault.
type TextReader interface {
	// ReadText is one page of any version's segments, read by the host from the version's file, or
	// ErrNotExist.
	ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (TextPage, error)
	// ReadTextRange returns the segments of a version that cover length bytes of its whole text from start,
	// truncated at whole characters. A length of zero reads to the end.
	ReadTextRange(ctx context.Context, path string, id domain.TextVersionID, start, length int) (TextRange, error)
}

// errRepeatedCursor is a text that returned a page with the cursor it was requested with.
var errRepeatedCursor = errors.New("host: the same page was returned twice")

// ReadWholeText is every page of a version's segments, joined.
func ReadWholeText(ctx context.Context, texts TextReader, version domain.TextVersion) (domain.VersionText, error) {
	var segments []domain.TextSegment
	cursor := ""
	for {
		page, err := texts.ReadText(ctx, version.Path, version.GetID(), cursor)
		if err != nil {
			return domain.VersionText{}, err
		}
		segments = append(segments, page.Segments...)
		if page.NextCursor == "" {
			return domain.NewVersionText(version, segments), nil
		}
		if page.NextCursor == cursor {
			return domain.VersionText{}, fmt.Errorf("read %s of %s: %w", version.GetID(), version.Path, errRepeatedCursor)
		}
		cursor = page.NextCursor
	}
}

// WriteText writes segments as a version file of the plugin's folder, replacing the file
// atomically. The plugin puts the version after the write.
func WriteText(ctx context.Context, store StoreWriter, name string, segments []domain.TextSegment) error {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return err
	}
	return store.Write(ctx, name, raw)
}

// AppendText adds segments to the end of a version file of the plugin's folder, making the file
// where it is not there.
func AppendText(ctx context.Context, store StoreWriter, name string, segments []domain.TextSegment) error {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return err
	}
	return store.Append(ctx, name, raw)
}


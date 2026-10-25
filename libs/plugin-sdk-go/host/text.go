package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var (
	// ErrInvalidVersion is what a put gets for a version with no key, no format the core reads, or
	// an input that is no other version of the file.
	ErrInvalidVersion = errors.New("host: the version is not valid")
	// ErrNotOwner is what a call gets for a version of another plugin, or from a plugin that declares
	// no text input.
	ErrNotOwner = errors.New("host: the version is another plugin's")
)

// TextPut is what a plugin reports of one of its text versions. Input names the version the text
// was made from, and is empty for the file itself. InputHash is domain.HashContent of the file for
// the first, and the Hash of the input version for the second. Hash is the version's own text now.
type TextPut struct {
	Path      string
	Key       string
	Input     domain.TextVersionID
	Format    domain.TextFormat
	InputHash string
	Hash      string
}

// TextListing is the versions of a file and the one the host shows, which is empty where none is live.
type TextListing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// TextPage is one page of a text. An empty NextCursor is the last page.
type TextPage struct {
	Parts      []domain.TextPart
	NextCursor string
}

// Texts is the text versions of the vault the host shows, as a plugin works them. TextClient works
// them through HostTextService; tests work them through hosttest.Texts.
type Texts interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key, and tells
	// the host the text of the file changed.
	PutVersion(ctx context.Context, put TextPut) (domain.TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, path, key string) error
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, path string) (TextListing, error)
	// ReadText is one page of any version's text, read by the host from the plugin that made it, or
	// ErrNotExist.
	ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (TextPage, error)
}

// errRepeatedCursor is a text that answered a page with the cursor it was asked with.
var errRepeatedCursor = errors.New("host: the same page was answered twice")

// ReadWholeText is every page of a version's text, joined.
func ReadWholeText(ctx context.Context, texts Texts, version domain.TextVersion) (domain.VersionText, error) {
	var parts []domain.TextPart
	cursor := ""
	for {
		page, err := texts.ReadText(ctx, version.Path, version.GetID(), cursor)
		if err != nil {
			return domain.VersionText{}, err
		}
		parts = append(parts, page.Parts...)
		if page.NextCursor == "" {
			return domain.NewVersionText(version, parts), nil
		}
		if page.NextCursor == cursor {
			return domain.VersionText{}, fmt.Errorf("read %s of %s: %w", version.GetID(), version.Path, errRepeatedCursor)
		}
		cursor = page.NextCursor
	}
}

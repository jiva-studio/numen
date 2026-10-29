package textversion

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// GetLiveText is the text of the file's live version, or textport.ErrNoText.
func (u *Versions) GetLiveText(ctx context.Context, v domain.Vault, path, fileHash string) (domain.VersionText, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	live, ok, err := u.findLive(ctx, v, versions, fileHash)
	if err != nil {
		return domain.VersionText{}, err
	}
	if !ok {
		return domain.VersionText{}, fmt.Errorf("%w: %s has no live version", textport.ErrNoText, path)
	}
	return u.readText(ctx, v, live)
}

// GetVersionText is the text of one version, live or not, whether or not the
// plugin that made it is enabled. A version the file has not, or whose file
// changed since it was recorded, is textport.ErrNoText.
func (u *Versions) GetVersionText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID) (domain.VersionText, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	i := indexOf(versions, id)
	if i < 0 {
		return domain.VersionText{}, fmt.Errorf("%w: version %s of %s", textport.ErrNoText, id, path)
	}
	return u.readText(ctx, v, versions[i])
}

// TextPage is one page of a version's segments. An empty NextCursor is the last page.
type TextPage struct {
	Segments   []domain.TextSegment
	NextCursor string
}

// pageSize is how many segments one page of a version's text holds.
const pageSize = 1000

// GetTextPage is one page of a version's segments. cursor is the NextCursor of
// the page before, and empty for the first.
func (u *Versions) GetTextPage(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (TextPage, error) {
	start := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return TextPage{}, fmt.Errorf("%w: cursor %q", textport.ErrNoText, cursor)
		}
		start = n
	}
	text, err := u.GetVersionText(ctx, v, path, id)
	if err != nil {
		return TextPage{}, err
	}
	segments := text.GetSegments()
	if start > len(segments) {
		return TextPage{}, fmt.Errorf("%w: cursor %q", textport.ErrNoText, cursor)
	}
	end := min(start+pageSize, len(segments))
	page := TextPage{Segments: segments[start:end]}
	if end < len(segments) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

// readText reads a version's file. It returns textport.ErrNoText when the file
// is missing or has a hash that differs from the one recorded with the version.
func (u *Versions) readText(ctx context.Context, v domain.Vault, version domain.TextVersion) (domain.VersionText, error) {
	files, err := u.files.Open(v)
	if err != nil {
		return domain.VersionText{}, err
	}
	file, err := files.ReadTextFile(ctx, version.PluginID, version.File)
	if err != nil {
		return domain.VersionText{}, fmt.Errorf("read text %s of %s: %w", version.GetID(), version.Path, err)
	}
	if file.Hash != version.Hash {
		return domain.VersionText{}, fmt.Errorf("%w: the file of %s of %s changed", textport.ErrNoText, version.GetID(), version.Path)
	}
	return domain.NewVersionText(version, file.Segments), nil
}

// Open is the text of one vault, as the index, the search and the highlighting read it.
func (u *Versions) Open(v domain.Vault) (textport.VaultTexts, error) {
	return vaultTexts{versions: u, vault: v}, nil
}

type vaultTexts struct {
	versions *Versions
	vault    domain.Vault
}

func (t vaultTexts) FindLiveVersion(ctx context.Context, path, fileHash string) (domain.TextVersionID, error) {
	listing, err := t.versions.ListVersions(ctx, t.vault, path, fileHash)
	if err != nil {
		return "", err
	}
	if listing.Live == "" {
		return "", fmt.Errorf("%w: %s has no live version", textport.ErrNoText, path)
	}
	return listing.Live, nil
}

func (t vaultTexts) GetLiveText(ctx context.Context, path, fileHash string) (domain.VersionText, error) {
	return t.versions.GetLiveText(ctx, t.vault, path, fileHash)
}

func (t vaultTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	return t.versions.GetVersionText(ctx, t.vault, path, id)
}

func (t vaultTexts) RequestText(ctx context.Context, path, fileHash string) error {
	return t.versions.RequestText(ctx, t.vault, path, fileHash)
}

func (t vaultTexts) IsTextSource(ctx context.Context, path string) (bool, error) {
	return t.versions.IsTextSource(ctx, t.vault, path)
}
